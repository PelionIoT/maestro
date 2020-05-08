package maestroConfig

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/armPelionEdge/devicedb/client"
	"github.com/armPelionEdge/devicedb/client_relay"
	"github.com/armPelionEdge/maestro/log"
	"github.com/armPelionEdge/maestroSpecs"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

var configClient *DDBRelayConfigClient

// DDBRelayConfigClient specifies some attributes for devicedb server that use to setup the client
type DDBMonitor struct {
	Uri             string
	Relay           string
	Bucket          string
	Prefix          string
	CaChain         string
	DDBConfigClient *DDBRelayConfigClient
}

//This function is called during maestro initilization to connect to deviceDB
//ddbConnConfig should contain a valid connection config
func NewDeviceDBMonitor(ddbConnConfig *DeviceDBConnConfig) (err error, ddbMonitor *DDBMonitor) {
	var tlsConfig *tls.Config

	if ddbConnConfig.RelayId == "" {
		fmt.Fprintf(os.Stderr, "No relay_id provided\n")
		err = errors.New("No relay_id provided\n")
	}

	if ddbConnConfig.CaChainCert == "" {
		fmt.Fprintf(os.Stderr, "No ca_chain provided\n")
		err = errors.New("No ca_chain provided\n")
	}

	relayCaChain, err := ioutil.ReadFile(ddbConnConfig.CaChainCert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load CA chain from %s: %v\n", ddbConnConfig.CaChainCert, err)
		err = errors.New(fmt.Sprintf("Unable to load CA chain from %s: %v\n", ddbConnConfig.CaChainCert, err))
	}

	caCerts := x509.NewCertPool()
	if !caCerts.AppendCertsFromPEM(relayCaChain) {
		fmt.Fprintf(os.Stderr, "CA chain loaded from %s is not valid: %v\n", ddbConnConfig.CaChainCert, err)
		err = errors.New(fmt.Sprintf("CA chain loaded from %s is not valid: %v\n", ddbConnConfig.CaChainCert, err))
	}

	tlsConfig = &tls.Config{
		RootCAs: caCerts,
	}

	configClient = NewDDBRelayConfigClient(tlsConfig, ddbConnConfig.DeviceDBUri, ddbConnConfig.RelayId, ddbConnConfig.DeviceDBPrefix, ddbConnConfig.DeviceDBBucket)

	ddbMonitor = &DDBMonitor{
		Uri:             ddbConnConfig.DeviceDBUri,
		Relay:           ddbConnConfig.RelayId,
		Bucket:          ddbConnConfig.DeviceDBBucket,
		Prefix:          ddbConnConfig.DeviceDBPrefix,
		DDBConfigClient: configClient,
		CaChain:         ddbConnConfig.CaChainCert,
	}

	return
}

//This function is used to add a configuration monitor for the "config" object with name "configName".
//configAnalyzer object is used for comparing new and old config objects which is used by the monitor
//when it detects a config object change
func (ddbMonitor *DDBMonitor) AddMonitorConfig(config interface{}, updatedConfig interface{}, configName string, configAnalyzer *maestroSpecs.ConfigAnalyzer) (err error) {
	go configMonitor(config, updatedConfig, configName, configAnalyzer, ddbMonitor.DDBConfigClient)
	return
}

//This function is used to delete a configuration monitor with name "configName".
func (ddbMonitor *DDBMonitor) RemoveMonitorConfig(configName string) (err error) {
	configWatcher := ddbMonitor.DDBConfigClient.Config(configName).Watch()
	configWatcher.Stop()
	return
}

