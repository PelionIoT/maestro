const assert = require('assert');
const YAML = require('json2yaml');

var Commands = require('./commands.js');

let maestro_commands = new Commands();

// Allow 30 seconds for the test to run, and provide 5 seconds for test cleanup
const timeout = 30000;
const timeout_cleanup = 10000;

let failure_count = 0;

describe('Maestro Config', function() {

    afterEach(function(done) {
        this.timeout(timeout_cleanup);
        this.done = done;
        maestro_commands.run_shell(Commands.list.kill_maestro, function() {
            if (!this.currentTest.state.includes('passed')) {
                maestro_commands.save_maestro_log(failure_count++, this.done);
            } else {
                this.done();
            }
        }.bind(this));
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
                        {if_name: 'eth1', clear_addresses: true, existing: 'replace', dhcpv4: true}
                    ],
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', '172.28.128.', function(contains_ip, interface_up) {
                        assert(contains_ip, 'Interface eth1 not set with an IP address prefixed with 172.28.128.xxx');
                        assert(interface_up, 'Interface eth1 not UP');
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
                        {if_name: 'eth1', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.123.123.123', ipv4_mask: 24}
                    ],
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view), 

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', '10.123.123.123', function(contains_ip, interface_up) {
                        assert(contains_ip, 'Interface eth1 not set with IP address 10.123.123.123');
                        assert(interface_up, 'Interface eth1 not UP');
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
                        {if_name: 'eth1', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.99.99.99', ipv4_mask: 24},
                        {if_name: 'eth2', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.88.88.88', ipv4_mask: 24}
                    ],
                },
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

                // Function to run when done with the workflow
                function() {
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_ip_addr('eth1', this.interfaces[0].ipv4_addr, function(contains_ip, interface_up) {
                        assert(contains_ip, 'Interface ' + this.interfaces[0].if_name + ' not set with IP address ' + this.interfaces[0].ipv4_addr);
                        assert(interface_up, 'Interface eth1 not UP');
                        // Compare the IP address of the VM to verify maestro set the IP properly
                        maestro_commands.check_ip_addr('eth2', this.interfaces[1].ipv4_addr, function(contains_ip,interface_up) {
                            assert(contains_ip, 'Interface ' + this.interfaces[1].if_name + ' not set with IP address ' + this.interfaces[1].if_name);
                            assert(interface_up, 'Interface eth2 not UP');
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
                        {if_name: 'eth1', clear_addresses: true, existing: 'replace', dhcpv4: true}
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
                        {if_name: 'eth1', clear_addresses: true, existing: 'replace', dhcpv4: true}
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

    /**
     * Service control tests
     **/
    describe('ServiceCtl', function() {

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

            maestro_commands.get_service_to_default_state('exampleservice.service');
            maestro_commands.get_service_to_default_state('exampleservice1.service');
        });

        it('should enable/start/disable/start_on_boot service', function(done) {
            this.timeout(timeout);

            // Create the config
            let view = {
                network: {
                    interfaces: [
                        {if_name: 'eth1', clear_addresses: true, existing: 'replace', dhcpv4: true}
                    ],
                },
                services: [
                    {name: "exampleservice.service", start_on_boot: false, enable_service: true, start_service: true},
                    {name: "exampleservice1.service", start_on_boot: true, disable_service: true}
                ],
                config_end: true
            };

            maestro_commands.maestro_workflow(YAML.stringify(view),

            // Function to run when done with the workflow
            function() {
                // Compare the IP address of the VM to verify maestro set the IP properly
                maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                    assert(!not_found, 'Service exampleservice not found');
                    assert(is_running, 'Service exampleservice is not running');
                    assert(is_enabled, 'Service exampleservice is not enabled');
                    // Compare the IP address of the VM to verify maestro set the IP properly
                    maestro_commands.check_service_state('exampleservice1.service', function(not_found, is_enabled, is_running) {
                        assert(!not_found, 'Service exampleservice not found');
                        assert(is_running, 'Service exampleservice is not running');
                        assert(!is_enabled, 'Service exampleservice is not disabled');
                        this.done();
                    }.bind(this));
                }.bind(this));
            }.bind({done: done}),

            // Function to run whenever a new stream of data comes accross STDOUT
            function(data) {
                return data.includes('Enabling service exampleservice.service') && data.includes('Started service exampleservice.service') && data.includes('Disabling service exampleservice1.service') && data.includes('Started service exampleservice1.service');
            }.bind()
        );
        });

    });
});

describe('Maestro API', function() {

    before(function(done) {
        this.timeout(timeout);
        this.done = done;
        // Flush the IP table to get a clean slate
        maestro_commands.run_shell(Commands.list.ip_flush, function() {

            // Create the config
            this.view = {
                sysLogSocket: '/run/systemd/journal/syslog',
                network: {
                    interfaces: [
                        {if_name: 'eth1', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.99.99.99', ipv4_mask: 24},
                        {if_name: 'eth2', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.88.88.88', ipv4_mask: 24}
                    ]
                },
                sys_stats: {
                    vm_stats: {every: '15s', name: 'vm'},
                    disk_stats: {every: '15s', name: 'disk'}
                },
                targets: [
                    {name: 'toCloud', format_time: "\"timestamp\":%ld%03d, ", filters: [{levels: 'all'}]},
                    {file: '/var/log/maestro/maestro.log', name: '/var/log/maestro/maestro.log', format_time: "\"timestamp\":%ld%03d, ", filters: [{levels: 'error'}]},
                ],
                config_end: true
            };
            this.file_target = this.view.targets[1].file;
            this.file_target_default_filter = this.view.targets[1].filters[0].levels;
            maestro_commands.maestro_workflow(YAML.stringify(this.view), null, null);
            setTimeout(this.done, 10000);
        }.bind(this));
    });

    after(function(done) {
        this.timeout(timeout_cleanup);
        maestro_commands.run_shell(Commands.list.kill_maestro, done);
    });

    afterEach(function(done) {
        this.timeout(timeout_cleanup);
        if (!this.currentTest.state.includes('passed')) {
            maestro_commands.save_maestro_log(failure_count++, done);
        } else {
            done();
        }
    });

    /**
     * DHCP tests
     **/
    describe('DHCP', function() {
        it('should retrieve the active networking maestro config', function(done) {
            this.timeout(timeout);
            this.done = done;
            let command = Commands.list.maestro_shell_get;
            command = command.replace('{{url}}', '/net/interfaces');
            maestro_commands.run_shell(command, function(active_iface) {
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
            maestro_commands.maestro_api_set_ip_address(this, 'eth1', '10.234.234.234');
        });

        it('should change the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.maestro_api_set_ip_address(this, 'eth2', '10.229.229.229');
        });
    });

        /**
     * Servicectl tests
     **/
    describe('ServiceCtl', function() {

        it('should start the service', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.get_service_to_default_state('exampleservice.service');
            maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                this.is_enabled = is_enabled;
                this.is_running = is_running;
                this.not_found = not_found;
            }.bind(this));
            assert.equal(false, this.is_running);
            maestro_commands.maestro_api_control_service('exampleservice.service', 'start');
            maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                this.is_enabled = is_enabled;
                this.is_running = is_running;
                this.not_found = not_found;
            }.bind(this));
            assert.equal(true, this.is_running);
            this.done();
        });

        it('should enable the service', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.get_service_to_default_state('exampleservice.service');
            maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                this.is_enabled = is_enabled;
                this.is_running = is_running;
                this.not_found = not_found;
            }.bind(this));
            assert.equal(false, this.is_enabled);
            maestro_commands.maestro_api_control_service('exampleservice.service', 'enable');
            maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                this.is_enabled = is_enabled;
                this.is_running = is_running;
                this.not_found = not_found;
            }.bind(this));
            assert.equal(true, this.is_enabled);
            this.done();
        });

        it('should get the status of the service', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.check_service_state('exampleservice.service', function(not_found, is_enabled, is_running) {
                this.is_enabled = is_enabled;
                this.is_running = is_running;
                this.not_found = not_found;
            }.bind(this));
            let command = Commands.list.maestro_shell_get;
            command = command.replace('{{url}}', '/services/exampleservice.service');
            maestro_commands.run_shell(command, function(service_status_payload) {
                let service_status = JSON.parse(service_status_payload);

                assert.equal(false, this.not_found);
                assert.equal(service_status.IsEnabled, this.is_enabled);
                assert.equal(service_status.IsRunning, this.is_running);

                this.done();
            }.bind(this));
        });
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

        beforeEach(function(done) {
            this.timeout(timeout);
            let command = Commands.list.remove_logs.replace('{{filename}}', '/var/log/maestro/maestro.log*');
            maestro_commands.run_shell(command, done);
        });

        it('should retrieve the active logging maestro config', function(done) {
            this.timeout(timeout);
            this.done = done;
            let command = Commands.list.maestro_shell_get;
            command = command.replace('{{url}}', '/log/target');
            maestro_commands.run_shell(command, function(active_logging) {
                let active_config = JSON.parse(active_logging);

                let active_file = this.file_target;
                let active_filter = this.file_target_default_filter;
                let file_arr = active_config.filter(function (el) {
                    return el.File.includes(this);
                }.bind(active_file));

                assert.equal(file_arr[0].File, active_file);
                assert.equal(file_arr[0].Filters[0].Levels, active_filter);

                this.done();
            }.bind(this));
        });

        it('should add the warn filter to the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;

            maestro_commands.maestro_api_post_log_filter(this.file_target, ['warn'], function(result) {
                maestro_commands.maestro_api_verify_log_filters(this.file_target, ['warn', this.file_target_default_filter], this.done);
            }.bind(this));
        });

        it('should delete the error filter from the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;

            maestro_commands.maestro_api_delete_log_filter(this.file_target, [this.file_target_default_filter], function(result) {
                maestro_commands.maestro_api_verify_log_filters(this.file_target, ['warn'], this.done);
            }.bind(this));
        });

        it('should delete the warn filter and add the info filter to the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;

            maestro_commands.maestro_api_delete_log_filter(this.file_target, ['warn'], function(result) {
                maestro_commands.maestro_api_post_log_filter(this.file_target, ['info'], function(result) {
                    maestro_commands.maestro_api_verify_log_filters(this.file_target, ['info'], this.done);
                }.bind(this));
            }.bind(this));
        });

        it('should set the all filter for the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;

            maestro_commands.maestro_api_post_log_filter(this.file_target, ['all'], function(result) {
                maestro_commands.maestro_api_verify_log_filters(this.file_target, ['all'], this.done);
            }.bind(this));
        });
    });
});

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
                        sysLogSocket: '/run/systemd/journal/syslog',
                        network: {
                            interfaces: [
                                {if_name: 'eth1', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.77.77.77', ipv4_mask: 24},
                                {if_name: 'eth2', existing: 'replace', clear_addresses: true, dhcpv4: false, ipv4_addr: '10.66.66.66', ipv4_mask: 24}
                            ]
                        },
                        devicedb_conn_config: {
                            devicedb_uri: 'https://' + device_id + ':9090',
                            devicedb_prefix: 'vagrant',
                            devicedb_bucket: 'lww',
                            relay_id: device_id,
                            ca_chain: Commands.devicedb_src + '/hack/certs/myCA.pem'
                        },
                        targets: [
                            {file: '/var/log/maestro/maestro.log', format_time: "\"timestamp\":%ld%03d, ", filters: [{levels: 'error'}]},
                        ],
                        config_end: true
                    };
                    this.file_target = view.targets[0].file;
                    this.file_target_default_filter = view.targets[0].filters[0].levels;

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

    afterEach(function(done) {
        this.timeout(timeout_cleanup);
        if (!this.currentTest.state.includes('passed')) {
            maestro_commands.save_maestro_log(failure_count++, done);
        } else {
            done();
        }
    });

    /**
     * DHCP tests
     **/
    describe('DHCP', function() {
        it('should set the IP address of the first network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_ip_address(this, 'eth1', '10.122.122.122');
        });

        it('should change the IP address of the first network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_ip_address(this, 'eth1', '10.234.234.234');
        });

        it('should set the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_ip_address(this, 'eth1', '10.125.125.125');
        });

        it('should change the IP address of the second network adapter', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_ip_address(this, 'eth2', '10.222.222.222');
        });
    });

    /**
     * Logging tests
     **/
    describe('Logging', function() {

        beforeEach(function(done) {
            this.timeout(timeout);
            let command = Commands.list.remove_logs.replace('{{filename}}', '/var/log/maestro/maestro.log*');
            maestro_commands.run_shell(command, done);
        });

        it('should add the warn filter to the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_log_filter(this, this.file_target, ['warn'], function(result) {
                maestro_commands.maestro_api_verify_log_filters(this.file_target, ['warn', this.file_target_default_filter], this.done);
            }.bind(this));
        });
    });

        /**
     * Servicectl tests
     **/
    describe('ServiceCtl', function() {

        beforeEach(function(done) {
            this.timeout(timeout);
            let command = Commands.list.remove_logs.replace('{{filename}}', '/var/log/maestro/maestro.log*');
            maestro_commands.run_shell(command, done);
        });

        it('should add the warn filter to the file log target', function(done) {
            this.timeout(timeout);
            this.done = done;
            maestro_commands.devicedb_set_log_filter(this, this.file_target, ['warn'], function(result) {
                maestro_commands.maestro_api_verify_log_filters(this.file_target, ['warn', this.file_target_default_filter], this.done);
            }.bind(this));
        });
    });
});

