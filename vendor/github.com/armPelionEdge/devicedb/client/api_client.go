package client
//
 // Copyright (c) 2019 ARM Limited.
 //
 // SPDX-License-Identifier: MIT
 //
 // Permission is hereby granted, free of charge, to any person obtaining a copy
 // of this software and associated documentation files (the "Software"), to
 // deal in the Software without restriction, including without limitation the
 // rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 // sell copies of the Software, and to permit persons to whom the Software is
 // furnished to do so, subject to the following conditions:
 //
 // The above copyright notice and this permission notice shall be included in all
 // copies or substantial portions of the Software.
 //
 // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 // SOFTWARE.
 //


import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "net/http"

    "github.com/armPelionEdge/devicedb/routes"
    . "github.com/armPelionEdge/devicedb/error"
)

type APIClientConfig struct {
    Servers []string
}

type APIClient struct {
    servers []string
    nextServerIndex int
    httpClient *http.Client
}

func New(config APIClientConfig) *APIClient {
    return &APIClient{
        servers: config.Servers,
        nextServerIndex: 0,
        httpClient: &http.Client{ },
    }
}

func (client *APIClient) nextServer() (server string) {
    if len(client.servers) == 0 {
        return
    }

    server = client.servers[client.nextServerIndex]
    client.nextServerIndex = (client.nextServerIndex + 1) % len(client.servers)

    return
}

func (client *APIClient) ClusterOverview(ctx context.Context) (routes.ClusterOverview, error) {
    encodedOverview, err := client.sendRequest(ctx, "GET", "/cluster", nil)

    if err != nil {
        return routes.ClusterOverview{}, err
    }

    var clusterOverview routes.ClusterOverview

    err = json.Unmarshal(encodedOverview, &clusterOverview)

    if err != nil {
        return routes.ClusterOverview{}, err
    }

    return clusterOverview, nil
}

func (client *APIClient) RelayStatus(ctx context.Context, relayID string) (routes.RelayStatus, error) {
    encodedStatus, err := client.sendRequest(ctx, "GET", "/relays/" + relayID, nil)

    if err != nil {
        return routes.RelayStatus{}, err
    }

    var relayStatus routes.RelayStatus

    err = json.Unmarshal(encodedStatus, &relayStatus)

    if err != nil {
        return routes.RelayStatus{}, err
    }

    return relayStatus, nil
}

func (client *APIClient) AddSite(ctx context.Context, siteID string) error {
    _, err := client.sendRequest(ctx, "PUT", "/sites/" + siteID, nil)

    if err != nil {
        return err
    }

    return nil
}

func (client *APIClient) RemoveSite(ctx context.Context, siteID string) error {
    _, err := client.sendRequest(ctx, "DELETE", "/sites/" + siteID, nil)

    if err != nil {
        return err
    }

    return nil
}

func (client *APIClient) AddRelay(ctx context.Context, relayID string) error {
    _, err := client.sendRequest(ctx, "PUT", "/relays/" + relayID, nil)

    if err != nil {
        return err
    }

    return nil
}

func (client *APIClient) MoveRelay(ctx context.Context, relayID string, siteID string) error {
    var relaySettingsPatch routes.RelaySettingsPatch = routes.RelaySettingsPatch{ Site: siteID }

    body, err := json.Marshal(relaySettingsPatch)

    if err != nil {
        return err
    }

    _, err = client.sendRequest(ctx, "PATCH", "/relays/" + relayID, body)

    if err != nil {
        return err
    }

    return nil
}

func (client *APIClient) RemoveRelay(ctx context.Context, relayID string) error {
    _, err := client.sendRequest(ctx, "DELETE", "/relays/" + relayID, nil)

    if err != nil {
        return err
    }

    return nil
}

func (client *APIClient) Batch(ctx context.Context, siteID string, bucket string, batch Batch) (int, int, error) {
    transportUpdateBatch := batch.ToTransportUpdateBatch()
    encodedTransportUpdateBatch, err := json.Marshal(transportUpdateBatch)

    if err != nil {
        return 0, 0, err
    }

    response, err := client.sendRequest(ctx, "POST", fmt.Sprintf("/sites/%s/buckets/%s/batches", siteID, bucket), encodedTransportUpdateBatch)

    if err != nil {
        return 0, 0, err
    }

    var batchResult routes.BatchResult

    err = json.Unmarshal(response, &batchResult)

    if err != nil {
        return 0, 0, err
    }

    if batchResult.Quorum {
        return int(batchResult.Replicas), int(batchResult.NApplied), nil
    }

    return int(batchResult.Replicas), int(batchResult.NApplied), ENoQuorum
}

