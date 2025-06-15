#!/bin/bash
set -e
umask 022

if [ -z "$1" ]; then
    # Create the persistent data folder if it doesn't exist
    install -d -m 0755 /data || exit 1

    # Run as a server
    /usr/local/bin/whisper server --listen :80
else
    exec /usr/local/bin/whisper "$@"
fi
