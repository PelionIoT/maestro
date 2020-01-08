const assert = require('assert');
const shell = require('shelljs');
const fs = require('fs');

let maestro_src = '/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro';
let maestro_bin = '/home/vagrant/work/gostuff/bin/maestro';
let maestro_config_name = 'maestro.config';

let vm_name = 'default';
let command_maestro = 'vagrant ssh ' + vm_name + ' -c "sudo maestro"';
let command_vagrant_upload = 'vagrant upload {{in_file}} {{out_file}} ' + vm_name;
let command_kill = 'vagrant ssh ' + vm_name + ' -c "sudo pkill maestro"';
let command_dhcp_check = 'vagrant ' + vm_name + ' ssh -c "ps -aef | grep dhcp"';
let command_ip_addr = 'vagrant ssh ' + vm_name + ' -c "ip addr show eth"';
let command_ip_flush = 'vagrant ssh ' + vm_name + ' -c "sudo ip addr flush dev eth1; sudo ip addr flush dev eth2"';

let template_network_config = `
network:
    interfaces:
        - if_name: eth1
          existing: replace
          dhcpv4: true
          hw_addr: "{{ARCH_ETHERNET_MAC}}"
config_end: true
`;

let template_network_config_no_dhcp = `
network:
    interfaces:
        - if_name: eth1
          existing: replace
          dhcpv4: false
          ipv4_addr: 10.123.123.123
          ipv4_mask: 24
config_end: true
`;

let template_network_config_two_no_dhcp = `
network:
    interfaces:
        - if_name: eth1
          existing: replace
          dhcpv4: false
          ipv4_addr: 10.123.123.123
          ipv4_mask: 24
        - if_name: eth2
          existing: replace
          dhcpv4: false
          ipv4_addr: 10.123.123.124
          ipv4_mask: 24
config_end: true
`;

// Allow 30 seconds for the test to run, and provide 5 seconds for test cleanup
const timeout = 30000;
const timeout_cleanup = 5000;

/**
 * Check an ip addr command to verify a valid IP address exists
 * @param {Number} interface - Numerical representation of what ethernet interface you would like to check
 * @param {String} ip - IP address that should be available (or portion of an IP)
 * @param {callback} cb - Callback to run when done
 **/
function check_ip_addr(interface, ip, cb)
{
    shell.exec(command_ip_addr + interface, { silent: true}, function(code, stdout, stderr) {
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
 * @param {String} file_name - Name of the file on disk to save to
 * @param {String} config - Config content to write
 * @param {callback} cb - Callback to run when done
 **/
function save_config(file_name, config, cb)
{
    fs.writeFile(file_name, config, function(err) {
        assert.ifError(err);
        if (!err)
            this.cb(this.file_name);
    }.bind({ctx: this, cb: cb, file_name: file_name}));
}

/**
 * Upload a file to the vagrant VM
 * @param {String} file_name - Name of the file on disk to upload
 * @param {callback} cb - Callback when the upload has finished
 **/
function upload_to_vagrant(file_name, cb)
{
    // Insert file names into the command
    let command = command_vagrant_upload.replace('{{in_file}}', file_name);
    command = command.replace('{{out_file}}', maestro_src + '/' + file_name);
    // Upload config file to the vagrant VM
    shell.exec(command, { silent: true}, function(code, stdout, stderr) {
        assert.equal(code, 0);
        assert.equal(stderr, '');
        this.cb();
    }.bind({ctx: this, cb: cb}));
}

/**
 * Start maestro on the vagrant VM
 * @param {callback} done - Callback to the mocha test done callback
 * @param {callback} condition_cb - Callback to the function to test if the condition has been met
 **/
function start_maestro(done, condition_cb)
{
    // Start maestro up in the background, and listen for 
    var child = shell.exec(command_maestro, { async: true, silent: true });
    child.stdout.on('data', function(data) {
        if (this.cb(data)) {
            shell.exec(command_kill, { silent: true});
            this.done();
        }
    }.bind({ctx: this, done: done, cb: condition_cb}));
}

/**
 * Workflow that every test should run. Use this within an in block
 * @param {String} config - Configuration to use for maestro
 * @param {callback} done - Callback to the mocha test done callback
 * @param {callback} checker_cb - Callback to run when a new output is received from maestro
 **/
function maestro_workflow(config, done, checker_cb)
{
    save_config(maestro_config_name, config, function(file_name) {
        upload_to_vagrant(file_name, function() {
            start_maestro(this.done, this.checker_cb);
        }.bind(this));
    }.bind({ctx: this, done: done, checker_cb: checker_cb}));
}


/**
 * Networking tests
 **/
describe('Networking', function() {

    /**
     * DHCP tests
     **/
    describe('DCHP', function() {

        afterEach(function() {
            this.timeout(timeout_cleanup);
            shell.exec(command_kill, { silent: true});
        });

        it('should enable DCHP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            shell.exec(command_ip_flush, { silent: true});
            maestro_workflow(template_network_config, done, function(data) {
                return data.includes('DHCP') && data.includes('Lease acquired');
            });
        });

        it('should now have a DHCP enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            check_ip_addr(1, '172.28.128.', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with an IP address prefixed with 172.28.128.xxx');
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            shell.exec(command_ip_flush, { silent: true});
            maestro_workflow(template_network_config_no_dhcp, done, function(data) {
                return data.includes('Static address set on eth1 of 10.123.123.123');
            });
        });

        it('should now have a static enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP for eth1 and eth2 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            shell.exec(command_ip_flush, { silent: true});
            maestro_workflow(template_network_config_two_no_dhcp, done, function(data) {
                return data.includes('Static address set on eth1 of 10.123.123.123') || data.includes('Static address set on eth1 of 10.123.123.124');
            });
        });

        it('should now have 2 static enabled IP addresses', function(done) {
            this.timeout(timeout);
            check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                check_ip_addr(2, '10.123.123.124', function(contains_ip) {
                    assert(contains_ip, 'Interface eth2 not set with IP address 10.123.123.124');
                    this.done();
                }.bind({ctx: this.ctx, done: this.done}));
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP when no networking configuration is specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            shell.exec(command_ip_flush, { silent: true});
            maestro_workflow('', done, function(data) {
                return data.includes('Static address set on');
            });
        });

        it('should now have a static enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                this.done();
            }.bind({ctx: this, done: done}));
        });
    });
});