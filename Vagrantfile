# -*- mode: ruby -*-
# vi: set ft=ruby :

def set_up_base_box(config)
  config.vm.box = "ubuntu/xenial64"
end

def set_jenkins_network_configs(config)
  config.vm.network "private_network", ip: "172.28.128.10"
  config.vm.network "private_network", ip: "172.28.129.5", auto_config: false
  config.vm.network "private_network", ip: "172.28.130.2", auto_config: false
end

def set_testing_network_configs(config)
  config.vm.network "private_network", type: "dhcp"
  config.vm.network "private_network", ip: "172.28.128.1", auto_config: false
  config.vm.network "private_network", ip: "172.28.129.2", auto_config: false
end

def shared_config(config)
  config.vm.provision "file", source: "~/.ssh/id_rsa", destination: "~/.ssh/id_rsa"
  config.vm.provision "file", source: "~/.ssh/known_hosts", destination: "~/.ssh/known_hosts"
  config.vm.provision "file", source: "~/.ssh/id_rsa.pub", destination: "~/.ssh/id_rsa.pub"
  config.vm.provision :shell, path: "vagrant/provision.sh"
  config.vm.provision :reload
  config.vm.provision "file", source: "vagrant/build.sh", destination: "/tmp/build_maestro"
  config.vm.provision "shell", inline: "mv /tmp/build_maestro /usr/sbin/build_maestro; chmod +x /usr/sbin/build_maestro"
  config.ssh.extra_args = "-t"
  config.vm.provider :virtualbox do |vb|
    vb.customize [ "modifyvm", :id, "--memory", 2048 ]
  end
end

Vagrant.configure("2") do |config|

  config.vm.define :test, primary: true do |test|
    set_up_base_box test
    set_testing_network_configs test
    shared_config test
  end

  config.vm.define :jenkins_build, autostart: false do |jenkins_build|
    set_up_base_box jenkins_build
    set_jenkins_network_configs jenkins_build
    shared_config jenkins_build
  end

  config.vm.define :jenkins_coverity, autostart: false do |jenkins_coverity|
    set_up_base_box jenkins_coverity
    jenkins_coverity.disksize.size = "20GB"
    set_jenkins_network_configs jenkins_coverity
    shared_config jenkins_coverity
  end

end
