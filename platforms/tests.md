
```
ed@box:~/work/gostuff/src/github.com/PelionIoT/maestro/plugin$  
   sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" \
     GOPATH=$GOPATH \
     LD_LIBRARY_PATH=../../greasego/deps/lib \
     /opt/go/bin/go test
```

or below - if you want to use `vendor` dir

```
ed@box:~/work/gostuff/src/github.com/PelionIoT/maestro/plugin$  
   sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" \
     GOPATH=$GOPATH \
     LD_LIBRARY_PATH=$GOPATH/src/github.com/PelionIoT/maestro/vendor/github.com/PelionIoT/greasego/deps/lib \
     /opt/go/bin/go test
```
