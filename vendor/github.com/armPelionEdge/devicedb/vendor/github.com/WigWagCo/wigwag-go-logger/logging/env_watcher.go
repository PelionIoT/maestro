package logging

import (
	"io/ioutil"
	"os"
	"time"
)

const (
	// debug, info, notice, warning, error, critical
	LogLevelEnvironmentVariable     string = "WIGWAG_LOG_LEVEL"
	LogComponentEnvironmentVariable string = "WIGWAG_LOG_COMPONENT"
	LogConfigSyncPeriodSeconds      int    = 1
)

// Checks the log level environment variable periodically for changes
// update running log level if necessary
func watchLoggingLevelConfig() {
	var logLevelSetting string

	for {
		time.Sleep(time.Second * time.Duration(LogConfigSyncPeriodSeconds))

		var logLevelSettingFile string = os.Getenv(LogLevelEnvironmentVariable)

		if logLevelSettingFile == "" {
			continue
		}

		contents, err := ioutil.ReadFile(logLevelSettingFile)

		if err != nil {
			Log.Errorf("Unable to retrieve log level from %s: %v", logLevelSettingFile, err)

			continue
		}

		var newLogLevelSetting string = string(contents)

		if logLevelSetting != newLogLevelSetting && LogLevelIsValid(newLogLevelSetting) {
			Log.Debugf("Setting logging level to %s", newLogLevelSetting)

			logLevelSetting = newLogLevelSetting

			SetLoggingLevel(newLogLevelSetting)
		}
	}
}

func watchLoggingComponentConfig() {
	var logComponentSetting string

	for {
		time.Sleep(time.Second * time.Duration(LogConfigSyncPeriodSeconds))

		var logComponentSettingFile string = os.Getenv(LogComponentEnvironmentVariable)

		if logComponentSettingFile == "" {
			continue
		}

		contents, err := ioutil.ReadFile(logComponentSettingFile)
		if err != nil {
			Log.Errorf("Unable to retrieve log component from %s: %v", logComponentSettingFile, err)

			continue
		}

		var newLogComponentSetting string = string(contents)

		if logComponentSetting != newLogComponentSetting {
			Log.Debugf("Setting logging component to %s", newLogComponentSetting)

			logComponentSetting = newLogComponentSetting

			SetLoggingComponent(newLogComponentSetting)
		}
	}
}
