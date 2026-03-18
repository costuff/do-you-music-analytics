#!/usr/bin/env bash
# Mount the remote music NAS on your dev machine.
# Run this once; the mount persists until you reboot or run unmount-nas.sh.
#
# Requirements:
#   macOS:  brew install macfuse sshfs  (or use SMB below)
#   Linux:  sudo apt install sshfs
#
# After mounting, set in .env.local:
#   NAS_MOUNT_PATH=/Volumes/RemoteMusic/doyou-world-radio-tracks

set -e

# ---- Config — adjust to your environment ----
REMOTE_USER="skoutelis"
REMOTE_HOST="${NAS_HOST:-100.110.101.64}"   # override with NAS_HOST env var
REMOTE_PATH="/mnt/mydrive/ExtHDD-8TB/Music"
LOCAL_MOUNT="/Volumes/RemoteMusic"
# ---------------------------------------------

TRACKS_DIR="$LOCAL_MOUNT/doyou-world-radio-tracks"

if mount | grep -q "$LOCAL_MOUNT"; then
    echo "✅ Already mounted at $LOCAL_MOUNT"
    echo "   Tracks path: $TRACKS_DIR"
    exit 0
fi

if [ ! -d "$LOCAL_MOUNT" ]; then
    echo "Creating mount point $LOCAL_MOUNT..."
    sudo mkdir -p "$LOCAL_MOUNT"
    sudo chown "$USER" "$LOCAL_MOUNT"
fi

# Prefer sshfs; fall back to macOS SMB if available
if command -v sshfs >/dev/null 2>&1; then
    echo "🔌 Mounting via SSHFS: $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH → $LOCAL_MOUNT"
    sshfs "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH" "$LOCAL_MOUNT" \
        -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3,follow_symlinks
elif command -v mount_smbfs >/dev/null 2>&1; then
    # macOS SMB — adjust share name for your NAS
    SMB_SHARE="${SMB_SHARE:-//skoutelis@mini.local/Music}"
    echo "🔌 Mounting via SMB: $SMB_SHARE → $LOCAL_MOUNT"
    mount_smbfs "$SMB_SHARE" "$LOCAL_MOUNT"
else
    echo "❌ Neither sshfs nor mount_smbfs found."
    echo "   Install sshfs: brew install macfuse sshfs"
    echo "   Or mount manually via Finder → Go → Connect to Server → smb://mini.local"
    exit 1
fi

echo ""
echo "✅ Mounted! Update .env.local:"
echo "   NAS_MOUNT_PATH=$TRACKS_DIR"
echo ""
echo "📁 Top-level contents:"
ls "$LOCAL_MOUNT" 2>/dev/null | head -5
