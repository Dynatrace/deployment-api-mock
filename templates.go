package main

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	daemonScript = `#!/bin/bash
while sleep 5m; do echo $SECONDS seconds running; done
`

	sysVServiceScript = `#!/bin/bash

set -e

SCRIPT=/opt/dynatrace-mock/oneagentwatchdog
RUNAS=root

PIDFILE=/var/run/oneagentwatchdog.pid
LOGFILE=/var/log/oneagentwatchdog.log

start() {
  if [ -f /var/run/$PIDNAME ] && kill -0 $(cat /var/run/$PIDNAME); then
    echo 'Service already running' >&2
    return 1
  fi
  echo 'Starting service...' >&2
  local CMD="$SCRIPT &> \"$LOGFILE\" & echo \$!"
  su -c "$CMD" $RUNAS > "$PIDFILE"
  echo 'Service started' >&2
}

stop() {
  if [ ! -f "$PIDFILE" ] || ! kill -0 $(cat "$PIDFILE"); then
    echo 'Service not running' >&2
    return 1
  fi
  echo 'Stopping service...' >&2
  kill -15 $(cat "$PIDFILE") && rm -f "$PIDFILE"
  echo 'Service stopped' >&2
}

case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  retart)
    stop
    start
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
esac
`

	systemDServiceUnit = `[Unit]
Description=Mock for the Dynatrace Agent
After=syslog.target network.target

[Service]
ExecStart=/opt/dynatrace-mock/oneagentwatchdog

[Install]
WantedBy=multi-user.target
`

	unixUnsuccessfulInstaller = `#!/bin/bash
exit %s
`
)

var unixSuccessfulInstaller = `#!/bin/bash

set -e

DAEMON_PATH=/opt/dynatrace-mock/oneagentwatchdog
SYSV_SERVICE_PATH=/etc/init.d/oneagent
SYSTEMD_SERVICE_NAME=oneagent
SYSTEMD_SERVICE_PATH=/etc/systemd/system/oneagent.service

# Daemon
mkdir /opt/dynatrace-mock
echo "##daemonScript##" | base64 -d > $DAEMON_PATH
chmod +x $DAEMON_PATH

# Service
if [ -f "/bin/systemctl" ]; then
  echo "##systemDServiceUnit##" | base64 -d > $SYSTEMD_SERVICE_PATH

  systemctl start oneagent
else
  echo "##sysVServiceScript##" | base64 -d > $SYSV_SERVICE_PATH
  chmod +x $SYSV_SERVICE_PATH

  $SYSV_SERVICE_PATH start
fi

exit 0
`

// makeUnixInstaller returns a Shell script that mocks the installer of the OneAgent.
//
// If exitCode is non-0, then the script will just exit with the value.
//
// If exitCode is 0, then the returned script also register a SysV/SystemD service on the environment.
func makeUnixInstaller(exitCode string) string {
	if exitCode != "0" {
		return fmt.Sprintf(unixUnsuccessfulInstaller, exitCode)
	}

	var out = unixSuccessfulInstaller

	// Replace variables inside the template.
	for k, v := range map[string]string{
		"##daemonScript##":       daemonScript,
		"##sysVServiceScript##":  sysVServiceScript,
		"##systemDServiceUnit##": systemDServiceUnit,
	} {
		v64 := base64.StdEncoding.EncodeToString([]byte(v))
		out = strings.Replace(out, k, v64, -1)
	}

	return out
}