func (client *APIClient) Get(ctx context.Context, siteID string, bucket string, keys []string) ([]Entry, error) {
    url := fmt.Sprintf("/sites/%s/buckets/%s/keys?", siteID, bucket)

    for i, key := range keys {
        url += "key=" + key

        if i != len(keys) - 1 {
            url += "&"
        }
    }

    encodedAPIEntries, err := client.sendRequest(ctx, "GET", url, nil)

    if err != nil {
        return nil, err
    }

    var apiEntries []routes.APIEntry

    err = json.Unmarshal(encodedAPIEntries, &apiEntries)

    if err != nil {
        return nil, err
    }

    var entries []Entry = make([]Entry, len(apiEntries))

    for i, apiEntry := range apiEntries {
        entries[i] = Entry{
            Context: apiEntry.Context,
            Siblings: apiEntry.Siblings,
        }
    }

    return entries, nil
}

func (client *APIClient) GetMatches(ctx context.Context, siteID string, bucket string, keys []string) (EntryIterator, error) {
    url := fmt.Sprintf("/sites/%s/buckets/%s/keys?", siteID, bucket)

    for i, key := range keys {
        url += "prefix=" + key

        if i != len(keys) - 1 {
            url += "&"
        }
    }

    encodedAPIEntries, err := client.sendRequest(ctx, "GET", url, nil)

    if err != nil {
        return EntryIterator{}, err
    }

    var apiEntries []routes.APIEntry

    err = json.Unmarshal(encodedAPIEntries, &apiEntries)

    if err != nil {
        return EntryIterator{}, err
    }

    var entryIterator EntryIterator = EntryIterator{ currentEntry: -1, entries: make([]iteratorEntry, len(apiEntries)) }

    for i, apiEntry := range apiEntries {
        entryIterator.entries[i] = iteratorEntry{
            key: apiEntry.Key,
            prefix: apiEntry.Prefix,
            entry: Entry{
                Context: apiEntry.Context,
                Siblings: apiEntry.Siblings,
            },
        }
    }

    return entryIterator, nil
}

func (client *APIClient) LogDump(ctx context.Context) (routes.LogDump, error) {
    url := "/log_dump"
    response, err := client.sendRequest(ctx, "GET", url, nil)

    if err != nil {
        return routes.LogDump{}, err
    }

    var logDump routes.LogDump

    if err := json.Unmarshal(response, &logDump); err != nil {
        return routes.LogDump{}, err
    }

    return logDump, nil
}

func (client *APIClient) Snapshot(ctx context.Context) (routes.Snapshot, error) {
    url := "/snapshot"
    response, err := client.sendRequest(ctx, "POST", url, nil)

    if err != nil {
        return routes.Snapshot{}, err
    }

    var snapshot routes.Snapshot

    if err := json.Unmarshal(response, &snapshot); err != nil {
        return routes.Snapshot{}, err
    }

    return snapshot, nil
}

func (client *APIClient) GetSnapshot(ctx context.Context, uuid string) (routes.Snapshot, error) {
    url := "/snapshot/" + uuid
    response, err := client.sendRequest(ctx, "GET", url, nil)

    if err != nil {
        return routes.Snapshot{}, err
    }

    var snapshot routes.Snapshot

    if err := json.Unmarshal(response, &snapshot); err != nil {
        return routes.Snapshot{}, err
    }

    return snapshot, nil
}

func (client *APIClient) DownloadSnapshot(ctx context.Context, uuid string) (io.ReadCloser, error) {
    url := "/snapshot/" + uuid + ".tar"
    response, err := client.sendRequestRaw(ctx, "GET", url, nil)

    if err != nil {
        return nil, err
    }

    return response, nil
}

func (client *APIClient) sendRequestRaw(ctx context.Context, httpVerb string, endpointURL string, body []byte) (io.ReadCloser, error) {
    u := fmt.Sprintf("http://%s%s", client.nextServer(), endpointURL)
    request, err := http.NewRequest(httpVerb, u, bytes.NewReader(body))

    if err != nil {
        return nil, err
    }

    request = request.WithContext(ctx)

    resp, err := client.httpClient.Do(request)

    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)
        
        if err != nil {
            return nil, err
        }
       
        return nil, &ErrorStatusCode{ Message: string(errorMessage), StatusCode: resp.StatusCode }
    }

    return resp.Body, nil
}

func (client *APIClient) sendRequest(ctx context.Context, httpVerb string, endpointURL string, body []byte) ([]byte, error) {
    u := fmt.Sprintf("http://%s%s", client.nextServer(), endpointURL)
    request, err := http.NewRequest(httpVerb, u, bytes.NewReader(body))

    if err != nil {
        return nil, err
    }

    request = request.WithContext(ctx)

    resp, err := client.httpClient.Do(request)

    if err != nil {
        return nil, err
    }
    
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        errorMessage, err := ioutil.ReadAll(resp.Body)
        
        if err != nil {
            return nil, err
        }
       
        return nil, &ErrorStatusCode{ Message: string(errorMessage), StatusCode: resp.StatusCode }
    }

    responseBody, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        return nil, err
    }

    return responseBody, nil
}