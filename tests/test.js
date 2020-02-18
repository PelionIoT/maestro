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
     * Generic tests
     **/
    describe('Generic', function() {

        it('should continue running when only "config_end: true" is specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.maestro_workflow('config_end: true',

                // Function to run when done with the workflow
                done,

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    return data.includes('[ERROR] Error starting networking subsystem!');
                }
            );
        });

        it('should continue running when nothing is specified in the configuration file provided to maestro', function(done) {
            this.timeout(timeout);
            maestro_commands.maestro_workflow('',

                // Function to run when done with the workflow
                done,

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    return data.includes('[ERROR] Error starting networking subsystem!');
                }
            );
        });
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
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', '172.28.128.', function(contains_ip) {
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
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view), 

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', '10.123.123.123', function(contains_ip) {
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
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', this.interfaces[0].ipv4_addr, function(contains_ip) {
                        assert(contains_ip, 'Interface ' + this.interfaces[0].if_name + ' not set with IP address ' + this.interfaces[0].ipv4_addr);
                        // Compare the IP address of the VM to verify maestro set the IP properly
                        maestro_commands.check_ip_addr('eth2', this.interfaces[1].ipv4_addr, function(contains_ip) {
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
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

        before(function(done) {
            this.timeout(timeout);
            this.done = done;

            // Get the site ID to connect to devicedb properly
            maestro_commands.get_certs(function(certs) {
                this.certs = certs;
                this.done();
            }.bind(this));
        });

        beforeEach(function(done) {
            this.timeout(timeout);

            let command = Commands.list.remove_logs.replace('{{filename}}', '/var/log/maestro/maestro.log*');
            maestro_commands.run_shell(command, done);
        });

        it('should enable symphony and cloud logging', function(done) {
            this.timeout(timeout);

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', dhcpv4: true}
                    ],
                },
                sys_stats: {
                    vm_stats: {every: '15s', name: 'vm'},
                    disk_stats: {every: '15s', name: 'disk'}
                },
                symphony: {
                    sys_stats_count_threshold: 15,
                    sys_stats_time_threshold: 120000,
                    client_cert: this.certs.cert,
                    client_key: this.certs.key,
                    host: 'gateways.mbedcloudstaging.net'
                },
                targets: [
                    {name: 'toCloud', format_time: "\"timestamp\":%ld%03d, ", filters: [{levels: 'all'}]}
                ],
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                done,

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    // Compare the log to see if the DHCP lease was acquired
                    return data.includes('RMI client workers started');
                }
            );
        });

        it('should enable file logging with 2 files to rotate', function(done) {
            this.timeout(timeout);
            this.done = done;

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', dhcpv4: true}
                    ],
                },
                sys_stats: {
                    vm_stats: {every: '15s', name: 'vm'},
                    disk_stats: {every: '15s', name: 'disk'}
                },
                targets: [
                    {
                        file: '/var/log/maestro/maestro.log',
                        format_time: "\"timestamp\":%ld%03d, ",
                        filters: [{levels: 'all'}],
                        rotate: {
                            max_files: 2,
                            max_file_size: 1000000,
                            max_total_size: 2200000,
                            rotate_on_start: true
                        }
                    }
                ],
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    maestro_commands.check_log_existence(['/var/log/maestro/maestro.log'], this);
                }.bind(done),

                // Function to run whenever a new stream of data comes accross STDOUT
                function(data) {
                    // Compare the log to see if the DHCP lease was acquired
                    return data.includes('Needs rotate...');
                }.bind(view.targets[0].file)
            );
        });

    });
});

function maestro_api_set_ip_address(ctx, interface, ip_address)
{
    let view = [{
        dhcpv4: false,
        if_name: interface,
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
            assert(contains_ip, 'Interface ' + this.interface + ' not set with IP address ' + this.ip_address);
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
            },
            config_end: true
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
            maestro_api_set_ip_address(this, 'eth1', '10.234.234.234');
        });

        it('should change the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_api_set_ip_address(this, 'eth2', '10.229.229.229');
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
            maestro_commands.check_ip_addr(this.interface, this.ip_address, function(contains_ip) {
                assert(contains_ip, 'Interface ' + this.interface + ' not set with IP address ' + this.ip_address);
                this.ctx.done();
            }.bind(this));
        }.bind(this), 5000);
    }.bind({interface: interface, ip_address: ip_address, ctx: ctx}));
}

describe('DeviceDB', function() {

    before(function(done) {
        this.timeout(timeout * 2);
        this.done = done;
        // Flush the IP table to get a clean slate
        maestro_commands.run_shell(Commands.list.ip_flush, function() {

            // Get the site ID to connect to devicedb properly
            maestro_commands.get_site_id(function(site_id) {
                // Check for invalid site IDs
                assert.notEqual(site_id, '', 'Site ID not obtained!');
                this.site_id = site_id;

                // Get the relay ID to connect to devicedb properly
                maestro_commands.get_device_id(function(device_id) {
                    // Check for invalid site IDs
                    assert.notEqual(device_id, '', 'Device/Relay ID not obtained!');
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
                        },
                        config_end: true
                    };

                    maestro_commands.maestro_workflow(YAML.stringify(view), null, null);
                    setTimeout(this.done, timeout);
                }.bind(this));
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

