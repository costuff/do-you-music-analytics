# Mac Mini Deployment Configuration

This directory contains Mac mini specific deployment files and scripts.

## Files

- **`.env.example`**: Template environment configuration (copy to project root as `.env`)
- **`setup.sh`**: Initial setup script for Mac mini
- **`start.sh`**: Start services with production config
- **`stop.sh`**: Gracefully stop services
- **`update.sh`**: Manually trigger update check

## Quick Start

```bash
# On Mac mini
cd ~/projects/do-you-music-analytics

# Copy environment template
cp deployments/mac-mini/.env.example .env

# Edit with your values
nano .env

# Run setup script
bash deployments/mac-mini/setup.sh

# Start services
bash deployments/mac-mini/start.sh
```

## Directory Structure

```
deployments/mac-mini/
├── .env.example          # Environment variables template
├── README.md            # This file
├── setup.sh             # Initial setup script
├── start.sh             # Start services
├── stop.sh              # Stop services
└── update.sh            # Manual update trigger
```

## Notes

- Ensure NAS path in `.env` matches your actual mount point
- Adjust `MAX_WORKERS` based on CPU cores available
- Watchtower checks for updates every 5 minutes automatically
- View logs: `docker-compose -f docker-compose.prod.yml logs -f`
