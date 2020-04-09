# Maestro Config

The following guide documents the options available in the `maestro.config` yaml file that is provided to maestro on startup. This file is hardcoded within maestro, and should live in the maestro project directory.

The following is a bare minimum configuration for maestro:

```
config_end: true
```

The remaining sections detail subsections that can be placed in the file above the `config_end: true` line.

## Networking

You must specify a network interface:

```
network:
    interfaces:
        - <interface 1> # Outlined below
        - <interface 2> # Outlined below
        - <interface ...> # Outlined below
```

### Static interface:

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

Breakdown:
* `if_name` - REQUIRED. Name of the interface Maestro will modify
* `existing` - OPTIONAL. Tells Maestro to `replace` or `override` the existing saved interface
* `clear_addresses` - REQUIRED. Clears any existing addresses assigned to the interface before setting up the specified addresses
* `dhcpv4` - REQUIRED. `false` for static interfaces
* `ipv4_addr` - REQUIRED. IP address to assign to the interface
* `ipv4_mask` - REQUIRED. IP mask to use for the subnet
* `hw_addr` - MAC address to use for the interface
* `default_gateway` - REQUIRED. IP address of the default gateway to be used

### DHCP interface:

```
- if_name: eth1
  existing: override
  clear_addresses: true
  dhcpv4: true
  hw_addr: "{{ARCH_ETHERNET_MAC}}"
```

Breakdown:
* `if_name` - REQUIRED. Name of the interface Maestro will modify
* `existing` - OPTIONAL. Tells Maestro to `replace` or `override` the existing saved interface
* `clear_addresses` - REQUIRED. Clears any existing addresses assigned to the interface before setting up the specified addresses
* `dhcpv4` - REQUIRED. `true` for dhcp interfaces
* `hw_addr` - MAC address to use for the interface

## DeviceDB

Only necessary if you want Maestro to connect to a deviceDB server:

```
devicedb_conn_config:
    devicedb_uri: "https://localhost:9090"
    devicedb_prefix: "vagrant"
    devicedb_bucket: "lww"
    relay_id: "2462a03b129928209ce396f"
    ca_chain: "~/.ssl_certs/myCA.pem"
```

Breakdown:
* `devicedb_uri` - REQUIRED. URL of the deviceDB edge instance. NOT the deviceDB cloud URL
* `devicedb_prefix` - REQUIRED. Table within devicedb to put data into
* `devicedb_bucket` - REQUIRED. Bucket within the table specified above.
* `relay_id` - REQUIRED. Unique identifier for the gateway. Usually the gateway ID specified by the cloud
* `ca_chain` - REQUIRED. Location of the root CA certificate that the deviceDB cloud instance is setup with.

## Logging

### SysLog

To enable syslog, add to your `maestro.config`:

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

Where the contents of the `echo` is the message, and `err` is the log level. Note that you must have the log level you specify match what is in your log target filters list otherwise it will not show up.

### Targets

A target is a destination where Maestro will output its logs to. It can be a file, the cloud, etc. For example, Maestro has the ability to take syslog's and dump them to a file. The file that the syslogs get dumped to is considered a `target`. Maestro can be setup to have multiple targets.

```
targets:
    - <target 1> # Outlined below
    - <target 2> # Outlined below
    - <target ...> # Outlined below
```

Breakdown:
* `name` - Unique identifier for the log target. `toCloud` is a special name for sending data to the cloud and is the required `name` for cloud targets. File targets do not need a `name`
* `file` - Required for log targets destined to a file. This is the output location for said file.
* `format_time` - REQUIRED. Used to specify the time format in the output logs
* `delim` - OPTIONAL. Used to specify the delimiter between logs
* `filters` - REQUIRED. Used to specify what level of logs make it to the output log
    * `warn`
    * `info` or `success`. They behave the same
    * `error`
    * `debug`
    * `all`
* `rotate` - Optional; only used for log targets destined to a file. Determines how to rotate between X amount of files
    * `max_files` - Max files to rotate between
    * `max_file_size` - Max size of each log file. In bytes
    * `max_total_size` - Max size of all of the log files. In bytes
    * `rotate_on_start` - Move to the next file when Maestro reboots

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

To view a file log, run the following:

```
sudo tail -f /var/log/maestro/maestro.log
```

Where `/var/log/maestro/maestro.log` is the file specified in the `file` field

#### Cloud target:
```
- name: "toCloud"
  format_time: "\"timestamp\":%ld%03d, "
  flag_json_escape_strings: true
  filters:
  - levels: all
```

Note: If you have a cloud target, you MUST have a section in your `maestro.config` for Symphony, which is explained below

### Symphony

If you have a log target with the name `toCloud`, this section is required in your `maestro.config`!

```
symphony:
    sys_stats_count_threshold: 15
    sys_stats_time_threshold: 120000
    client_cert: {{SYMPHONY_CLIENT_CRT}}
    client_key: {{SYMPHONY_CLIENT_KEY}}
    host: "{{SYMPHONY_HOST}}"
```

Breakdown:
* `sys_stats_count_threshold` - How many messages are stored before the stats are sent up to the cloud
* `sys_stats_time_threshold` - How long to wait before sending whatever is in the queue, regardless of message count. In milliseconds
* `client_cert` - Certificate to authenticate with the cloud. This needs to be the actual certificate in PEM format, not the file location
* `client_key` - Private key to authenticate with the cloud. This needs to be the actual key in PEM format, not the file location
* `host` - URL of the symphony cloud

### System Stats

```
sys_stats:
  vm_stats:
    every: "15s"
    name: vm
  disk_stats:
    every: "30s"
    name: disk
```

Breakdown:
* `vm_stats` - Memory statistics
* `disk_stats` - Disk statistics

How to Disable System Stats:
* Remove the System Stats section from the config file
* Add "disable_sys_stats: true" to the Symphony section in the config file

## config_end

You must put `config_end: true` at the end of your `maestro.config`

# Example

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
    - name: "toCloud"  # this is a special target for sending to the cloud. It must send as a JSON
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