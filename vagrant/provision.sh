#!/usr/bin/env bash

# Install prerequisite packages
apt-get update
apt-get install -y build-essential wget git m4 python

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
export MAESTRO_LOGS=/var/log/maestro
export COVERITY_HOME=/home/vagrant/cov-analysis-linux64-2020.03
export PATH=$PATH:$COVERITY_HOME/bin
EOF
. /etc/profile.d/envvars.sh

# Create directories for read/write of vagrant user
mkdir /var/maestro
chmod 777 /var/maestro

# Create directories
mkdir -p $MAESTRO_LOGS
chmod 777 $MAESTRO_LOGS

# Create a script to go to the maestro source and run maestro so maestro has access to its' configuration files
# Allows a user to run 'sudo maestro' and have everything work out
cat <<'EOF' > /usr/sbin/maestro
#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $MAESTRO_SRC
exec $GOBIN/maestro
EOF
chmod +x /usr/sbin/maestro

ln -sf /vagrant/vagrant/build.sh /usr/sbin/build_maestro
chmod +x /usr/sbin/build_maestro

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

# Setup default email and name for github. Needed to apply patches
git config --global user.email "red@pelion.com"
git config --global user.name "Pelion Consulting Engineering"

# Reload systemctl with the new devicedb_edge service
systemctl daemon-reload
