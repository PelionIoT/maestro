# structmutate
Golang utilities to manipulate structs

`func MergeInStruct(a interface{},b interface{}) (error)`

`a` and `b` must be pointers to structs. This will merge all exported fields of `b` into `a` if and only if their:
- field names match
- if `b`'s field have non zero-value values

```
import (
    "github.com/PelionIoT/structmutate"
    "fmt"
)

var debugout = func (format string, a ...interface{}) {
    s := fmt.Sprintf(format, a...)
    fmt.Printf("[debug-structmutate]  %s\n", s)
}

func TestMain(m *testing.M) {
    m.Run()
}

func TestMergeIn(t *testing.T) {
    structmutate.SetupUtilsLogs(debugout,nil) // only needed if you want to see what's happening

    type foo struct {
        One int
        Two string
        Three []byte
    }

    type bar struct {
        One int
        Notthere int
        Three []byte
    }

    foo1 := &foo{
        One: 1,
        Two: "foo!!",
        Three: []byte{0,1,2},
    }

    bar1 := &bar{
        One: 101,
        Notthere: 2,
        Three: []byte{99,100},
    }

    err := structmutate.MergeInStruct(foo1,bar1)
    if err != nil {
        log.Fatal("Error: ",err)
    }
    if foo1.One != 101 || len(foo1.Three) != 2 || foo1.Two != "foo!!" {
        log.Fatalf("Did not get expect results. foo1 = %+v\n",foo1)
    }

    fmt.Printf("foo1: %+v\n",foo1)
}
```
