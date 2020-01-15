const assert = require('assert');

var Config = require('./config.js');
var Commands = require('./commands.js');

let maestro_config = new Config();
let maestro_commands = new Commands();

// Allow 30 seconds for the test to run, and provide 5 seconds for test cleanup
const timeout = 30000;
const timeout_cleanup = 5000;

describe('Maestro Config', function() {

    /**
     * DHCP tests
     **/
    describe('DCHP', function() {

        before(function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.get_device_id(function(device_id) {
                this.ctx.device_id = device_id;
                this.done();
            }.bind({ctx: this, done: done}));
        });

        afterEach(function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.run_shell(Commands.list.kill_maestro, function() {
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should enable DCHP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                device_id: this.device_id,
                interfaces: [{interface_name: 'eth1', dhcp: true}]
            };
            let config = maestro_config.render(view);

            maestro_commands.maestro_workflow(config, done, function(data) {
                return data.includes('DHCP') && data.includes('Lease acquired');
            });
        });

        it('should now have a DHCP enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.check_ip_addr(1, '172.28.128.', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with an IP address prefixed with 172.28.128.xxx');
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                device_id: this.device_id,
                interfaces: [{interface_name: 'eth1', dhcp: false, ip_address: '10.123.123.123', ip_mask: 24}]
            };
            let config = maestro_config.render(view);

            maestro_commands.maestro_workflow(config, done, function(data) {
                return data.includes('Static address set on eth1 of 10.123.123.123');
            });
        });

        it('should now have a static enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP for eth1 and eth2 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                device_id: this.device_id,
                interfaces: [
                    {interface_name: 'eth1', dhcp: false, ip_address: '10.123.123.123', ip_mask: 24},
                    {interface_name: 'eth2', dhcp: false, ip_address: '10.123.123.124', ip_mask: 24}
                ]
            };
            let config = maestro_config.render(view);

            maestro_commands.maestro_workflow(config, done, function(data) {
                return data.includes('Static address set on eth1 of 10.123.123.123') || data.includes('Static address set on eth1 of 10.123.123.124');
            });
        });

        it('should now have 2 static enabled IP addresses', function(done) {
            this.timeout(timeout);
            maestro_commands.check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                maestro_commands.check_ip_addr(2, '10.123.123.124', function(contains_ip) {
                    assert(contains_ip, 'Interface eth2 not set with IP address 10.123.123.124');
                    this.done();
                }.bind({ctx: this.ctx, done: this.done}));
            }.bind({ctx: this, done: done}));
        });

        it('should disable DCHP when no networking configuration is specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.ip_flush, null);
            maestro_commands.maestro_workflow('', done, function(data) {
                return data.includes('Static address set on');
            });
        });

        it('should now have a static enabled IP address', function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                this.done();
            }.bind({ctx: this, done: done}));
        });
    });
});

describe('Maestro-Shell', function() {

    /**
     * DHCP tests
     **/
    describe('DCHP', function() {

        before(function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.ip_flush, null);
            maestro_commands.get_device_id(function(device_id) {
                // Create the config
                let view = {
                    device_id: device_id,
                    interfaces: [
                        {interface_name: 'eth1', dhcp: false, ip_address: '10.123.123.123', ip_mask: 24},
                        {interface_name: 'eth2', dhcp: false, ip_address: '10.123.123.124', ip_mask: 24}
                    ]
                };
                let config = maestro_config.render(view);
                maestro_commands.maestro_workflow(config, null, null);
                setTimeout(this.done, 5000);
            }.bind({ctx: this, done: done}));
        });

        after(function(done) {
            this.timeout(timeout_cleanup);
            maestro_commands.run_shell(Commands.list.kill_maestro, function() {
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should retrieve the active maestro config', function(done) {
            this.timeout(timeout);
            maestro_commands.run_shell(Commands.list.maestro_shell_get_iface, function(active_iface) {
                let active_config = JSON.parse(active_iface);
                var eth1Array = active_config.filter(function (el) {
                    return el.StoredIfconfig.if_name === 'eth1';
                });
                assert.equal(eth1Array[0].StoredIfconfig.ipv4_addr, '10.123.123.123');
                var eth2Array = active_config.filter(function (el) {
                    return el.StoredIfconfig.if_name === 'eth2';
                });
                assert.equal(eth2Array[0].StoredIfconfig.ipv4_addr, '10.123.123.124');
                this.done();
            }.bind({ctx: this, done: done}));
        });

        it('should change the IP address of one of the network adapters', function(done) {
            this.timeout(timeout);

            let view = [{
                dhcpv4enabled: false,
                ifname: "eth1",
                ipv4addr: "10.234.234.234",
                ipv4mask: 24,
                clearaddresses: false
            }];
            let json_view = JSON.stringify(view);

            let command = Commands.list.maestro_shell_put_iface;
            command = command.replace('{{payload}}', json_view);

            maestro_commands.run_shell(command, function(result) {
                maestro_commands.check_ip_addr(1, '10.234.234.234', function(contains_ip) {
                    assert(contains_ip, 'Interface eth1 not set with IP address 10.234.234.234');
                    this.done();
                }.bind(this));
            }.bind({ctx: this, done: done}));
        });

        it('should change the IP address of 2 different network adapters', function(done) {
            this.timeout(timeout);

            let view = [{
                dhcpv4enabled: false,
                ifname: "eth1",
                ipv4addr: "10.432.432.432",
                ipv4mask: 24,
                clearaddresses: false
            },{
                dhcpv4enabled: false,
                ifname: "eth2",
                ipv4addr: "10.155.155.155",
                ipv4mask: 24,
                clearaddresses: false
            }];
            let json_view = JSON.stringify(view);

            let command = Commands.list.maestro_shell_put_iface;
            command = command.replace('{{payload}}', json_view);

            maestro_commands.run_shell(command, function(result) {
                maestro_commands.check_ip_addr(1, '10.234.234.234', function(contains_ip) {
                    assert(contains_ip, 'Interface eth1 not set with IP address 10.234.234.234');
                    maestro_commands.check_ip_addr(2, '10.155.155.155', function(contains_ip) {
                        assert(contains_ip, 'Interface eth2 not set with IP address 10.155.155.155');
                        this.done();
                    }.bind(this));
                }.bind(this));
            }.bind({ctx: this, done: done}));
        });

    });
});