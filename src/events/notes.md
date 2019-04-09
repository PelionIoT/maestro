Running the test
=====

The eventManager deals with two type of events. Those which "fanout" to all listeners
queue all events for every listener. This is `fanout` `true` in setup.  When `fanout` is false
events will attempt to send in the order recieved from `SubmitEvent` to each slave channel 
simultaneously. If a given channel blocks - then that listeners event is dropped. 

Run something like this in this directory to test. Adjust for your environment:

```
sudo GOROOT=/opt/go PATH="$PATH:/opt/go/bin" GOPATH=/home/ed/work/gostuff \
   LD_LIBRARY_PATH=../../greasego/deps/lib /opt/go/bin/go test
```