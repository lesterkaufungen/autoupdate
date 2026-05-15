#!/bin/sh
PID=$1
SRC=$2
DST=$3
SERVICE_NAME=$4
shift 4

# Give it a moment to exit gracefully if it was already triggered
sleep 1

# Kill the process if it's still alive
if kill -0 $PID 2>/dev/null; then
    kill -9 $PID 2>/dev/null
fi

# Wait for it to die
while kill -0 $PID 2>/dev/null; do
    sleep 0.1
done

# Replace binary
mv -f "$SRC" "$DST"
chmod +x "$DST"

# Remove self
rm -f "$0"

# Restart
if [ -n "$SERVICE_NAME" ]; then
    OS=$(uname)
    if [ "$OS" = "Darwin" ]; then
        # macOS launchd support
        # We try to detect if it's a system or user daemon.
        # For a LaunchDaemon, 'system/' is the domain.
        # For a LaunchAgent, 'gui/' or 'user/' domain might be needed.
        # We try to use kickstart -k which is the modern way to restart.
        if launchctl list "$SERVICE_NAME" >/dev/null 2>&1; then
             # Try system domain (common for LaunchDaemons)
             launchctl kickstart -k "system/$SERVICE_NAME" 2>/dev/null || \
             # Try gui domain (common for LaunchAgents)
             launchctl kickstart -k "gui/$(id -u)/$SERVICE_NAME" 2>/dev/null || \
             # Legacy fallback
             (launchctl stop "$SERVICE_NAME" && launchctl start "$SERVICE_NAME")
        fi
    else
        # Linux systemd
        systemctl restart "$SERVICE_NAME"
    fi
else
    # Running as standalone (binary or .app bundle binary)
    # Check if we are inside a .app bundle and might want to use 'open'
    # but exec $DST is more reliable for inheriting environment/args.
    exec "$DST" "$@"
fi
