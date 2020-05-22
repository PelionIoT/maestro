#!/bin/bash -e

function cleanup {
	echo "Cleaning up maestro"
	sudo pkill maestro
	set +o xtrace
}

trap cleanup EXIT

. /etc/profile.d/envvars.sh

# set -o xtrace

sudo ip addr flush dev eth1
sudo ip addr flush dev eth2

SITE_ID=$(cat $DEVICEDB_SRC/hack/certs/site_id)
DEVICE_ID=$(cat $DEVICEDB_SRC/hack/certs/device_id)

cp ./feature_devicedb.yaml $MAESTRO_SRC/maestro.config

echo "Starting maestro"
sudo maestro &>/tmp/maestro.log &

echo "Waiting 5 seconds for maestro to boot up..."
sleep 5

inject_syslog() {
	echo "Injecting syslog messages"
	echo 'test - Error' | systemd-cat -p err
	echo 'test - Info' | systemd-cat -p info
	echo 'test - Warn' | systemd-cat -p warning
	echo 'test - Debug' | systemd-cat -p debug
	sleep 2
}

devicedb_send() {
	devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$DEVICE_ID.$1 -value '{"name":"vagrant.'$DEVICE_ID'.'$1'","relay":"'$DEVICE_ID'","body":"'"$3"'"}'
	sleep 2
	devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$DEVICE_ID.$2 -value '{"name":"vagrant.'$DEVICE_ID'.'$2'","body":"{\"config_commit\":true}"}'
	sleep 2
}

## Test case #1
##   Verify error is on by default

echo | sudo tee /var/log/maestro/maestro.log

inject_syslog
if [ ! "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Error')" ]; then
	echo "Error not found"
	exit 1
fi
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Warn')" ]; then
	echo "Warn found and it shouldn't have been"
	exit 1
fi
echo "COMPLETE - test case #1"

## Test case #2
##   Submit warning and error
echo | sudo tee /var/log/maestro/maestro.log

DEVIECDB_BODY="{\\\"targets\\\":[{\\\"file\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"name\\\":\\\"\\\",\\\"existing\\\":\\\"replace\\\",\\\"filters\\\":[{\\\"target\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"levels\\\":\\\"warn\\\"},{\\\"target\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"levels\\\":\\\"error\\\"}]}]}"
devicedb_send "MAESTRO_LOG_CONFIG_ID" "MAESTRO_LOG_CONFIG_COMMIT_FLAG" "$DEVIECDB_BODY"

inject_syslog
if [ ! "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Error')" ]; then
	echo "Error not found"
	exit 1
fi
if [ ! "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Warn')" ]; then
	echo "Warn not found"
	exit 1
fi
echo "COMPLETE - test case #2"

## Test case #3
##   Submit warning (but not error)
echo | sudo tee /var/log/maestro/maestro.log

DEVIECDB_BODY="{\\\"targets\\\":[{\\\"file\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"name\\\":\\\"\\\",\\\"existing\\\":\\\"replace\\\",\\\"filters\\\":[{\\\"target\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"levels\\\":\\\"warn\\\"}]}]}"
devicedb_send "MAESTRO_LOG_CONFIG_ID" "MAESTRO_LOG_CONFIG_COMMIT_FLAG" "$DEVIECDB_BODY"

inject_syslog
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Error')" ]; then
	echo "Error found and it shouldn't have been"
	exit 1
fi
if [ ! "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Warn')" ]; then
	echo "Warn not found"
	exit 1
fi
echo "COMPLETE - test case #3"

## Test case #4
##   Submit info (but not error or warn)
echo | sudo tee /var/log/maestro/maestro.log

DEVIECDB_BODY="{\\\"targets\\\":[{\\\"file\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"name\\\":\\\"\\\",\\\"existing\\\":\\\"replace\\\",\\\"filters\\\":[{\\\"target\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"levels\\\":\\\"info\\\"}]}]}"
devicedb_send "MAESTRO_LOG_CONFIG_ID" "MAESTRO_LOG_CONFIG_COMMIT_FLAG" "$DEVIECDB_BODY"

inject_syslog
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Error')" ]; then
	echo "Error found and it shouldn't have been"
	exit 1
fi
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Warn')" ]; then
	echo "Warn found and it shouldn't have been"
	exit 1
fi
if [ ! "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Info')" ]; then
	echo "Info not found"
	exit 1
fi
echo "COMPLETE - test case #4"

## Test case #5
##   Clear all levels
echo | sudo tee /var/log/maestro/maestro.log

DEVIECDB_BODY="{\\\"targets\\\":[{\\\"file\\\":\\\"/var/log/maestro/maestro.log\\\",\\\"name\\\":\\\"\\\",\\\"existing\\\":\\\"replace\\\",\\\"filters\\\":[{\\\"target\\\":\\\"\\\",\\\"levels\\\":\\\"\\\"}]}]}"
devicedb_send "MAESTRO_LOG_CONFIG_ID" "MAESTRO_LOG_CONFIG_COMMIT_FLAG" "$DEVIECDB_BODY"

inject_syslog
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Error')" ]; then
	echo "Error found and it shouldn't have been"
	exit 1
fi
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Warn')" ]; then
	echo "Warn found and it shouldn't have been"
	exit 1
fi
if [ "$(sudo cat /var/log/maestro/maestro.log | grep 'test - Info')" ]; then
	echo "Info found and it shouldn't have been"
	exit 1
fi
echo "COMPLETE - test case #5"
