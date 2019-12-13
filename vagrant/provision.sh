#!/usr/bin/env bash

# Install prerequisite packages
apt-get update
apt-get install -y build-essential python wget git nodejs npm m4

# Download GO
wget https://dl.google.com/go/go1.13.5.linux-amd64.tar.gz
tar xvzf go1.13.5.linux-amd64.tar.gz
rm go1.13.5.linux-amd64.tar.gz
rm -rf /opt/go
mv go /opt/go

# Set GO environment variables
echo "export GIT_TERMINAL_PROMPT=1
export GOROOT=/opt/go
export GOPATH=/home/vagrant/work/gostuff
export GOBIN=/home/vagrant/work/gostuff/bin
export PATH=$PATH:/opt/go/bin:/home/vagrant/work/gostuff/bin
export MAESTRO_SRC=/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro
export LD_LIBRARY_PATH=/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro/vendor/github.com/armPelionEdge/greasego/deps/lib
" > /etc/profile.d/envvars.sh
. /etc/profile.d/envvars.sh

# Create directories for read/write of vagrant user
mkdir /var/maestro
chmod 777 /var/maestro

# Create a script to go to the maestro source and run maestro so maestro has access to its' configuration files
# Allows a user to run 'sudo maestro' and have everything work out
echo "#!/bin/bash -ue
. /etc/profile.d/envvars.sh
cd $MAESTRO_SRC
exec $GOBIN/maestro
" > /usr/sbin/maestro
chmod +x /usr/sbin/maestro

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
