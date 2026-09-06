#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}
STORAGE_DIR=${STORAGE_DIR:-/storage}

# Ensure storage directory exists with proper ownership and 0775 permissions
mkdir -p "$STORAGE_DIR"
chown -R "$PUID:$PGID" "$STORAGE_DIR"
chmod 0775 "$STORAGE_DIR"

# Drop privileges and execute CMD as PUID:PGID
exec su-exec "$PUID:$PGID" "$@"
