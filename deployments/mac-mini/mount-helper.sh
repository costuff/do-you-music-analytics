#!/bin/bash
# Simple SSH-based mount using macOS built-in tools
# This script should be run before starting docker-compose

set -e

REMOTE_USER="skoutelis"
REMOTE_HOST="100.110.101.64"
REMOTE_PATH="/mnt/mydrive/ExtHDD-8TB/Music"
LOCAL_MOUNT="/Volumes/RemoteMusic"

echo "🚀 DoYou Music Analytics - Mount Helper"
echo "======================================="
echo ""

# Check if already mounted
if mount | grep -q "$LOCAL_MOUNT" 2>/dev/null; then
    echo "✅ $LOCAL_MOUNT is already mounted"
    echo ""
    df -h "$LOCAL_MOUNT"
    exit 0
fi

# Create mount point
if [ ! -d "$LOCAL_MOUNT" ]; then
    echo "📁 Creating mount point..."
    sudo mkdir -p "$LOCAL_MOUNT"
    sudo chown $USER "$LOCAL_MOUNT"
fi

# Try SSHFS first (if macFUSE is approved)
echo "🔌 Attempting to mount via SSHFS..."
if command -v sshfs &> /dev/null; then
    sshfs "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH" "$LOCAL_MOUNT" \
        -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 2>/dev/null
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully mounted via SSHFS!"
        df -h "$LOCAL_MOUNT"
        ls -lh "$LOCAL_MOUNT" | head -5
        exit 0
    fi
fi

# If SSHFS fails, suggest alternatives
echo ""
echo "⚠️  SSHFS mount failed (macFUSE may not be approved)"
echo ""
echo "📋 Options:"
echo ""
echo "1️⃣  Approve macFUSE (requires one-time setup):"
echo "    • Connect via VNC/Screen Sharing to this Mac"
echo "    • Go to System Settings → Privacy & Security"
echo "   • Click 'Allow' next to macFUSE"
echo "    • Restart and run this script again"
echo ""
echo "2️⃣  Deploy on the remote Mac instead:"
echo "    ssh $REMOTE_USER@$REMOTE_HOST"
echo "    cd /srv && git clone https://github.com/costuff/do-you-music-analytics.git"
echo "    cd do-you-music-analytics && bash deployments/mac-mini/setup.sh"
echo ""
echo "3️⃣  Use NFS (run setup-nfs-mount.sh instead)"
echo ""

exit 1