//Go routine which monitors the config object changes, it then calls config analyzer which in turns calls the hook functions
func configMonitor(config interface{}, updatedConfig interface{}, configName string, configAnalyzer *maestroSpecs.ConfigAnalyzer, configClient *DDBRelayConfigClient) {
	configWatcher := configClient.Config(configName).Watch()
	configWatcher.Run()

	//Make a copy of original config
	prevconfig := config
	for {
		//log.MaestroWarnf("configMonitor: waiting on Next:%s", configName)
		exists := configWatcher.Next(updatedConfig)

		if !exists {
			fmt.Printf("Configuration %s no longer exists or not be watched, no need to listen anymore\n", configName)
			break
		}

		//log.MaestroWarnf("[%s] Configuration %s was updated: \nold:%+v \nnew:%v\n", time.Now(), configName, config, updatedConfig)
		same, noaction, err := configAnalyzer.CallChanges(prevconfig, updatedConfig)
		if err != nil {
			log.MaestroErrorf("Error from CallChanges: %s\n", err.Error())
		} else {
			log.MaestroInfof("CallChanges ret same=%+v noaction=%+v\n", same, noaction)
		}

		//Make a copy of previous config
		prevconfig := updatedConfig
		//The below statement is just to avoid compiler erroring about prevconfig not used
		_ = prevconfig
	}
}

/*
RelayConfigClient will be used by a gateway program.
It will provide a client to monitor the relative configuration file.
*/
type RelayConfigClient interface {
	Config(n string) Config
}

/*
Config interface provides the means for client to get the specified config,
and also could watch the updates about the config.
*/
type Config interface {
	Get(t interface{}) error
	Put(t interface{}) error
	Delete() error
	Watch() Watcher
}

/*
Watcher interface lets the client could use it to run the monitor,
receive the updates about the config and parse it as expected format
*/
type Watcher interface {
	// Run would start the go routine that handles the updates about the monitoring config
	Run()

	// Stop would stop the go routine that handles the updates about the monitoring config
	Stop()

	// Next would parse the config as the given interface and return true when the
	// configuration with given key still exists, otherwise it will return false
	Next(t interface{}) bool
}

// DDBRelayConfigClient specifies some attributes for devicedb server that use to setup the client
type DDBRelayConfigClient struct {
	Relay  string
	Bucket string
	Prefix string
	Client client_relay.Client
}

// Generic wrapper for storing the config structs
type ConfigWrapper struct {
	Name  string      `json:"name"`
	Relay string      `json:"relay"`
	Body  interface{} `json:"body"`
}

// NewDDBRelayConfigClient will initialize an client from devicedb/client_relay,
// and it will setup an instance of DDBRelayConfigClient
func NewDDBRelayConfigClient(tlsConfig *tls.Config, uri string, relay string, prefix string, bucket string) *DDBRelayConfigClient {
	config := client_relay.Config{
		ServerURI:             uri,
		TLSConfig:             tlsConfig,
		WatchReconnectTimeout: 5 * time.Second,
	}

	client := client_relay.New(config)

	return &DDBRelayConfigClient{
		Relay:  relay,
		Client: client,
		Bucket: bucket,
		Prefix: prefix,
	}
}

// Config return a config instance which is a monitoring client that specified by the given config name
func (ddbClient *DDBRelayConfigClient) Config(name string) Config {
	configName := fmt.Sprintf("%v.%v.%v", ddbClient.Prefix, ddbClient.Relay, name)

	return &DDBConfig{
		Key:          configName,
		Bucket:       ddbClient.Bucket,
		ConfigClient: ddbClient,
	}
}

// DDBConfig has the name of the config and also include the instance of DDBRelayConfigClient,
// which is used by the implementation of the Config interface
type DDBConfig struct {
	Key          string
	Bucket       string
	ConfigClient *DDBRelayConfigClient
}

func (rcc *DDBRelayConfigClient) IsAvailable() bool {
	_, err := rcc.Client.Get(context.Background(), rcc.Bucket, []string{"NULL"})
	if err != nil {
		log.MaestroErrorf("DDBRelayConfigClient.IsAvailable(): false, Error: %v", err)
		return false
	}
	log.MaestroInfo("DDBRelayConfigClient.IsAvailable(): true")
	return true
}

