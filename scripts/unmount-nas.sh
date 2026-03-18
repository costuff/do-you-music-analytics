#!/usr/bin/env bash
# Unmount the remote NAS from your dev machine.
set -e
LOCAL_MOUNT="/Volumes/RemoteMusic"
if mount | grep -q "$LOCAL_MOUNT"; then
    echo "Unmounting $LOCAL_MOUNT..."
    if command -v umount >/dev/null 2>&1; then
        umount "$LOCAL_MOUNT" 2>/dev/null || diskutil unmount "$LOCAL_MOUNT"
    else
        diskutil unmount "$LOCAL_MOUNT"
    fi
    echo "✅ Unmounted."
else
    echo "Not mounted."
fi
