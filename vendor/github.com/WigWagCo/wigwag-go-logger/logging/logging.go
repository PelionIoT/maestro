package logging

import (
	"os"
	"strings"
	"sync"

	"github.com/op/go-logging"
)

type splitLogBackend struct {
	rwMu          sync.RWMutex
	component     string
	outLogBackend logging.LeveledBackend
	errLogBackend logging.LeveledBackend
}

func newSplitLogBackend(outLogBackend, errLogBackend logging.LeveledBackend) *splitLogBackend {
	return &splitLogBackend{
		outLogBackend: outLogBackend,
		errLogBackend: errLogBackend,
	}
}

func (slb *splitLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	// Uses RWMutex so that calls to Log can happen concurrently with each other but not
	// with updates to the log level
	slb.rwMu.RLock()
	defer slb.rwMu.RUnlock()

	rec.Module = slb.component

	if level <= logging.WARNING {
		return slb.errLogBackend.Log(level, calldepth+1, rec)
	}

	return slb.outLogBackend.Log(level, calldepth+1, rec)
}

func (slb *splitLogBackend) SetLevel(level logging.Level, module string) {
	slb.rwMu.Lock()
	defer slb.rwMu.Unlock()

	slb.outLogBackend.SetLevel(level, module)
	slb.errLogBackend.SetLevel(level, module)
}

func (slb *splitLogBackend) SetComponent(component string) {
	slb.rwMu.Lock()
	defer slb.rwMu.Unlock()

	slb.component = component
}

var Log = logging.MustGetLogger("")
var log = Log
var loggingBackend *splitLogBackend

func init() {
	var format = logging.MustStringFormatter(`%{color}%{time:[2006-01-02T15:04:05.999]} [%{level:.8s}] [%{module}]%{color:reset} %{message}`)
	var outBackend = logging.NewLogBackend(os.Stdout, "", 0)
	var outBackendFormatter = logging.NewBackendFormatter(outBackend, format)
	var outLogBackend = logging.AddModuleLevel(outBackendFormatter)
	var errBackend = logging.NewLogBackend(os.Stderr, "", 0)
	var errBackendFormatter = logging.NewBackendFormatter(errBackend, format)
	var errLogBackend = logging.AddModuleLevel(errBackendFormatter)

	loggingBackend = newSplitLogBackend(outLogBackend, errLogBackend)

	logging.SetBackend(loggingBackend)

	go watchLoggingLevelConfig()
	go watchLoggingComponentConfig()
}

func LogLevelIsValid(ll string) bool {
	_, err := logging.LogLevel(strings.ToUpper(ll))

	return err == nil
}

func SetLoggingLevel(ll string) {
	logLevel, err := logging.LogLevel(strings.ToUpper(ll))

	if err != nil {
		logLevel = logging.ERROR
	}

	loggingBackend.SetLevel(logLevel, "")
}

func SetLoggingComponent(component string) {
	loggingBackend.SetComponent(component)
}
