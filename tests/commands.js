'use strict';

const shell = require('shelljs');
const fs = require('fs');
const assert = require('assert');
var async = require('async');

module.exports = class Commands {

    static get maestro_src()
    {
        return '/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro';
    }

    static get devicedb_src()
    {
        return '/home/vagrant/work/gostuff/src/github.com/armPelionEdge/devicedb';
    }

    static get maestro_certs()
    {
        return '/var/lib/maestro/certs';
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
        let maestro_certs = Commands.maestro_certs;
        return {
            start_maestro: 'vagrant ssh -c "sudo maestro 2>&1 | tee /tmp/maestro.log"',
            cat_maestro_debug_log: 'vagrant ssh -c "cat /tmp/maestro.log"',
            upload_vagrant: 'vagrant upload {{in_file}} {{out_file}}',
            kill_maestro: 'vagrant ssh -c "sudo pkill maestro"',
            check_dhcp: 'vagrant ssh -c "ps -aef | grep dhcp"',
            ip_addr: 'vagrant ssh -c "ip addr show {{interface}}"',
            ip_flush: 'vagrant ssh -c "sudo ip addr flush dev eth1; sudo ip addr flush dev eth2"',
            get_device_id: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/device_id"',
            get_site_id: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/site_id"',
            send_maestro_config: 'vagrant ssh -c "cat ' + devicedb_src + '/hack/certs/device_id"',

            check_file_existence: 'vagrant ssh -c "sudo ls {{filename}}"',
            remove_logs: 'vagrant ssh -c "echo | sudo tee {{filename}}"',
            get_logs: 'vagrant ssh -c "sudo tail -100 {{filename}}"',
            send_logs: 'vagrant ssh -c \'echo "test - Error" | systemd-cat -p err; echo "test - Info" | systemd-cat -p info; echo "test - Warn" | systemd-cat -p warning; echo "test - Debug" | systemd-cat -p debug\'',

            maestro_identity_get_cert: 'vagrant ssh -c "cat ' + maestro_certs + '/device_cert.pem"',
            maestro_identity_get_key: 'vagrant ssh -c "cat ' + maestro_certs + '/device_private_key.pem"',

            maestro_shell_get: 'vagrant ssh -c "sudo curl -XGET --unix-socket /tmp/maestroapi.sock http:{{url}}"',
            maestro_shell_put: 'vagrant ssh -c "sudo curl -XPUT --unix-socket /tmp/maestroapi.sock http:{{url}} -d \'{{payload}}\'"',
            maestro_shell_post: 'vagrant ssh -c "sudo curl -XPOST --unix-socket /tmp/maestroapi.sock http:{{url}} -d \'{{payload}}\'"',
            maestro_shell_delete: 'vagrant ssh -c "sudo curl -XDELETE --unix-socket /tmp/maestroapi.sock http:{{url}} -d \'{{payload}}\'"',

            devicedb_commit: 'vagrant ssh -c \'devicedb cluster put -site {{site_id}} -bucket lww -key vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_COMMIT_FLAG -value \'\\\'\'{"name":"vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_COMMIT_FLAG","body":"{\\\"config_commit\\\":true}"}\'\\\'',
            devicedb_put: 'echo \'devicedb cluster put -site {{site_id}} -bucket lww -key vagrant.{{relay_id}}.MAESTRO_NETWORK_CONFIG_ID -value \'\\\'\'{{payload}}\'\\\' | vagrant ssh'
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

    get_certs(cb)
    {
        this.run_shell(Commands.list.maestro_identity_get_key, function(result) {
            this.ctx.run_shell(Commands.list.maestro_identity_get_cert, function(result) {
                let output = {key: this.key.replace(/\\r/g, ''), cert: result.replace(/\\r/g, '')};
                this.cb(output);
            }.bind({ctx: this.ctx, cb: this.cb, key: result}));
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
     * Check an ip addr command to verify a valid IP address exists and if the interface is UP
     * @param {String} ip - IP address that should be available (or portion of an IP)
     * @param {callback} cb - Callback to run when done
     **/
    check_ip_addr(iface, ip, cb)
    {
        let command = Commands.list.ip_addr.replace('{{interface}}', iface);
        this.run_shell(command, function(stdout) {
            let arr = stdout.split('\n');
            var interface_up = false
            for (var i in arr) {
                let line = arr[i].trim();
                let words = line.split(' ');
                if( i==0 && words[7] === 'state' && words[8] === 'UP') {
                    interface_up = true
                }
                if (words.length >= 4 && words[0] === 'inet' && words[1].includes(ip)) {
                    this.cb(true,interface_up);
                    return;
                }
            }
            this.cb(false,interface_up);
        }.bind({ctx: this, cb: cb}));
    }

    check_log_existence(filenames, done)
    {
        async.eachSeries(filenames, function(key, next) {
            let command = Commands.list.check_file_existence.replace('{{filename}}', key);
            this.run_shell(command, function(stdout) {
                assert.equal(stdout, this.filename);
                if (!stdout.includes(this.filename))
                    this.next('File not found');
                this.next();
            }.bind({next: next, filename: key}));
        }.bind(this), done);
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

    save_maestro_log(count, cb)
    {
        if (!fs.existsSync('failureLogs')) {
            fs.mkdirSync('failureLogs');
        }
        this.run_shell(Commands.list.cat_maestro_debug_log, function(result) {
            fs.writeFile('failureLogs/maestro_log_fail' + this.count, result, function() {
                console.log('Maestro log for failure ' + this.count + ' saved to: ' + 'failureLogs/maestro_log_fail' + this.count);
                this.cb();
            }.bind(this));
        }.bind({count: count, cb: cb}));
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
     * Run a deviceDB command to put the payload into devicedb edge
     * @param {String} device_id - Device ID obtained from the gateway
     * @param {String} site_id - Site ID obtained from the gateway
     * @param {String} key - Key in the database to put the payload into
     * @param {Object} payload - Javascript dictionary containing the payload to send to devicedb
     * @param {callback} callback - Callback to run when the devicedb command is complete
     **/
    devicedb_command(device_id, site_id, key, payload, callback)
    {
        let body_string = JSON.stringify(payload);
        // Create the master view
        let view = {
            name: "vagrant.{{relay_id}}." + key,
            relay: "{{relay_id}}",
            body: body_string                
        };
        let json_view = JSON.stringify(view);
        // Formulate the command to send to devicedb
        let command = Commands.list.devicedb_put;
        command = command.replace('{{payload}}', json_view);
        command = command.replace(/{{relay_id}}/g, device_id);
        command = command.replace(/{{site_id}}/g, site_id);

        this.run_shell(command, function(result) {

            let command = Commands.list.devicedb_commit;
            command = command.replace(/{{relay_id}}/g, this.device_id);
            command = command.replace(/{{site_id}}/g, this.site_id);

            this.ctx.run_shell(command, this.callback);
        }.bind({ctx: this, device_id: device_id, site_id: site_id, callback: callback}));
    }

    maestro_api_verify_log_filters(target, filters, cb)
    {
        // Force all 4 levels into the logger to see if only the expected ones get outputted
        this.run_shell(Commands.list.send_logs, function(result) {
            setTimeout(function() {
                // Get the log file
                let command = Commands.list.get_logs;
                command = command.replace('{{filename}}', this.target);
                this.ctx.run_shell(command, function(output) {

                    // Check to see if the exepcted logs are in the log file
                    // Check to see if the non-expected logs are missing from the log file
                    let masterLevels = ['Info', 'Warn', 'Error', 'Debug'];
                    for (var level in masterLevels) {
                        if (this.filters.includes(masterLevels[level].toLowerCase()) || this.filters.includes('all')) {
                            assert(output.search('test - ' + masterLevels[level]) >= 0, 'Log level ' + masterLevels[level] + ' not found in output log');
                        } else {
                            assert(output.search('test - ' + masterLevels[level]) === -1, 'Log level ' + masterLevels[level] + ' found in output log but should not be');
                        }
                    }
                    this.cb();
                }.bind(this));
            }.bind(this), 5000);
        }.bind({ctx: this, cb: cb, filters: filters, target: target}));
    }

    maestro_api_delete_log_filter(target, filters, cb)
    {
        for (var filter in filters) {
            // Formulate payload
            let view = {Target: target, Levels: filters[filter], Tag: ''};
            let json_view = JSON.stringify(view);
            json_view = json_view.replace(/"/g, '\\\"');
            // Formulate command
            let command = Commands.list.maestro_shell_delete;
            command = command.replace('{{url}}', '/log/filter');
            command = command.replace('{{payload}}', json_view);
            // DELETE log target levels
            this.run_shell(command, function(result) {
                this(result);
            }.bind(cb));
        }
    }

    maestro_api_post_log_filter(target, filters, cb)
    {
        for (var filter in filters) {
            // Formulate payload
            let view = {Target: target, Levels: filters[filter], Tag: ''};
            let json_view = JSON.stringify(view);
            json_view = json_view.replace(/"/g, '\\\"');
            // Formulate command
            let command = Commands.list.maestro_shell_post;
            command = command.replace('{{url}}', '/log/filter');
            command = command.replace('{{payload}}', json_view);

            // PUT log target levels
            this.run_shell(command, function(result) {
                this(result);
            }.bind(cb));
        }
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