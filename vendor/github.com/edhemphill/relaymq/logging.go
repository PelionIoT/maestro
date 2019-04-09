package relaymq

import (
    "os"
    "github.com/op/go-logging"
    "strings"
)

type SplitLogBackend struct {
    outLogBackend logging.LeveledBackend
    errLogBackend logging.LeveledBackend
}

func NewSplitLogBackend(outLogBackend, errLogBackend logging.LeveledBackend) SplitLogBackend {
    return SplitLogBackend{
        outLogBackend: outLogBackend,
        errLogBackend: errLogBackend,
    }
}

func (slb SplitLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
    if level <= logging.WARNING {
        return slb.errLogBackend.Log(level, calldepth + 2, rec)
    }
    
    return slb.outLogBackend.Log(level, calldepth + 2, rec)
}

func (slb SplitLogBackend) SetLevel(level logging.Level, module string) {
    slb.outLogBackend.SetLevel(level, module)
    slb.errLogBackend.SetLevel(level, module)
}

var Log = logging.MustGetLogger("devicedb")
var log = Log
var loggingBackend SplitLogBackend

func init() {
    var format = logging.MustStringFormatter(`%{color}%{time:15:04:05.000} ▶ %{level:.4s} %{shortfile}%{color:reset} %{message}`)
    var outBackend = logging.NewLogBackend(os.Stdout, "", 0)
    var outBackendFormatter = logging.NewBackendFormatter(outBackend, format)
    var outLogBackend = logging.AddModuleLevel(outBackendFormatter)
    var errBackend = logging.NewLogBackend(os.Stderr, "", 0)
    var errBackendFormatter = logging.NewBackendFormatter(errBackend, format)
    var errLogBackend = logging.AddModuleLevel(errBackendFormatter)
    
    loggingBackend = NewSplitLogBackend(outLogBackend, errLogBackend)
    
    logging.SetBackend(loggingBackend)
}

func SetLoggingLevel(ll string) {
    logLevel, err := logging.LogLevel(strings.ToUpper(ll))
    
    if err != nil {
        logLevel = logging.ERROR
    }
    
    loggingBackend.SetLevel(logLevel, "")
}