// Get function will get the config with the expecting format and fill it into the parameter t.
// It will return nil error when there is no such config exists or the config value could be
// parsed as the format that client specified, otherwise it will return false when the config
// value could not be parsed as expecting format
func (ddbConfig *DDBConfig) Get(t interface{}) (err error) {
	configEntries, err := ddbConfig.ConfigClient.Client.Get(context.Background(), ddbConfig.ConfigClient.Bucket, []string{ddbConfig.Key})
	if err != nil {
		log.MaestroErrorf("DDBConfig.Get(): Failed to get the matched config from the devicedb. Error: %v", err)
		return err
	}

	if configEntries == nil {
		err = errors.New(fmt.Sprintf("Object %s does not exist", ddbConfig.Key))
	} else {
		//fmt.Printf("\nDDBConfig: Object:%s is %v", ddbConfig.Key, configEntries)
		// the length of configEntries will be the same as the length of keys string that provided in the above Get function.
		// Since we only have one key for the Get function, we should have configEntries with length of 1. But the only entry
		// of the configEntries could be a nil value since the key might not exist
		if (len(configEntries) > 0) && (configEntries[0] != nil) && (len(configEntries[0].Siblings) != 0) {
			// the length of siblings might not be one since it might exist
			// multiple same config entries in the devicedb server. In this case,
			// we generally use the first one of the sorted siblings
			sortableConfigs := sort.StringSlice(configEntries[0].Siblings)
			sort.Sort(sortableConfigs)

			if len(sortableConfigs) > 0 {
				// the config are stored as the storage.Configuration struct,
				// and the config value that should be parsed as the expecting
				// format should be the value of config.Body

				var config ConfigWrapper
				_ = json.Unmarshal([]byte(sortableConfigs[0]), &config)
				configJSON := fmt.Sprintf("%v", config.Body)

				// parse the value of config.Body into the expecting format
				err = json.Unmarshal([]byte(configJSON), &t)
				if err != nil {
					fmt.Printf("\nDDBConfig.Get() could not parse the configuration value into expected format. Error %v", err)
					log.MaestroErrorf("DDBConfig.Get() could not parse the configuration value into expected format. Error %v", err)
					return err
				}
			}
		} else {
			err = errors.New(fmt.Sprintf("Object %s is deleted", ddbConfig.Key))
		}
	}

	return err
}

// Put function will write the passed config object(t) with the configName in ddbConfig object
func (ddbConfig *DDBConfig) Put(t interface{}) (err error) {
	//Ensure t is not nil
	if t != nil {
		bodyJSON, err := json.Marshal(&t)
		if err == nil {
			var config ConfigWrapper
			config.Body = string([]byte(bodyJSON))
			config.Relay = ""
			config.Name = ddbConfig.Key

			//Marshal the storage object to put into deviceDB
			bodyJSON, err := json.Marshal(&config)

			//log.MaestroInfof("DDBConfig.Put(): bodyJSON: %s\n", bodyJSON)
			if err == nil {
				var devicedbClientBatch *client.Batch
				ctx, _ := context.WithCancel(context.Background())
				devicedbClientBatch = client.NewBatch()
				devicedbClientBatch.Put(ddbConfig.Key, string([]byte(bodyJSON)), "")
				err = ddbConfig.ConfigClient.Client.Batch(ctx, ddbConfig.Bucket, *devicedbClientBatch)
				if err != nil {
					log.MaestroErrorf("DDBConfig.Put(): %v", err)
					return err
				}
			}
		}
	} else {
		err = errors.New("Put: Invalid argument")
		log.MaestroErrorf("DDBConfig.Put() Invalid argument. Error %v", err)
	}
	return err
}

// Delete function will remove the config object with the configName in ddbConfig object
func (ddbConfig *DDBConfig) Delete() (err error) {
	var devicedbClientBatch *client.Batch
	ctx, _ := context.WithCancel(context.Background())
	devicedbClientBatch = client.NewBatch()
	devicedbClientBatch.Delete(ddbConfig.Key, "")
	//log.MaestroErrorf("DDBConfig.Delete() Deleting key: %s", ddbConfig.Key)
	err = ddbConfig.ConfigClient.Client.Batch(ctx, ddbConfig.Bucket, *devicedbClientBatch)
	if err != nil {
		log.MaestroErrorf("DDBConfig.Delete(): %v", err)
	}

	return
}

// Watch will register a watcher for the client to
// monitor the updates about the given config
func (ddbConfig *DDBConfig) Watch() Watcher {
	return &DDBWatcher{
		Updates:              make(chan string),
		handleWatcherControl: make(chan string),
		Config:               ddbConfig,
	}
}

