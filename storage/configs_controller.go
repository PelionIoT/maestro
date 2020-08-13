package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/armPelionEdge/devicedb/client"

	"github.com/armPelionEdge/maestro/logging"
)

type contextKey string

const (
	CtxRequestID = contextKey("RequestID")
	CtxAccountID = contextKey("AccountID")
)

// ConfigStore specify what kinds of functions that a controller of configuration should have
type ConfigStore interface {
	AddConfig(ctx context.Context, config Configuration) error
	GetConfigs(ctx context.Context, query ConfigurationQuery) (ConfigurationPage, error)
	RemoveConfig(ctx context.Context, deviceID string, name string) error
}

// Configuration specifies the attributes of Configuration
type Configuration struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Relay string      `json:"relay"`
	Body  interface{} `json:"body"`
}

// ConfigurationPage specifies the response for GET /v3alpha/devices/{device-id}/configurations request
type ConfigurationPage struct {
	After      string          `json:"after"`
	Data       []Configuration `json:"data"`
	HasMore    bool            `json:"has_more"`
	Limit      uint64          `json:"limit"`
	Object     string          `json:"object"`
	Order      string          `json:"order"`
	TotalCount uint64          `json:"total_count,omitempty"`
}

type ConfigurationQuery struct {
	DeviceID   string
	AccountID  string
	Name       string
	Limit      uint64
	Descending bool
	After      string
	TotalCount bool
}

// DDBConfigStore specifies the struct of the a DDB version of ConfigStore
type DDBConfigStore struct {
	Prefix    string
	Bucket    string
	Servers   []string
	APIClient client.APIClient
}

func extractKeyFromContext(ctx context.Context) (interface{}, interface{}) {
	var requestID interface{}
	requestID = ctx.Value(CtxRequestID)
	if requestID == nil {
		logging.Log.Warningf("extractKeyFromContext(): Missing requestID in the context")
		return "", ""
	}
	var accountID interface{}
	accountID = ctx.Value(CtxAccountID)
	if accountID == nil {
		logging.Log.Warningf("extractKeyFromContext(): Missing accountID in the context")
		return "", ""
	}
	return requestID, accountID
}

// NewDDBConfigStore will initialize an instance of DDBConfigStore
func NewDDBConfigStore(servers []string, prefix string, bucket string) (*DDBConfigStore, error) {
	config := client.APIClientConfig{Servers: servers}
	apiClient := *client.New(config)
	return &DDBConfigStore{Prefix: prefix, Bucket: bucket, Servers: servers, APIClient: apiClient}, nil
}

// AddConfig will update the configuration in the DeviceDB, will save it if the original config doesn't exist, or will update it if it already exists
func (store *DDBConfigStore) AddConfig(ctx context.Context, config Configuration) error {
	// Extract the requestID and accountID from context
	requestID, accountID := extractKeyFromContext(ctx)

	// Key: configs.device.{device_id}.{configname}
	key := fmt.Sprintf("%v.%v.%v", store.Prefix, config.Relay, config.Name)

	value, err := json.Marshal(config)
	if err != nil {
		logging.Log.Warningf("[RequestID:%v,AccountID:%v] DDBConfigStore.AddConfig(): Warning: failed to encode the configuration - %v", requestID, accountID, err)
		return err
	}

	batch := client.NewBatch()
	batch.Put(key, string(value), "")
	_, _, err = store.APIClient.Batch(ctx, config.Relay, store.Bucket, *batch)
	if err != nil {
		logging.Log.Errorf("[RequestID:%v,AccountID:%v] DDBConfigStore.AddConfig(): Error: %v", requestID, accountID, err)

		return err
	}

	return nil
}

// GetConfigs will get the configurations that matched the provided key
func (store *DDBConfigStore) GetConfigs(ctx context.Context, query ConfigurationQuery) (ConfigurationPage, error) {
	// Extract the requestID and accountID from context
	requestID, accountID := extractKeyFromContext(ctx)

	deviceID := query.DeviceID

	var prefixes []string

	var prefix string
	if query.Name == "" {
		prefix = fmt.Sprintf("%v.%v", store.Prefix, deviceID)
	} else {
		prefix = fmt.Sprintf("%v.%v.%v", store.Prefix, deviceID, query.Name)
	}

	prefixes = append(prefixes, prefix)

	configIterator, err := store.APIClient.GetMatches(ctx, query.DeviceID, store.Bucket, prefixes)
	if err != nil {
		logging.Log.Errorf("[RequestID:%v,AccountID:%v] DDBConfigStore().GetConfigs(): Error: %v", requestID, accountID, err)

		return ConfigurationPage{}, err
	}

	var page ConfigurationPage
	page.Data = make([]Configuration, 0)

	// retrieve all configurations for pagination response
	configsMapping := make(map[string]*Configuration)
	configIDs := make([]string, 0)
	for configIterator.Next() {
		sortableConfigs := sort.StringSlice(configIterator.Entry().Siblings)
		sort.Sort(sortableConfigs)

		if len(sortableConfigs) > 0 {
			var config Configuration
			err := json.Unmarshal([]byte(sortableConfigs[0]), &config)
			if err == nil {
				configsMapping[config.ID] = &config
				configIDs = append(configIDs, config.ID)
			}
		}
	}

	page.After = query.After
	page.Limit = query.Limit
	page.Object = "list"

	// default order is ascending by time, which is determined by the first six bytes of configuration.ID
	if query.Descending {
		page.Order = "DESC"
		sort.Sort(sort.Reverse(sort.StringSlice(configIDs)))
	} else {
		page.Order = "ASC"
		sort.Sort(sort.StringSlice(configIDs))
	}
	// handle pagination, searchIndex is the index of the configuration after the After Cursor
	searchIndex := 0
	if query.After != "" {
		searchIndex = sort.SearchStrings(configIDs, query.After) + 1
	}

	for ; searchIndex < len(configIDs); searchIndex++ {
		if query.Limit > 0 {
			page.Data = append(page.Data, *configsMapping[configIDs[searchIndex]])
		} else {
			break
		}

		query.Limit--
	}

	// if searchIndex is still not out of the range of the array of config IDs, set HasMore as true
	if searchIndex < len(configIDs) {
		page.HasMore = true
	}

	if query.TotalCount {
		page.TotalCount = uint64(len(page.Data))
	}

	return page, nil
}

// RemoveConfig will remove the configuration from the DeviceDB server with exact key
func (store *DDBConfigStore) RemoveConfig(ctx context.Context, deviceID string, name string) error {
	// Extract the requestID and accountID from context
	requestID, accountID := extractKeyFromContext(ctx)

	batch := client.NewBatch()

	// Key: configs.device.{device_id}.{configname}
	batch.Delete(fmt.Sprintf("%v.%v.%v", store.Prefix, deviceID, name), "")

	_, _, err := store.APIClient.Batch(ctx, deviceID, store.Bucket, *batch)
	if err != nil {
		logging.Log.Errorf("[RequestID:%v,AccountID:%v] DDBConfigStore.RemoveConfig(): Error: %v", requestID, accountID, err)

		return err
	}

	return nil
}
