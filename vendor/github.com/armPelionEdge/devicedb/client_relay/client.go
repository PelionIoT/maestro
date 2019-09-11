package client_relay

import (
    "bytes"    
    "context"
    "crypto/tls"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "net/url"
    "time"
    "github.com/armPelionEdge/devicedb/client"
    "github.com/armPelionEdge/devicedb/transport"
)

type Client interface {
    // Execute a batch update in DeviceDB. The context is bound to the
    // request. The bucket should be the name of the devicedb bucket to
    // which this update should be applied.
    Batch(ctx context.Context, bucket string, batch client.Batch) error
    // Get the value of one or more keys in devicedb. The bucket shold be
    // the name of the devicedb bucket to which this update should be applied.
    // The keys array describes which keys should be retrieved. If no error
    // occurs this function will return an array of values corresponding
    // to the keys that were requested. The results array will mirror the
    // keys array. In other words, the ith value in the result is the value
    // for key i. If a key does not exist the value will be nil.
    Get(ctx context.Context, bucket string, keys []string) ([]*client.Entry, error)
    // Get keys matching one or more prefixes. The keys array represents
    // a list of prefixes to query. The resulting iterator will iterate
    // through database values whose key matches one of the specified
    // prefixes
    GetMatches(ctx context.Context, bucket string, keys []string) (EntryIterator, error)
    // Watch for updates to a set of keys or keys matching certain prefixes
    // lastSerial specifies the serial number of the last received update.
    // The update channel that is returned by this function will stream relevant
    // updates to a consumer. If a disconnection happens from the server
    // then this client will push an error to the error channel and attempt
    // to form a new connection until it is successful. If the consumer supplies
    // a context that is cancellable they can cancel the context which will
    // cause both the update and error channels to close. These channels will
    // not close until the context is cancelled even during disconnections
    // from the server. The client must consume all messages from the update
    // and error channels until they are closed to prevent blocking of the watcher 
    // goroutine.
    Watch(ctx context.Context, bucket string, keys []string, prefixes []string, lastSerial uint64) (chan Update, chan error)
}

type Config struct {
    // The server URI is the base URI for the devicedb server
    // An example of this is https://localhost:9000
    ServerURI string
    // Provide a TLS config if you are connecting to a TLS
    // enabled devicedb relay server. You will need to provide
    // the relay CA and server name (the relay ID)
    TLSConfig *tls.Config
    // When a watcher is established by a call to Watch()
    // disconnections may occur while the watcher is still
    // up. This field determines how often the watcher
    // will try to reconnect until a new connection can be
    // established.
    WatchReconnectTimeout time.Duration
}

// Create a new DeviceDB client
func New(config Config) Client {
    if config.WatchReconnectTimeout == 0 {
        config.WatchReconnectTimeout = time.Second
    }

    return &HTTPClient{
        server: config.ServerURI,
        httpClient: &http.Client{ Transport: &http.Transport{ TLSClientConfig: config.TLSConfig } },
        watchReconnectTimeout: config.WatchReconnectTimeout,
    }
}

type HTTPClient struct {
    server string
    httpClient *http.Client
    watchReconnectTimeout time.Duration
}

func (c *HTTPClient) Batch(ctx context.Context, bucket string, batch client.Batch) error {
    var transportBatch transport.TransportUpdateBatch = batch.ToTransportUpdateBatch()    
    url := fmt.Sprintf("/%s/batch", bucket)

    body, err := json.Marshal(transportBatch)

    if err != nil {
        return err
    }

    respBody, err := c.sendRequest(ctx, "POST", url, body)

    if err != nil {
        return err
    }

    respBody.Close()

    return nil
}

func (c *HTTPClient) Get(ctx context.Context, bucket string, keys []string) ([]*client.Entry, error) {
    url := fmt.Sprintf("/%s/values?", bucket)
    body, err := json.Marshal(keys)

    if err != nil {
        return nil, err
    }

    respBody, err := c.sendRequest(ctx, "POST", url, body)

    if err != nil {
        return nil, err
    }

    defer respBody.Close()

    var decoder *json.Decoder = json.NewDecoder(respBody)
    var transportSiblingSets []*transport.TransportSiblingSet

    err = decoder.Decode(&transportSiblingSets)

    if err != nil {
        return nil, err
    }

    if len(transportSiblingSets) != len(keys) {
        return nil, errors.New(fmt.Sprintf("A protocol error occurred: Asked for %d keys but received %d values in the result array", len(keys), len(transportSiblingSets)))
    }

    var entries []*client.Entry = make([]*client.Entry, len(transportSiblingSets))

    for i, _ := range keys {
        if transportSiblingSets[i] == nil {
            continue
        }

        entries[i] = &client.Entry{
            Context: transportSiblingSets[i].Context,
            Siblings: transportSiblingSets[i].Siblings,
        }
    }

    return entries, nil
}

