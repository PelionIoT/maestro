const assert = require('assert');
const YAML = require('json2yaml');

var Commands = require('./commands.js');

let maestro_commands = new Commands();

// Allow 30 seconds for the test to run, and provide 5 seconds for test cleanup
const timeout = 30000;
const timeout_cleanup = 10000;

describe('Maestro Config', function() {

    afterEach(function(done) {
        this.timeout(timeout_cleanup);
        maestro_commands.run_shell(Commands.list.kill_maestro, done);
    });

    /**
     * DHCP tests
     **/
    describe('DHCP', function() {

        it('should enable DHCP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            // Flush the IP table to get a clean slate
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', dhcpv4: true}
                    ],
                }
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr(1, '172.28.128.', function(contains_ip) {
                        assert(contains_ip, 'Interface eth1 not set with an IP address prefixed with 172.28.128.xxx');
                        this();
                    }.bind(this));
                }.bind(done),

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    // Compare the log to see if the DHCP lease was acquired
                    return data.includes('DHCP') && data.includes('Lease acquired');
                }
            );
        });

        it('should disable DHCP for eth1 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            // Flush the IP table to get a clean slate
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', dhcpv4: false, ipv4_addr: '10.123.123.123', ip_mask: 24}
                    ],
                }
            };

            maestro_commands.maestro_workflow(YAML.stringify(view), 

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr(1, '10.123.123.123', function(contains_ip) {
                        assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                        this();
                    }.bind(this));
                }.bind(done),

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    // Compare the log to see if the DHCP lease was acquired
                    return data.includes('Static address set on ' + this.if_name + ' of ' + this.ipv4_addr);
                }.bind(view.network.interfaces[0])
            );
        });

        it('should disable DHCP for eth1 and eth2 when specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            // Flush the IP table to get a clean slate
            maestro_commands.run_shell(Commands.list.ip_flush, null);

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', dhcpv4: false, ipv4_addr: '10.99.99.99', ip_mask: 24},
                        {if_name: 'eth2', existing: 'replace', dhcpv4: false, ipv4_addr: '10.88.88.88', ip_mask: 24}
                    ],
                }
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr(1, this.interfaces[0].ipv4_addr, function(contains_ip) {
                        assert(contains_ip, 'Interface ' + this.interfaces[0].if_name + ' not set with IP address ' + this.interfaces[0].ipv4_addr);
                        // Compare the IP address of the VM to verify maestro set the IP properly
                        maestro_commands.check_ip_addr(2, this.interfaces[1].ipv4_addr, function(contains_ip) {
                            assert(contains_ip, 'Interface ' + this.interfaces[1].if_name + ' not set with IP address ' + this.interfaces[1].if_name);
                            this.done();
                        }.bind(this));
                    }.bind(this));
                }.bind({done: done, interfaces: view.network.interfaces}),

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    return data.includes('Static address set on ' + this[1].if_name + ' of ' + this[1].ipv4_addr);
                }.bind(view.network.interfaces)
            );
        });

        it('should disable DHCP when no networking configuration is specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            this.skip(); // Currently doesn't support not having a network configuration
            maestro_commands.run_shell(Commands.list.ip_flush, null);
            maestro_commands.maestro_workflow('config_end: true',

                // Function to run when done with the workflow
                done,

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    return data.includes('Static address set on');
                }
            );
        });
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

    });
});

