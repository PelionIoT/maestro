package client

import (
    "errors"
    "net/http"
    "bytes"
    "io/ioutil"
    "encoding/json"
    "time"
    "strings"
    "context"
    "fmt"

    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/cluster"

    "github.com/armPelionEdge/devicedb/rest"
)

const DefaultClientTimeout = time.Second * 10

type ErrorStatusCode struct {
    StatusCode int
    Message string
}

func (errorStatus *ErrorStatusCode) Error() string {
    return errorStatus.Message
}

type ClientConfig struct {
    Timeout time.Duration
}

var EClientTimeout = errors.New("Client request timed out")

type Client struct {
    httpClient *http.Client
}

func NewClient(config ClientConfig) *Client {
    if config.Timeout == 0 {
        config.Timeout = DefaultClientTimeout
    }

    return &Client{
        httpClient: &http.Client{ 
            Timeout: config.Timeout,
        },
    }
}

func (client *Client) sendRequest(ctx context.Context, httpVerb string, endpointURL string, body []byte) ([]byte, error) {
    request, err := http.NewRequest(httpVerb, endpointURL, bytes.NewReader(body))

    if err != nil {
        return nil, err
    }

    request = request.WithContext(ctx)

    resp, err := client.httpClient.Do(request)

    if err != nil {
        if strings.Contains(err.Error(), "Timeout") {
            return nil, EClientTimeout
        }

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

// Use an existing cluster member to bootstrap the addition of another node
// to that cluster. host and port indicate the address of the existing cluster 
// member while nodeAddress contains the ID, host name and port of the new
// cluster member
//
// Return Values:
//   EClientTimeout: The request to the node timed out
func (client *Client) AddNode(ctx context.Context, memberAddress PeerAddress, newMemberConfig NodeConfig) error {
    encodedNodeConfig, _ := json.Marshal(newMemberConfig)
    _, err := client.sendRequest(ctx, "POST", memberAddress.ToHTTPURL("/cluster/nodes"), encodedNodeConfig)

    if _, ok := err.(*ErrorStatusCode); ok {
        var dbError DBerror

        parseErr := json.Unmarshal([]byte(err.(*ErrorStatusCode).Message), &dbError)

        if parseErr == nil {
            return dbError
        }
    }

    return err
}

// Ask a cluster member to initiate the removal of some node from its cluster.
// host and port indicate the address of the initiator node while nodeID is
// the ID of the node that should be removed.
//
// Return Values:
//   EClientTimeout: The request to the node timed out
func (client *Client) RemoveNode(ctx context.Context, memberAddress PeerAddress, nodeID uint64, replacementNodeID uint64, decommission, forwarded bool) error {
    var queryString = ""

    endpoint := memberAddress.ToHTTPURL("/cluster/nodes/" + fmt.Sprintf("%d", nodeID))

    if forwarded {
        queryString += "forwarded=true&"
    }

    if decommission {
        queryString += "decommission=true&"
    }

    if replacementNodeID != 0 {
        queryString += fmt.Sprintf("replace=%d&", replacementNodeID)
    }

    if len(queryString) != 0 {
        // take off the last &
        queryString = queryString[:len(queryString) - 1]
    }

    endpoint += "?" + queryString

    _, err := client.sendRequest(ctx, "DELETE", endpoint, []byte{ })
    
    return err
}

func (client *Client) DecommissionNode(ctx context.Context, memberAddress PeerAddress, nodeID uint64) error {
    return client.RemoveNode(ctx, memberAddress, nodeID, 0, true, false)
}

func (client *Client) ForceRemoveNode(ctx context.Context, memberAddress PeerAddress, nodeID uint64) error {
    return client.RemoveNode(ctx, memberAddress, nodeID, 0, false, false)
}

func (client *Client) ReplaceNode(ctx context.Context, memberAddress PeerAddress, nodeID uint64, replacementNodeID uint64) error {
    return client.RemoveNode(ctx, memberAddress, nodeID, replacementNodeID, false, false)
}

func (client *Client) MerkleTreeStats(ctx context.Context, memberAddress PeerAddress, siteID string, bucketName string) (rest.MerkleTree, error) {
    endpoint := memberAddress.ToHTTPURL(fmt.Sprintf("/sites/%s/buckets/%s/merkle", siteID, bucketName))
    response, err := client.sendRequest(ctx, "GET", endpoint, []byte{ })

    if err != nil {
        return rest.MerkleTree{}, err
    }

    var merkleTree rest.MerkleTree

    if err := json.Unmarshal(response, &merkleTree); err != nil {
        return rest.MerkleTree{}, err
    }

    return merkleTree, nil
}

func (client *Client) MerkleTreeNode(ctx context.Context, memberAddress PeerAddress, siteID string, bucketName string, nodeID uint32) (rest.MerkleNode, error) {
    endpoint := memberAddress.ToHTTPURL(fmt.Sprintf("/sites/%s/buckets/%s/merkle/nodes/%d", siteID, bucketName, nodeID))
    response, err := client.sendRequest(ctx, "GET", endpoint, []byte{ })

    if err != nil {
        return rest.MerkleNode{}, err
    }

    var merkleNode rest.MerkleNode

    if err := json.Unmarshal(response, &merkleNode); err != nil {
        return rest.MerkleNode{}, err
    }

    return merkleNode, nil
}

func (client *Client) MerkleTreeNodeKeys(ctx context.Context, memberAddress PeerAddress, siteID string, bucketName string, nodeID uint32) (rest.MerkleKeys, error) {
    endpoint := memberAddress.ToHTTPURL(fmt.Sprintf("/sites/%s/buckets/%s/merkle/nodes/%d/keys", siteID, bucketName, nodeID))
    response, err := client.sendRequest(ctx, "GET", endpoint, []byte{ })

    if err != nil {
        return rest.MerkleKeys{}, err
    }

    var merkleKeys rest.MerkleKeys

    if err := json.Unmarshal(response, &merkleKeys); err != nil {
        return rest.MerkleKeys{}, err
    }

    return merkleKeys, nil
}
