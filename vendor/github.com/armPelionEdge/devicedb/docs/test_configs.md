# Test Configurations
The .yaml files in test_configs directory are devicedb configurations files mainly used for testing.<br>
See the inline comments in ```test_config_nocluster.yaml``` for description on configuration items available.<br>

## Building and Running devicedb locally(with no cluster configuration)
For test purposes you can build devicedb and run it locally using ```test_config_nocluster.yaml``` configuration.<br> 
Following methods are available to build and run devicedb locally in no cluster configuration.<br>
In this configuration devicedb will be listening on port 9090.<br>

### Natively
1. Install and setup golang
   1. Instructions at https://golang.org/doc/install
1. Create a setup script with following contents. This script assumes that you are creating devicedb under ```$HOME/devicedb_test``` directory.<br>
   If you want to use a different directory change the script as required.
   ```shell
   #!/bin/bash
   export GIT_TERMINAL_PROMPT=1
   export GOROOT=/opt/go
   export GOPATH="$HOME/devicedb_test"
   export GOBIN="$HOME/devicedb_test"
   export PATH="$PATH:$GOROOT/bin:$GOBIN:$GOPATH"
   ```
1. Execute the script as follows   
   1. ```source ./my_setup_script.sh```<br> **where my_setup_script.sh will have the contents mentioned above.**
1. Create and build the devicedb repo using ```go get``` command as follow:
   1. go get github.com:armPelionEdge/devicedb
1. Executing devicedb
   1. cd $GOBIN
   1. ./devicedb start -conf=src/github.com/armPelionEdge/devicedb/test_configs/test_config_nocluster.yaml<br> **This should start running devicedb, you should be able to connect to it using URI = 127.0.0.1:9090**

### Using Docker
If you have Docker installed you can build and run devicedb as follows:
1. Clone the devicedb repo using ```git clone```
1. Change directory to where you cloned devicedb
1. Execute docker build using Dockerfile.test_local as follows.
   1. docker build -t devicedb_test_local -f Dockerfile.test_local .
1. Run the devicedb_test_local docker container as follows.
   1. docker run -p 127.0.0.1:9090:9090/tcp -it devicedb_test_local <br> **This should start running devicedb in the container, you should be able to connect to it using URI = 127.0.0.1:9090**


