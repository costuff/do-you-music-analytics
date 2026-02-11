#!/bin/bash
# Setup NFS export on remote Mac for music directory

REMOTE_USER="skoutelis"
REMOTE_HOST="100.110.101.64"
REMOTE_PATH="/mnt/mydrive/ExtHDD-8TB/Music"
LOCAL_MOUNT="/Volumes/RemoteMusic"
YOUR_MAC_IP=$(ipconfig getifaddr en0 || ipconfig getifaddr en1)

echo "🔧 Setting up NFS mount from remote Mac..."
echo ""

# Configure NFS on remote machine
ssh $REMOTE_USER@$REMOTE_HOST "sudo bash" << 'ENDSSH'
# Add NFS export if not already present
if ! grep -q "/mnt/mydrive/ExtHDD-8TB/Music" /etc/exports 2>/dev/null; then
    echo "/mnt/mydrive/ExtHDD-8TB/Music -alldirs -maproot=root:wheel -network 100.110.101.0 -mask 255.255.255.0" | sudo tee -a /etc/exports
    echo "✅ Added NFS export"
else
    echo "✅ NFS export already exists"
fi

# Enable and restart NFS
sudo nfsd enable
sudo nfsd restart
echo "✅ NFS server restarted"
ENDSSH

if [ $? -ne 0 ]; then
    echo "❌ Failed to configure NFS on remote machine"
    exit 1
fi

# Create mount point
if [ ! -d "$LOCAL_MOUNT" ]; then
    sudo mkdir -p "$LOCAL_MOUNT"
fi

# Mount via NFS
echo "🔌 Mounting via NFS..."
sudo mount -t nfs -o resvport $REMOTE_HOST:$REMOTE_PATH $LOCAL_MOUNT

if [ $? -eq 0 ]; then
    echo "✅ Successfully mounted!"
    echo ""
    df -h "$LOCAL_MOUNT"
    echo ""
    echo "📁 Contents:"
    ls -lh "$LOCAL_MOUNT" | head -10
else
    echo "❌ Mount failed"
    exit 1
fi
