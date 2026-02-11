#!/bin/bash
# Unmount remote music directory

LOCAL_MOUNT="/Volumes/RemoteMusic"

if mount | grep -q "$LOCAL_MOUNT"; then
    echo "🔌 Unmounting $LOCAL_MOUNT..."
    umount "$LOCAL_MOUNT" || diskutil unmount "$LOCAL_MOUNT"
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully unmounted!"
    else
        echo "⚠️  Force unmounting..."
        diskutil unmount force "$LOCAL_MOUNT"
    fi
else
    echo "ℹ️  $LOCAL_MOUNT is not mounted"
fi
