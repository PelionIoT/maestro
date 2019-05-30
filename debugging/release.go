// +build !debug

package debugging

const DebugEnabled = false

func DEBUG_OUT(args ...interface{})  {}
func DEBUG_OUT2(args ...interface{}) {}

func DumpMemStats()                                   {}
func DebugApp(pprof bool, runtime bool, duration int) {}
