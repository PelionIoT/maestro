#!/bin/bash -e

. /etc/profile.d/envvars.sh

inputLevels=$@

echoerr() {
	echo "$@" 1>&2;
}

inject_syslog() {
	echo "Injecting syslog messages"
	echo 'syslog inject - error' | systemd-cat -p err
	echo 'syslog inject - info' | systemd-cat -p info
	echo 'syslog inject - warn' | systemd-cat -p warning
	echo 'syslog inject - debug' | systemd-cat -p debug
	sleep 2
}

devicedb_send() {

	SITE_ID=$(cat $DEVICEDB_SRC/hack/certs/site_id)
	DEVICE_ID=$(cat $DEVICEDB_SRC/hack/certs/device_id)

	echo "Sending devicedb PUT and COMMIT and waiting 5 seconds..."
	devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$DEVICE_ID.$1 -value '{"name":"vagrant.'$DEVICE_ID'.'$1'","relay":"'$DEVICE_ID'","body":"'"$3"'"}'
	sleep 2
	devicedb cluster put -site $SITE_ID -bucket lww -key vagrant.$DEVICE_ID.$2 -value '{"name":"vagrant.'$DEVICE_ID'.'$2'","body":"{\"config_commit\":true}"}'
	sleep 2
}

check_syslog() {
	allLevels=("warn" "info" "error" "debug")

	for level in ${allLevels[@]}; do
		echo "Checking for ${level}"
		result=$(sudo grep -c "syslog inject - $level" /var/log/maestro/maestro.log || true)

		# Level should be in file output
		if [[ " ${inputLevels[@]} " =~ " ${level} " ]]; then
			if [ "$result" -eq 0 ]; then
				echoerr "${level} not found in the file target and it should have been"
				exit 1
			fi
		fi

		# Level should not be in file output
		if [[ ! " ${inputLevels[@]} " =~ " ${level} " ]]; then
			if [ "$result" -ne 0 ]; then
				echoerr "${level} found in the file target and it shouldn't have been"
				exit 1
			fi
		fi
	done
}

create_log_body() {
	TMPL_BODY='{"targets":[{"file":"/var/log/maestro/maestro.log","name":"","existing":"replace","filters": []}]}'
	TMPL_LEVEL='{"target":"/var/log/maestro/maestro.log","levels":"{{level}}"}'

	BODY="$TMPL_BODY"
	for arg; do
		LEVEL=$(echo $TMPL_LEVEL | sed "s/{{level}}/$arg/g")
		BODY=$(jq -n --argjson data "$BODY" '$data' | jq '.targets[0].filters[.targets[0].filters| length] |= . + '"$LEVEL")
	done
	BODY=$(echo $BODY | sed "s/[[:blank:]]//g")
	DEVIECDB_BODY=$(echo $BODY | sed 's/"/\\"/g')
}

# Clear log to verify only this set of tests is checked
echo | sudo tee /var/log/maestro/maestro.log

# Create devicedb command to send
create_log_body $inputLevels

# Send devicedb command
devicedb_send "MAESTRO_LOG_CONFIG_ID" "MAESTRO_LOG_CONFIG_COMMIT_FLAG" "$DEVIECDB_BODY"

# Inject syslog strings
inject_syslog

# Check syslog to verify command was received by maestro
check_syslog

echoerr "PASS"
