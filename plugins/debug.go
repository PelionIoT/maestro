// +build debug

package plugins

import(
	"fmt"
	"unsafe"
	"reflect"
	"plugin"
)

func InspectPlugin(p *plugin.Plugin) {
	pl := (*Plug)(unsafe.Pointer(p))

	fmt.Printf("Plugin %s exported symbols (%d): \n", pl.Path, len(pl.Symbols))

	for name, pointers := range pl.Symbols {
		fmt.Printf("symbol: %s, pointer: %v, type: %v\n", name, pointers, reflect.TypeOf(pointers))
	}
}
