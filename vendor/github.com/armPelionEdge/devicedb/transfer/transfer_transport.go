package transfer

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "io"

    . "github.com/armPelionEdge/devicedb/cluster"
)

const DefaultEndpointURL = "/partitions/%d/keys"
var EBadResponse = errors.New("Node responded with a bad response")

type PartitionTransferTransport interface {
    Get(nodeID uint64, partition uint64) (io.Reader, func(), error)
}

type HTTPTransferTransport struct {
    httpClient *http.Client
    configController ClusterConfigController
    endpointURL string
}

func NewHTTPTransferTransport(configController ClusterConfigController, httpClient *http.Client) *HTTPTransferTransport {
    return &HTTPTransferTransport{
        httpClient: httpClient,
        configController: configController,
        endpointURL: DefaultEndpointURL,
    }
}

func (transferTransport *HTTPTransferTransport) SetEndpointURL(endpointURL string) *HTTPTransferTransport {
    transferTransport.endpointURL = endpointURL

    return transferTransport
}

func (transferTransport *HTTPTransferTransport) Get(nodeID uint64, partition uint64) (io.Reader, func(), error) {
    peerAddress := transferTransport.configController.ClusterController().ClusterMemberAddress(nodeID)

    if peerAddress.IsEmpty() {
        return nil, nil, ENoSuchNode
    }

    endpointURL := peerAddress.ToHTTPURL(fmt.Sprintf(transferTransport.endpointURL, partition))
    request, err := http.NewRequest("GET", endpointURL, nil)

    if err != nil {
        return nil, nil, err
    }

    ctx, cancel := context.WithCancel(context.Background())
    request.WithContext(ctx)

    resp, err := transferTransport.httpClient.Do(request)

    if err != nil {
        cancel()

        if resp != nil && resp.Body != nil {
            resp.Body.Close()
        }

        return nil, nil, err
    }

    if resp.StatusCode != http.StatusOK {
        cancel()
        resp.Body.Close()

        return nil, nil, EBadResponse
    }
    
    close := func() {
        // should do any cleanup on behalf of this request
        cancel()
        resp.Body.Close()
    }

    return resp.Body, close, nil
}