func (c *HTTPClient) GetMatches(ctx context.Context, bucket string, keys []string) (EntryIterator, error) {
    url := fmt.Sprintf("/%s/matches?", bucket)
    body, err := json.Marshal(keys)

    if err != nil {
        return nil, err
    }

    respBody, err := c.sendRequest(ctx, "POST", url, body)

    if err != nil {
        return nil, err
    }

    return &StreamedEntryIterator{ reader: respBody }, nil
}

func (c *HTTPClient) Watch(ctx context.Context, bucket string, keys []string, prefixes []string, lastSerial uint64) (chan Update, chan error) {
    var query url.Values = url.Values{}

    for _, key := range keys {
        query.Add("key", key)
    }

    for _, prefix := range prefixes {
        query.Add("prefix", prefix)
    }

    updates := make(chan Update)
    errorsChan := make(chan error)

    go func() {
        defer func() {
            close(updates)
            close(errorsChan)
        }()

        for {
            reqCtx, cancel := context.WithCancel(ctx)
            url := fmt.Sprintf("/%s/watch?%s&lastSerial=%d", bucket, query.Encode(), lastSerial)
            respBody, err := c.sendRequest(reqCtx, "GET", url, nil)

            if err == nil {
                // establish new iterator and stream updates
                var streamingMissedUpdates bool = true
                var highestMissedSerial uint64
                updateIterator := &StreamedUpdateIterator{ reader: respBody }                

                // stream updates until the response stream
                // is interrupted or an error occurs
                for updateIterator.Next() {
                    update := updateIterator.Update()

                    // the first chunk of updates are sent
                    // out of order (non-increasing serials)
                    // An empty update marks the end of the
                    // initial chunk of updates which are
                    // this client catching up with any missed
                    // updates and the start of receiving
                    // updates with increasing serial numbers
                    if streamingMissedUpdates {
                        if update.IsEmpty() {
                            // marks end of missed updates
                            streamingMissedUpdates = false

                            if lastSerial < highestMissedSerial {
                                lastSerial = highestMissedSerial
                            }

                            // Sends an empty update simply to let
                            // client know about a new stable serial
                            update.LastStableSerial = lastSerial
                            updates <- update

                            continue
                        }

                        if highestMissedSerial < update.Serial {
                            highestMissedSerial = update.Serial
                        }
                    }

                    // All received updates should have a serial number greater than
                    // the lastSerial provided with the exception of an update with
                    // a serial number of 0
                    if update.Serial <= lastSerial && (lastSerial != 0 || update.Serial != 0) {
                        errorsChan <- errors.New("Protocol error")
                        cancel()
                        break
                    } else if !streamingMissedUpdates {
                        lastSerial = update.Serial
                    }

                    // the last stable serial for an update
                    // sent to a consumer is the last serial number
                    // associated with some received update such that
                    // no future update in the future will have a serial
                    // number lower than that.
                    update.LastStableSerial = lastSerial
                    updates <- update
                }

                if updateIterator.Error() != nil {
                    // Only report the error if the context
                    // wasn't canceled. We don't want to send
                    // 'context canceled' errors
                    select {
                    case <-ctx.Done():
                    default:
                        errorsChan <- updateIterator.Error()
                    }
                }
            } else {
                // Only report the error if the context
                // wasn't canceled. We don't want to send
                // 'context canceled' errors
                select {
                case <-ctx.Done():
                default:
                    errorsChan <- err
                }
            }
            
            // stop if the watcher was cancelled or try
            // to re-establish the connection in a moment
            select {
            case <-ctx.Done():
                return
            case <-time.After(c.watchReconnectTimeout):
            }
        }
    }()

    return updates, errorsChan
}

func (c *HTTPClient) sendRequest(ctx context.Context, httpVerb string, endpointURL string, body []byte) (io.ReadCloser, error) {
    u := fmt.Sprintf("%s%s", c.server, endpointURL)
    request, err := http.NewRequest(httpVerb, u, bytes.NewReader(body))

    if err != nil {
        return nil, err
    }

    request = request.WithContext(ctx)

    resp, err := c.httpClient.Do(request)

    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)

        resp.Body.Close()
        
        if err != nil {
            return nil, err
        }
       
        return nil, &client.ErrorStatusCode{ Message: string(errorMessage), StatusCode: resp.StatusCode }
    }

    return resp.Body, nil
}