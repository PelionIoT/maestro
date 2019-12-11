# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/xenial64"
  config.vm.network "public_network", bridge: "en0: Wi-Fi (Wireless)"
  config.vm.provision "file", source: "~/.ssh/id_rsa", destination: "~/.ssh/id_rsa"
  config.vm.provision "file", source: "~/.ssh/known_hosts", destination: "~/.ssh/known_hosts"
  config.vm.provision :shell, path: "vagrant/provision.sh"
  config.vm.provision :reload
  config.vm.provision :shell, path: "vagrant/build.sh", run: 'always', privileged: false
end
