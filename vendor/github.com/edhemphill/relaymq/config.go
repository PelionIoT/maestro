package relaymq

import (
    "io/ioutil"
    "errors"
    "fmt"
    "gopkg.in/yaml.v2"
)

const (
    DefaultPrefetchLimit = 10
)

type YAMLServerConfig struct {
    Broker YAMLBroker `yaml:"broker"`
    Port int `yaml:"port"`
    LogLevel string `yaml:"logLevel"`
}

type YAMLBroker struct {
    PrefetchLimit int `yaml:"prefetchLimit"`
    Channels int `yaml:"channels"`    
    Host string `yaml:"host"`
    Port int `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

func (ysc *YAMLServerConfig) LoadFromFile(file string) error {
    rawConfig, err := ioutil.ReadFile(file)
    
    if err != nil {
        return err
    }
    
    err = yaml.Unmarshal(rawConfig, ysc)
    
    if err != nil {
        return err
    }
    
    if !isValidPort(ysc.Port) {
        return errors.New(fmt.Sprintf("%d is an invalid port for the forwarder server", ysc.Port))
    }

    if ysc.Broker.Host == "" {
        return errors.New(fmt.Sprintf("No host name specificed for broker"))
    }

    if !isValidPort(ysc.Broker.Port) {
        return errors.New(fmt.Sprintf("%d is not a valid port for the broker", ysc.Broker.Port))
    }

    if ysc.Broker.Channels <= 0 {
        ysc.Broker.Channels = 1
    }

    if ysc.Broker.PrefetchLimit <= 0 {
        ysc.Broker.PrefetchLimit = DefaultPrefetchLimit
    }
    
    SetLoggingLevel(ysc.LogLevel)
    
    return nil
}

func isValidPort(p int) bool {
    return p >= 0 && p < (1 << 16)
}
