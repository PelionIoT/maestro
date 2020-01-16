'use strict';

const shell = require('shelljs');
const fs = require('fs');
const assert = require('assert');

module.exports = class Commands {

    static get maestro_src()
    {
        return '/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro';
    }

    static get devicedb_src()
    {
        return '/home/vagrant/work/gostuff/src/github.com/armPelionEdge/devicedb';
    }

    static get go_bin()
    {
        return '/home/vagrant/work/gostuff/bin';
    }

    static get config_name()
    {
        return 'maestro.config';
    }

    static get list()
    {
        let devicedb_src = Commands.devicedb_src;
        return {
            start_maestro: 'vagrant ssh -c "sudo maestro 2>&1 | tee /tmp/maestro.log"',
            upload_vagrant: 'vagrant upload {{in_file}} {{out_file}}',
            kill_maestro: 'vagrant ssh -c "sudo pkill maestro"',
            check_dhcp: 'vagrant ssh -c "ps -aef | grep dhcp"',
            ip_addr: 'vagrant ssh -c "ip addr show eth"',
            ip_flush: 'vagrant ssh -c "sudo ip addr flush dev eth1; sudo ip addr flush dev eth2"',
            get_device_id: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/device_id"',
            get_site_id: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/site_id"',
            send_maestro_config: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/device_id"',
            maestro_shell_get_iface: 'vagrant ssh -c "sudo curl -XGET --unix-socket /tmp/maestroapi.sock http:/net/interfaces"',
            maestro_shell_put_iface: 'vagrant ssh -c "sudo curl -XPUT --unix-socket /tmp/maestroapi.sock http:/net/interfaces -d \'{{payload}}\'"',
            devicedb_commit: 'vagrant ssh -c \'devicedb cluster put -site {{site_id}} -bucket lww -key vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_COMMIT_FLAG -value \'\\\'\'{"name":"vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_COMMIT_FLAG","body":"{\\\"config_commit\\\":true}"}\'\\\'',
            devicedb_put_iface: 'echo \'devicedb cluster put -site {{site_id}} -bucket lww -key vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_ID -value \'\\\'\'{{payload}}\'\\\' | vagrant ssh'
        };
    }

    constructor() {
    }

    run_shell(command, cb)
    {
        shell.exec(command, {silent: true}, function(code, stdout, stderr) {
            if (this.cb)
                this.cb(stdout.trim());
        }.bind({ctx: this, cb: cb}));
    }

    get_device_id(cb)
    {
        this.run_shell(Commands.list.get_device_id, cb);
    }

    get_site_id(cb)
    {
        this.run_shell(Commands.list.get_site_id, cb);
    }

    /**
     * Check an ip addr command to verify a valid IP address exists
     * @param {String} ip - IP address that should be available (or portion of an IP)
     * @param {callback} cb - Callback to run when done
     **/
    check_ip_addr(iface, ip, cb)
    {
        this.run_shell(Commands.list.ip_addr + iface, function(stdout) {
            let arr = stdout.split('\n');
            for (var i in arr) {
                let line = arr[i].trim();
                let words = line.split(' ');
                if (words.length >= 4 && words[0] === 'inet' && words[1].includes(ip)) {
                    this.cb(true);
                    return;
                }
            }
            this.cb(false);
        }.bind({ctx: this, cb: cb}));
    }

    /**
     * Save a config file to disk
     * @param {String} config - Config content to write
     * @param {callback} cb - Callback to run when done
     **/
    save_config(config, cb)
    {
        fs.writeFile(Commands.config_name, config, function(err) {
            assert.ifError(err);
            if (!err)
                this.cb(Commands.config_name);
        }.bind({ctx: this, cb: cb}));
    }

    /**
     * Upload a file to the vagrant VM
     * @param {callback} cb - Callback when the upload has finished
     **/
    upload_to_vagrant(cb)
    {
        // Insert file names into the command
        let command = Commands.list.upload_vagrant;
        command = command.replace('{{in_file}}', Commands.config_name);
        command = command.replace('{{out_file}}', Commands.maestro_src + '/' + Commands.config_name);
        // Upload config file to the vagrant VM
        this.run_shell(command, cb);
    }

    /**
     * Start maestro on the vagrant VM
     * @param {callback} done - Callback to the mocha test done callback
     * @param {callback} condition_cb - Callback to the function to test if the condition has been met
     **/
    start_maestro(done, condition_cb)
    {
        // Start maestro up in the background, and listen for 
        var child = shell.exec(Commands.list.start_maestro, { async: true, silent: true });
        child.stdout.on('data', function(data) {
            if (this.cb && this.cb(data)) {
                this.ctx.run_shell(Commands.list.kill_maestro, function() {
                    if (this.done)
                        this.done();
                }.bind(this));
            }
        }.bind({ctx: this, done: done, cb: condition_cb}));
    }

    /**
     * Workflow that every test should run. Use this within an in block
     * @param {String} config - Configuration to use for maestro
     * @param {callback} done - Callback to the mocha test done callback
     * @param {callback} checker_cb - Callback to run when a new output is received from maestro
     **/
    maestro_workflow(config, done, checker_cb)
    {
        this.save_config(config, function() {
            this.ctx.upload_to_vagrant(function() {
                this.ctx.start_maestro(this.done, this.checker_cb);
            }.bind(this));
        }.bind({ctx: this, done: done, checker_cb: checker_cb}));
    }

}