Running these tests
=====

* Most tests require you to modify the interface, to a network interface you have available on your system for testing. For instance,
on my box, my USB ethernet dongle goes to `eth2` - so in most cases I change the test to use `eth2` - if it involves needing a real physical interface. You should modify tests in `src/networking` not `networking` which is not in version control.

* If you modify the test with your specific interface, you should re-run the building the builder script, from the top level of the project, with desired debug levels: `DEBUG=1 DEBUG2=1 ./build.sh`

* When running the tests, you will need to indicate to the shell where the maestro shared libs are.

Out of the /networking folder, from the project root, run something like:

```
ed@box:~/work/gostuff/src/github.com/PelionIoT/maestro/networking$
   sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" \
     GOPATH=/home/ed/work/gostuff \
     LD_LIBRARY_PATH=../../greasego/deps/lib \
     /opt/go/bin/go test -run SetupInterfaces
```
