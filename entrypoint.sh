#!/bin/sh
set -e

# Auto-fix permissions for data directory when running as root
if [ "$(id -u)" = "0" ]; then
    echo "Running as root, fixing /app/data permissions..."
    mkdir -p /app/data
    chown -R appuser:appuser /app/data
    echo "Switching to appuser..."
    exec su-exec appuser "$@"
fi

# Already running as appuser
exec "$@"
