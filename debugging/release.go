// +build !debug

package debugging

const DebugEnabled = false

func DumpMemStats() {}

func DEBUG_OUT(args ...interface{}) {}
func DEBUG_OUT2(args ...interface{}) {}
