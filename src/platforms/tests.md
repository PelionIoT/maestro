
```
ed@box:~/work/gostuff/src/github.com/WigWagCo/maestro/plugin$  
   sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" \
     GOPATH=$GOPATH \
     LD_LIBRARY_PATH=../../greasego/deps/lib \
     /opt/go/bin/go test
```

or below - if you want to use `vendor` dir

```
ed@box:~/work/gostuff/src/github.com/WigWagCo/maestro/plugin$  
   sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" \
     GOPATH=$GOPATH \
     LD_LIBRARY_PATH=$GOPATH/src/github.com/WigWagCo/maestro/vendor/github.com/WigWagCo/greasego/deps/lib \
     /opt/go/bin/go test
```
