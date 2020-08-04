# Maestro config

This document describes the Maestro configuration options, which you define in the `maestro.config` YAML file that is provided to Maestro on startup. This file is hardcoded in Maestro, and should live in the Maestro project directory.

The following is a bare minimum configuration for maestro:

```
config_end: true
```

You can include additional subsections in the file, above the `config_end: true` line, as shown in the examples.

## Networking

To specify a network interface, add:

```
network:
    interfaces:
        - <interface 1> # Outlined below
        - <interface 2> # Outlined below
        - <interface ...> # Outlined below
```

### Static interface

```
- if_name: eth1
  existing: override
  clear_addresses: true
  dhcpv4: false
  ipv4_addr: 10.0.103.103
  ipv4_mask: 24
  hw_addr: "{{ARCH_ETHERNET_MAC}}"
  default_gateway: 10.0.103.1
```

Parameters:
* `if_name` - REQUIRED. Name of the interface Maestro modifies.
* `existing` - OPTIONAL. Tells Maestro to `replace` or `override` the existing saved interface.
* `clear_addresses` - REQUIRED. Clears any existing addresses assigned to the interface before setting up the specified addresses.
* `dhcpv4` - REQUIRED. `false` for static interfaces.
* `ipv4_addr` - REQUIRED. IP address to assign to the interface.
* `ipv4_mask` - REQUIRED. IP mask to use for the subnet.
* `hw_addr` - MAC address to use for the interface.
* `default_gateway` - REQUIRED. IP address of the default gateway.

### DHCP interface

```
- if_name: eth1
  existing: override
  clear_addresses: true
  dhcpv4: true
  hw_addr: "{{ARCH_ETHERNET_MAC}}"
```

Breakdown:
* `if_name` - REQUIRED. Name of the interface Maestro modifies.
* `existing` - OPTIONAL. Tells Maestro to `replace` or `override` the existing saved interface.
* `clear_addresses` - REQUIRED. Clears any existing addresses assigned to the interface before setting up the specified addresses.
* `dhcpv4` - REQUIRED. `true` for DHCP interfaces.
* `hw_addr` - REQUIRED. MAC address to use for the interface.

### LTE interface

```
- if_name: lte1
  existing: override
  dhcpv4: true
  type: lte
  serial: ttyLTE
  apn: my_apn
```

Breakdown:
* `if_name` - REQUIRED. Name of the interface Maestro modifies.
* `existing` - OPTIONAL. Tells Maestro to `replace` or `override` the existing saved interface.
* `dhcpv4` - REQUIRED. `true` for DHCP interfaces.
* `type` - REQUIRED. Tells Maestro the type of connection, `lte` for LTE interfaces.
* `serial` - REQUIRED. Modem serial device.  It is possible to use "*" to refer to any available modem.
* `apn` - REQUIRED. Access point name.

## DeviceDB

To connect to a deviceDB server, add:

```
devicedb_conn_config:
    devicedb_uri: "https://localhost:9090"
    devicedb_prefix: "vagrant"
    devicedb_bucket: "lww"
    relay_id: "2462a03b129928209ce396f"
    ca_chain: "~/.ssl_certs/myCA.pem"
```

Parameters:
* `devicedb_uri` - REQUIRED. URL of the deviceDB edge instance. **Not the deviceDB cloud URL.**
* `devicedb_prefix` - REQUIRED. Table within deviceDB to put data into.
* `devicedb_bucket` - REQUIRED. Bucket within the table specified above.
* `relay_id` - REQUIRED. Unique identifier for the gateway. Usually, the gateway ID specified by the cloud.
* `ca_chain` - REQUIRED. Location of the root CA certificate the deviceDB cloud instance is set up with.

## Logging

### SysLog

To enable syslog, add:

```
sysLogSocket: /run/systemd/journal/syslog
```

Where `/run/systemd/journal/syslog` is the path to your syslog socket.

To inject a log into a log target, you can use syslog:

```
echo "test err message" | systemd-cat -p err
```

Available syslog levels are:

* `err`
* `warning`
* `info`
* `debug`

Where the content of the `echo` is the message, and `err` is the log level. The log level you specify must be one of the values in your log target `filters` list; otherwise, the log will not show up.

### Targets

A target is a destination to which Maestro outputs its logs. The target can be a file, the cloud, and so on. Maestro can have multiple targets.

To define the target into which Maestro dumps its logs, add:

```
targets:
    - <target 1> # Outlined below
    - <target 2> # Outlined below
    - <target ...> # Outlined below
```


#### File target:

```
- file: "/var/log/maestro/maestro.log"
  rotate:
    max_files: 4
    max_file_size: 10000000
    max_total_size: 42000000
    rotate_on_start: true
  delim: "\n"
  format_time: "[%ld:%d] "
  filters:
  - levels: warn
  - levels: error
```

Parameters:
* `file` - The location of the output file.
* `rotate` - OPTIONAL. Defines the log file rotation.
    * `max_files` - Maximum number of log files to rotate between.
    * `max_file_size` - Maximum size of each log file, in bytes.
    * `max_total_size` - Maximum total size of all log files, in bytes.
    * `rotate_on_start` - Move to the next file when Maestro reboots.
