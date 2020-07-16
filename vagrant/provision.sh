#!/usr/bin/env bash

# Install prerequisite packages
add-apt-repository ppa:rmescandon/yq
apt-get update
apt-get install -y build-essential python wget git nodejs-legacy npm m4 docker.io docker-compose uuid yq jq
systemctl start docker
systemctl enable docker

# Upgrade node. Needed since we are using nodejs-legacy package
# We need nodejs-legacy because mocha (test framework) expects `node` not `nodejs`
# `node` is nodejs-legacy and `nodejs` is nodejs
npm cache clean -f
npm install -g n
n stable
apt-get install --reinstall nodejs-legacy

# Install docker-compose
sudo curl -L "https://github.com/docker/compose/releases/download/1.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo usermod -aG docker vagrant

# Download GO
wget https://dl.google.com/go/go1.13.5.linux-amd64.tar.gz
tar xvzf go1.13.5.linux-amd64.tar.gz
rm go1.13.5.linux-amd64.tar.gz
rm -rf /opt/go
mv go /opt/go

# Set GO environment variables
cat <<'EOF' >/etc/profile.d/envvars.sh
export GIT_TERMINAL_PROMPT=1
export GOROOT=/opt/go
export GOPATH=/home/vagrant/work/gostuff
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOROOT/bin:$GOBIN

export MAESTRO_SRC=$GOPATH/src/github.com/armPelionEdge/maestro

export LD_LIBRARY_PATH=$MAESTRO_SRC/vendor/github.com/armPelionEdge/greasego/deps/lib
export DEVICEDB_SRC=$GOPATH/src/github.com/armPelionEdge/devicedb
export EDGE_CLIENT_RESOURCES=/etc/devicedb/shared
export CLOUD_HOST=localhost
export CLOUD_URI=ws://$CLOUD_HOST:8080/sync
export EDGE_DATA_DIRECTORY=/var/lib/devicedb/data
export EDGE_SNAP_DIRECTORY=/var/lib/devicedb/snapshots
export EDGE_LISTEN_PORT=9090
export EDGE_LOG_LEVEL=info
export MAESTRO_CERTS=/var/lib/maestro/certs
export MAESTRO_LOGS=/var/log/maestro
export EDGE_CLIENT_CERT=$EDGE_CLIENT_RESOURCES/client.crt
export EDGE_CLIENT_KEY=$EDGE_CLIENT_RESOURCES/client.key
export EDGE_CLIENT_CA=$EDGE_CLIENT_RESOURCES/myCA.pem
export COVERITY_HOME=/home/vagrant/cov-analysis-linux64-2020.03
export PATH=$PATH:$COVERITY_HOME/bin

function site_id() { cat $DEVICEDB_SRC/hack/certs/site_id; }
function relay_id() { cat $DEVICEDB_SRC/hack/certs/device_id; }
# shorthand for 'devicedb cluster' which provides reasonable defaults
function dc() {
    local site=$(site_id)
    local relay=$(relay_id)
    local bucket=lww
    local prefix=vagrant
    local cmd=get_matches
    local -a args=(cluster cmd)
    while [ -n "$1" ]; do
        shift_count=2
        case "$1" in
            -site)
                site="$2"
                ;;
            -relay)
                relay="$2"
                ;;
            -bucket)
                bucket="$2"
                ;;
            -prefix)
                prefix="$2"
                ;;
            -help)
                args+=("$1")
                shift_count=1
                ;;
            -*)
                args+=("$1" "$2")
                ;;
            *)
                cmd="$1"
                args[1]="$cmd"
                shift_count=1
                ;;
        esac
        shift $shift_count
    done
    if [ "${args[2]}" != "-help" ]; then
      case "$cmd" in
        relay_status)
            args+=(-site "$site" -relay "$relay")
            ;;
        get|put|delete)
            args+=(-site "$site" -bucket "$bucket")
            ;;
        get_matches)
            args+=(-site "$site" -bucket "$bucket" -prefix "$prefix")
            ;;
        *)  args+=(-site "$site")
            ;;
      esac
    fi
    devicedb "${args[@]}"
}
EOF
. /etc/profile.d/envvars.sh

# Create directories for read/write of vagrant user
mkdir /var/maestro
chmod 777 /var/maestro

# Create directories for devicedb resources (certs)
mkdir -p $EDGE_CLIENT_RESOURCES
chmod 777 $EDGE_CLIENT_RESOURCES
mkdir -p $EDGE_SNAP_DIRECTORY
chmod 777 $EDGE_SNAP_DIRECTORY
mkdir -p $EDGE_DATA_DIRECTORY
chmod 777 $EDGE_DATA_DIRECTORY
mkdir -p $MAESTRO_CERTS
chmod 777 $MAESTRO_CERTS
mkdir -p $MAESTRO_LOGS
chmod 777 $MAESTRO_LOGS

# Create a script to go to the maestro source and run maestro so maestro has access to its' configuration files
# Allows a user to run 'sudo maestro' and have everything work out
echo "#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $MAESTRO_SRC
exec $GOBIN/maestro
" > /usr/sbin/maestro
chmod +x /usr/sbin/maestro

# Do the same for maestro-shell
echo "#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $GOPATH/src/github.com/armPelionEdge/maestro-shell
exec $GOBIN/maestro-shell
" > /usr/sbin/maestro-shell
chmod +x /usr/sbin/maestro-shell

# Set the network interface to eth0 instead of Ubuntu 16.04 default enp0s3
rm /etc/udev/rules.d/70-persistent-net.rules
sed -i -e 's/GRUB_CMDLINE_LINUX=""/GRUB_CMDLINE_LINUX="net.ifnames=0 biosdevname=0"/g' /etc/default/grub
update-grub

# Add a secondary network interface for test traffic
echo "auto lo
iface lo inet loopback
auto eth0
iface eth0 inet dhcp
" > /etc/network/interfaces.d/50-cloud-init.cfg

# Add a script to start devicedb as an edge node
echo "#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $DEVICEDB_SRC
devicedb start -conf $EDGE_CLIENT_RESOURCES/devicedb.conf
" > /usr/sbin/devicedb_edge
chmod +x /usr/sbin/devicedb_edge

# Add a script to clear devicedb edge and cloud
echo "#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $DEVICEDB_SRC
docker stop devicedb_devicedb-cloud_1
docker rm devicedb_devicedb-cloud_1
sudo systemctl stop devicedb_edge
rm -rf $EDGE_DATA_DIRECTORY/*
docker-compose up -d
sudo systemctl start devicedb_edge
" > /usr/sbin/clear_devicedb
chmod +x /usr/sbin/clear_devicedb

# Create a systemctl service that always run devicedb on reboot
echo "[Unit]
Description=DeviceDB Edge
[Service]
Type=simple
ExecStart=/usr/sbin/devicedb_edge
[Install]
WantedBy=multi-user.target
" > /etc/systemd/system/devicedb_edge.service
chmod +x /etc/systemd/system/devicedb_edge.service

# Reload systemctl with the new devicedb_edge service
systemctl daemon-reload

# Provide a default host for devicedb that will be configured on build
echo "127.0.0.1       unconfigured-devicedb-host"  >> /etc/hosts