// DDBWatcher provides a channel that process the updates, and the
// config could be used while handling the updates from devicedb
type DDBWatcher struct {
	Updates              chan string
	Config               *DDBConfig
	handleWatcherControl chan string
}

// Run will start the go routine that handle the updates from
// the monitoring config value or errors from the devicedb
func (watcher *DDBWatcher) Run() {
	go watcher.handleWatcher()
}

// Stop will stop the handleWatched go routine in preparation for
// removing this watcher
func (watcher *DDBWatcher) Stop() {
	close(watcher.handleWatcherControl)
	close(watcher.Updates)
}

// Next would parse the config as the given interface and return true when the
// configuration with given key still exists, otherwise it will return false
func (watcher *DDBWatcher) Next(t interface{}) bool {
	//log.MaestroWarnf("DDBConfig: Entering Next()")
	for {
		// receive the updated config from the update channel of the Watcher
		u, ok := <-watcher.Updates

		//log.MaestroWarnf("DDBConfig.Next() Updates triggered")

		if !ok {
			log.MaestroWarnf("DDBConfig.Next() Updates channel closed, no need to listen anymore")
			break
		}

		if u == "" {
			log.MaestroWarnf("DDBWatcher.Next() found that the key has been deleted")
			return false
		}

		//log.MaestroInfof("DDBWatcher: Next: json:%s\n", []byte(u))
		// if the updated config could not be parsed as expecting format,
		// we will skip it util we could parse it successfully
		err := json.Unmarshal([]byte(u), &t)
		if err != nil {
			log.MaestroErrorf("DDBWatcher.Next() failed to parse the update into expected format. Error: %v", err)
			continue
		}

		return true
	}

	return false
}

// handleWatcher will monitor two channels: updates and errors.
// These two channel are returned by the Watch function from devicedb/client_relay

// For the updates channel, it will parse the config like the above Get function. And
// will send the message to the update channel of the Watcher, and Next() would handle
// the sent string for the update channel

// For the error channel, it will just simply print out the logs from the devicedb
func (watcher *DDBWatcher) handleWatcher() {
	//log.MaestroWarnf("DDBWatcher.handleWatcher(): bucket:%s key:%s", watcher.Config.ConfigClient.Bucket, watcher.Config.Key)
	updates, errors := watcher.Config.ConfigClient.Client.Watch(context.Background(), watcher.Config.ConfigClient.Bucket, []string{watcher.Config.Key}, []string{}, 0)

	// drain up the channel to avoid deadlock
	defer func() {
		go func() {
			for range updates {
			}

			for range errors {
			}

			for range watcher.Updates {
			}
		}()
	}()

	for {
		select {
		case update, ok := <-updates:

			if !ok {
				log.MaestroErrorf("DDBConfig.handleWatcher() the DeviceDB monitor encountered a protocol error and have already cancelled the watcher")
				break
			}

			if update.IsEmpty() {
				continue
			}

			sortableConfigs := sort.StringSlice(update.Siblings)
			sort.Sort(sortableConfigs)
			var configLen int
			configLen = len(sortableConfigs)
			if configLen == 0 {
				log.MaestroInfof("DDBConfig.handleWatcher(): configLen == 0")
				watcher.Updates <- ""
				continue
			}

			var config ConfigWrapper
			err := json.Unmarshal([]byte(sortableConfigs[0]), &config)
			if err == nil {
				bodyJSON := fmt.Sprintf("%v", config.Body)
				watcher.Updates <- string(bodyJSON)
			} else {
				log.MaestroWarnf("DDBConfig.handleWatcher(): json.Unmarshal error: %v configs:%v", err, sortableConfigs[0])
			}

		case err, ok := <-errors:
			log.MaestroErrorf("DDBConfig.handleWatcher() received an error from the watcher. Error: %v", err)
			if !ok {
				log.MaestroWarnf("DDBConfig.handleWatcher() the DeviceDB monitor encounter a protocol error and have already cancelled the watcher")
				break
			}

		case _, ok := <-watcher.handleWatcherControl:
			log.MaestroErrorf("DDBConfig.handleWatcher() -watcher.handleWatcherControl triggered")
			if !ok {
				log.MaestroWarnf("DDBConfig.handleWatcher() got channel closed, no need to listen anymore")
				break
			}
		}
	}
}