* `delim` - OPTIONAL. Specifies the delimiter between logs.
* `format_time` - REQUIRED. Specifies the time format in the output logs.
* `filters` - REQUIRED. Specifies what level of logs make it to the output log.
    * `warn`
    * `info` or `success` - Behave the same.
    * `error`
    * `debug`
    * `all`


**To view a file log, run:**

```
sudo tail -f /var/log/maestro/maestro.log
```

Where `/var/log/maestro/maestro.log` is the file specified in the `file` field.

#### Cloud target:
```
- name: "toCloud"
  format_time: "\"timestamp\":%ld%03d, "
  flag_json_escape_strings: true
  filters:
  - levels: all
```
<span class="notes">**Note:** If you have a cloud target, you MUST have a section in your `maestro.config` for [Symphony](#symphony).</span>

Parameters:
* `name` - Unique identifier of the log target. `toCloud` is a special name for sending data to the cloud and is the required `name` for cloud targets.
* `format_time` - REQUIRED. Specifies the time format in the output logs.
* `flag_json_escape_strings` - Sends log dumps in JSON format. Always `true` for cloud targets.
* `filters` - REQUIRED. Specifies what level of logs make it to the output log.
    * `warn`
    * `info` or `success` - Behave the same.
    * `error`
    * `debug`
    * `all`

### Symphony

If you have a cloud target, you must add:

```
symphony:
    sys_stats_count_threshold: 15
    sys_stats_time_threshold: 120000
    client_cert: {{SYMPHONY_CLIENT_CRT}}
    client_key: {{SYMPHONY_CLIENT_KEY}}
    host: "{{SYMPHONY_HOST}}"
```

Parameters:
* `sys_stats_count_threshold` - How many messages are stored before the stats are sent to the cloud.
* `sys_stats_time_threshold` - How long to wait, in milliseconds, before sending messages in the queue to the cloud, regardless of message count.
* `client_cert` - Certificate to authenticate with the cloud. The actual certificate in PEM format, not the file location.
* `client_key` - Private key to authenticate with the cloud. The actual key in PEM format, not the file location.
* `host` - URL of the Symphony cloud.

### System stats

To log system statistics, add:
```
sys_stats:
  vm_stats:
    every: "15s"
    name: vm
  disk_stats:
    every: "30s"
    name: disk
```

Parameters:
* `vm_stats` - Memory statistics.
* `disk_stats` - Disk statistics.

**To disable system stats:**

* Remove the `sys_stats` section from the config file.
* Add `disable_sys_stats: true` to the `symphony` section in the config file.

## config_end

You must put `config_end: true` at the end of your `maestro.config`.

## maestro.config file example

```
sysLogSocket: /run/systemd/journal/syslog
network:
    interfaces:
        - if_name: eth1
          existing: override
          clear_addresses: true
          dhcpv4: false
          ipv4_addr: 10.0.103.103
          ipv4_mask: 24
          hw_addr: "{{ARCH_ETHERNET_MAC}}"
          default_gateway: 10.0.103.1
        - if_name: eth2
          existing: override
          clear_addresses: true
          dhcpv4: false
          ipv4_addr: 10.0.102.102
          ipv4_mask: 24
          hw_addr: "{{ARCH_ETHERNET_MAC}}"
          default_gateway: 10.0.102.1
devicedb_conn_config:
    devicedb_uri: "https://{{DEVICE_ID}}:9090"
    devicedb_prefix: "vagrant"
    devicedb_bucket: "lww"
    relay_id: "{{DEVICE_ID}}"
    ca_chain: "{{DEVICEDB_SRC}}/hack/certs/myCA.pem"
sys_stats: # system stats intervals
  vm_stats:
    every: "15s"
    name: vm
  disk_stats:
    every: "30s"
    name: disk
symphony:
    disable_sys_stats: false
    sys_stats_count_threshold: 15     # send if you have 15 or more stats queued
    sys_stats_time_threshold: 120000  # every 120 seconds send stuff, no matter what
    client_cert: {{SYMPHONY_CLIENT_CRT}}
    client_key: {{SYMPHONY_CLIENT_KEY}}
    host: "{{SYMPHONY_HOST}}"
targets:
    - file: "/var/log/maestro/maestro.log"
      rotate:
        max_files: 4
        max_file_size: 10000000  # 10MB max file size
        max_total_size: 42000000
        rotate_on_start: true
      delim: "\n"
      format_time: "[%ld:%d] "
      format_level: "<%s> "
      format_tag: "{%s} "
      format_origin: "(%s) "
      filters:
      - levels: warn
        format_pre: "\u001B[33m"    # yellow
        format_post: "\u001B[39m"
      - levels: error
        format_pre: "\u001B[31m"    # red
        format_post: "\u001B[39m"
    - name: "toCloud"  # this sends log dumps to the cloud as a JSON.
      format_time: "\"timestamp\":%ld%03d, "
      format_level: "\"level\":\"%s\", "
      format_tag: "\"tag\":\"%s\", "
      format_origin: "\"origin\":\"%s\", "
      format_pre_msg: "\"text\":\""
      format_post: "\"},"
      flag_json_escape_strings: true
      filters:
      - levels: all
        format_pre: "{"     # you will wrap this output with { "log": [ OUTPUT ] }
config_end: true
```
