# DeviceDB

The following document will describe how to configure maestro from DeviceDB. This guide assumes you are running commands within the same environment where a devicedb edge process is running which is connected to a devicedb cloud instance.

Additionally, we assume maestro is configured correctly to talk to devicedb. For information regarding configuring maestro to talk to devicedb, see `maestro_config.md`.

## Sending DeviceDB commands

The following details how to use devicedb to add/change log filters in maestro and how to verify what was sent works.

1. Set the environment variables for our devicedb commands:
    ```
    SITE_ID=$(cat $DEVICEDB_SRC/hack/certs/site_id)
    RELAY_ID=$(cat $DEVICEDB_SRC/hack/certs/device_id)
    ```
1. The following command will insert a payload into devicedb. Note that this command is a template and specific examples will be shown below.
    ```
    devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$RELAY_ID.<key> -value '{"name":"vagrant.'$RELAY_ID'.<key>","relay":"'$RELAY_ID'","body":"<body>"}'
    ```
    The following items need to be replaced in the template:
    * `<key>` - The key where devicedb will insert the body text. Example: `MAESTRO_LOG_CONFIG_ID `
    * `<body>` - The JSON payload to store. Note that this is JSON string escaped. Example body (these options are shown in the `maestro_config.md` document):
        ```
        {\"targets\":[{\"file\":\"/var/log/maestro/maestro.log\",\"filters\":[{\"target\":\"/var/log/maestro/maestro.log\",\"levels\":\"warn\"},{\"target\":\"/var/log/maestro/maestro.log\",\"levels\":\"error\"}],\"name\":\"\",\"existing\":\"replace\"}]}
        ```

    Full example shown here:
    ```
    devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$RELAY_ID.MAESTRO_LOG_CONFIG_ID -value '{"name":"vagrant.'$RELAY_ID'.MAESTRO_LOG_CONFIG_ID","relay":"'$RELAY_ID'","body":"{\"targets\":[{\"file\":\"/var/log/maestro/maestro.log\",\"filters\":[{\"target\":\"/var/log/maestro/maestro.log\",\"levels\":\"warn\"},{\"target\":\"/var/log/maestro/maestro.log\",\"levels\":\"error\"}],\"name\":\"\",\"existing\":\"replace\"}]}"}'
    ```

1. To commit the changes and to make the log filters active, run:
    ```
    devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$RELAY_ID.<commit-key> -value '{"name":"vagrant.'$RELAY_ID'.<commit-key>","body":"{\"config_commit\":true}"}'
    ```
    The following items need to be replaced in the template:
    * `<commit-key>` - The commit key where devicedb will commit the payload. Example: `MAESTRO_LOG_CONFIG_COMMIT_FLAG `

    Full example shown here:
    ```
    devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$RELAY_ID.MAESTRO_LOG_CONFIG_COMMIT_FLAG -value '{"name":"vagrant.'$RELAY_ID'.MAESTRO_LOG_CONFIG_COMMIT_FLAG","body":"{\"config_commit\":true}"}'
    ```

## Verifying the changes

To verify that your changes above are active, choose a verification type listed below and run the commands.

### Maestro-Shell

1. Run maestro-shell with the following command.
    ```
    sudo LD_LIBRARY_PATH=/home/vagrant/work/gostuff/src/github.com/armPelionEdge/maestro/greasego/deps/lib /home/vagrant/work/gostuff/bin/maestro-shell
    ```
1. Inside of maestro-shell run the command
    ```
    > log get
    ```
1. It should show entries like below which verifies that `warn` and `error` targets have been set (which was the configuration from the example).
    ```
    [1]: {
        format_post: ""
        rotate: {
            max_files: 0.000000
            rotate_on_start: false
            max_file_size: 0.000000
            max_total_size: 0.000000
        }
        format_pre: ""
        format_origin: ""
        flag_json_escape_strings: false
        file: "/var/log/maestro/maestro.log"
        format_level: ""
        format_time: ""
        format_pre_msg: ""
        tty: ""
        format: {
            level: ""
            tag: ""
            origin: ""
            time: ""
        }
        delim: ""
        format_tag: ""
        name: "/var/log/maestro/maestro.log"
        existing: ""
        filters: [][0]: {
                pre: ""
                post: ""
                post_fmt_pre_msg: ""
                target: "/var/log/maestro/maestro.log"
                levels: "warn"
                tag: ""
            }

    [1]: {
                target: "/var/log/maestro/maestro.log"
                levels: "error"
                tag: ""
                pre: ""
                post: ""
                post_fmt_pre_msg: ""
            }

        example_file_opts: {
            Mode: 0.000000
            Flags: 0.000000
            Max_files: 0.000000
            Max_file_size: 0.000000
            Max_total_size: 0.000000
            Rotate_on_start: false
        }
    }
    ```

### Syslog messaging

1. You can also run the following command to send syslog messages.
    ```
    echo 'test - Error' | systemd-cat -p err; echo 'test - Info' | systemd-cat -p info; echo 'test - Warn' | systemd-cat -p warning; echo 'test - Debug' | systemd-cat -p debug
    ```
1. Then you can verify that it worked by dumping the log to see if the message is in there.
    ```
    sudo cat /var/log/maestro/maestro.log
    ```
1. It should dump out something like this:
    ```
    [1590774692:558] {--} (--) cat[6255]: test - Error
    [1590774692:562] {--} (--) cat[6259]: test - Warn
    ```

## Script automation

We have also provided a testing script that can be used to simplify the process above to a single command line.

Open two shells to your target device.

1. In one of the shells start maestro
    ```
    sudo maestro
    ```
1. In another shell, run the following commands:
    ```
    cd $MAESTRO_SRC/tests/scripts
    ./test_devicedb.sh error warn
    ```
    This will send `error` and `warn` filters down to maestro via devicedb

1. The following is an example of a success:
    ```
    Sending devicedb PUT and COMMIT and waiting 5 seconds...
    Injecting syslog messages
    Checking for warn
    Checking for info
    Checking for error
    Checking for debug
    PASS
    ```
