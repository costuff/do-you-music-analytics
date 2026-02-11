#!/bin/bash
# Mount remote music directory via SSHFS

REMOTE_USER="skoutelis"
REMOTE_HOST="100.110.101.64"
REMOTE_PATH="/mnt/mydrive/ExtHDD-8TB/Music"
LOCAL_MOUNT="/Volumes/RemoteMusic"

# Check if already mounted
if mount | grep -q "$LOCAL_MOUNT"; then
    echo "✅ Remote music directory is already mounted at $LOCAL_MOUNT"
    exit 0
fi

# Check if mount point exists
if [ ! -d "$LOCAL_MOUNT" ]; then
    echo "Creating mount point..."
    sudo mkdir -p "$LOCAL_MOUNT"
    sudo chown $USER "$LOCAL_MOUNT"
fi

echo "🔌 Mounting remote music directory..."
echo "   From: $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH"
echo "   To:   $LOCAL_MOUNT"
echo ""

# Mount with sshfs
sshfs "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH" "$LOCAL_MOUNT" \
    -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3

if [ $? -eq 0 ]; then
    echo "✅ Successfully mounted!"
    echo ""
    echo "📊 Mount info:"
    df -h "$LOCAL_MOUNT"
    echo ""
    echo "📁 Contents:"
    ls -lh "$LOCAL_MOUNT" | head -10
else
    echo "❌ Failed to mount. Please check:"
    echo "   1. macFUSE is approved in System Settings → Privacy & Security"
    echo "   2. SSH access to $REMOTE_HOST is working"
    echo "   3. Remote path exists: $REMOTE_PATH"
    exit 1
fi
