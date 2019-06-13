// +build !debug

package debugging

const DebugEnabled = false

func DEBUG_OUT(args ...interface{})  {}
func DEBUG_OUT2(args ...interface{}) {}

func DumpMemStats()                   {}
func DebugPprof(debugServerFlag bool) {}
func RuntimeMemStats(duration int)    {}