function maestro_api_set_ip_address(ctx, interface, ip_address)
{
    let view = [{
        dhcpv4: false,
        if_name: "eth" + interface,
        ipv4_addr: ip_address,
        ipv4_mask: 24,
        clear_addresses: true
    }];
    let json_view = JSON.stringify(view);
    json_view = json_view.replace(/"/g, '\\\"');

    let command = Commands.list.maestro_shell_put_iface;
    command = command.replace('{{payload}}', json_view);

    maestro_commands.run_shell(command, function(result) {
        maestro_commands.check_ip_addr(this.interface, this.ip_address, function(contains_ip) {
            assert(contains_ip, 'Interface eth' + this.interface + ' not set with IP address ' + this.ip_address);
            this.ctx.done();
        }.bind(this));
    }.bind({interface: interface, ip_address: ip_address, ctx: ctx}));
}

describe('Maestro API', function() {

    before(function(done) {
        this.timeout(timeout);
        // Flush the IP table to get a clean slate
        maestro_commands.run_shell(Commands.list.ip_flush, null);

        // Create the config
        this.view = {
            network: {
                interfaces: [
                    {if_name: 'eth1', existing: 'replace', dhcpv4: false, ipv4_addr: '10.99.99.99', ip_mask: 24},
                    {if_name: 'eth2', existing: 'replace', dhcpv4: false, ipv4_addr: '10.88.88.88', ip_mask: 24}
                ]
            }
        };

        maestro_commands.maestro_workflow(YAML.stringify(this.view), null, null);
        setTimeout(done, 5000);
    });

    after(function(done) {
        this.timeout(timeout_cleanup);
        maestro_commands.run_shell(Commands.list.kill_maestro, done);
    });

    /**
     * DHCP tests
     **/
    describe('DHCP', function() {
        it('should retrieve the active maestro config', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.run_shell(Commands.list.maestro_shell_get_iface, function(active_iface) {
                let active_config = JSON.parse(active_iface);

                let iface_arr = this.view.network.interfaces;
                // Loop through all the interfaces and verify they match the active maestro.config
                for (var interface in iface_arr) {
                    // Look for the specific interface
                    let filtered_arr = active_config.filter(function (el) {
                        return el.StoredIfconfig.if_name === this.if_name;
                    }.bind(iface_arr[interface]));
                    // Check the IP address
                    assert.equal(filtered_arr[0].StoredIfconfig.ipv4_addr, iface_arr[interface].ipv4_addr);
                }

                this.done();
            }.bind(this));
        });

        it('should change the IP address of the first network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_api_set_ip_address(this, 1, '10.234.234.234');
        });

        it('should change the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_api_set_ip_address(this, 2, '10.229.229.229');
        });
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

    });
});

function devicedb_set_ip_address(ctx, interface, ip_address)
{
    // Base view but needs to contain ALL of the interfaces
    let body = {
        interfaces: [{
            if_name: "eth1",
        },{
            if_name: "eth2"
        }]
    };
    // Find the interface that we need to modify
    var index = body.interfaces.findIndex(function (el) {
        return el.if_name == this;
    }.bind(interface));
    // Change the specific interface we are interested in
    if (index !== -1) {
        body.interfaces[index] = {
            if_name: interface,
            dhcpv4: false,
            ipv4_addr: ip_address,
            ipv4_mask: 24,
            clear_addresses: true,
            existing: "override"
        };
    }

    let key = 'MAESTRO_NETWORK_CONFIG_ID';
    maestro_commands.devicedb_command(ctx.device_id, ctx.site_id, key, body, function(output) {
        setTimeout(function() {
            maestro_commands.check_ip_addr(parseInt(this.interface.replace('eth', '')), this.ip_address, function(contains_ip) {
                assert(contains_ip, 'Interface ' + this.interface + ' not set with IP address ' + this.ip_address);
                this.ctx.done();
            }.bind(this));
        }.bind(this), 5000);
    }.bind({interface: interface, ip_address: ip_address, ctx: ctx}));
}

describe('DeviceDB', function() {

    before(function(done) {
        this.timeout(timeout);
        this.done = done;
        // Flush the IP table to get a clean slate
        maestro_commands.run_shell(Commands.list.ip_flush, null);

        // Get the site ID to connect to devicedb properly
        maestro_commands.get_site_id(function(site_id) {
            this.site_id = site_id;

            // Get the relay ID to connect to devicedb properly
            maestro_commands.get_device_id(function(device_id) {
                this.device_id = device_id;

                // Create the config
                let view = {
                    network: {
                        interfaces: [
                            {if_name: 'eth1', existing: 'replace', dhcpv4: false, ipv4_addr: '10.77.77.77', ip_mask: 24},
                            {if_name: 'eth2', existing: 'replace', dhcpv4: false, ipv4_addr: '10.66.66.66', ip_mask: 24}
                        ]
                    },
                    devicedb_conn_config: {
                        devicedb_uri: 'https://' + device_id + ':9090',
                        devicedb_prefix: 'vagrant',
                        devicedb_bucket: 'lww',
                        relay_id: device_id,
                        ca_chain: Commands.devicedb_src + '/hack/certs/myCA.pem'
                    }
                };

                maestro_commands.maestro_workflow(YAML.stringify(view), null, null);
                setTimeout(this.done, 15000);
            }.bind(this));
        }.bind(this));
    });

    after(function(done) {
        this.timeout(timeout_cleanup);
        maestro_commands.run_shell(Commands.list.kill_maestro, done);
    });

    /**
     * DHCP tests
     **/
    describe('DHCP', function() {
        it('should set the IP address of the first network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            devicedb_set_ip_address(this, 'eth1', '10.122.122.122');
        });

        it('should change the IP address of the first network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            devicedb_set_ip_address(this, 'eth1', '10.234.234.234');
        });

        it('should set the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            devicedb_set_ip_address(this, 'eth1', '10.125.125.125');
        });

        it('should change the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            devicedb_set_ip_address(this, 'eth2', '10.222.222.222');
        });
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

    });
});

