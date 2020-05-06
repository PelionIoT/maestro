# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"
  config.vm.network "private_network", type: "dhcp"
  config.vm.network "private_network", ip: "172.28.128.1", auto_config: false
  config.vm.network "private_network", ip: "172.28.129.2", auto_config: false
  config.vm.provision "file", source: "~/.ssh/id_rsa", destination: "~/.ssh/id_rsa"
  config.vm.provision "file", source: "~/.ssh/known_hosts", destination: "~/.ssh/known_hosts"
  config.vm.provision "file", source: "~/.ssh/id_rsa.pub", destination: "~/.ssh/id_rsa.pub"
  config.vm.provision :shell, path: "vagrant/provision.sh"
  config.vm.provision :reload
  config.vm.provision "file", source: "vagrant/build.sh", destination: "/tmp/build_maestro"
  config.vm.provision "shell", inline: "mv /tmp/build_maestro /usr/sbin/build_maestro; chmod +x /usr/sbin/build_maestro"
  config.vm.provision "file", source: "vagrant/docker-compose.yaml", destination: "/tmp/docker-compose.yaml"
  config.ssh.extra_args = "-t"
end
