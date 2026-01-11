#!/bin/bash

# Remote server deployment script for sing-box-easy
# Usage: bash deploy-remote.sh

REMOTE_HOST="root@192.168.31.200"
CONTAINER_NAME="sing-box-easy"
IMAGE="sealingp/sing-box-easy:latest"
PORT="8001"
CONFIG_PATH="/mnt/sata2-2/docker/singbox-easy"

echo "=== Deploying sing-box-easy to $REMOTE_HOST ==="

# Commands to run on remote server
ssh $REMOTE_HOST << 'ENDSSH'
echo "1. Stopping and removing existing container..."
docker stop sing-box-easy 2>/dev/null || true
docker rm sing-box-easy 2>/dev/null || true

echo "2. Removing old image..."
docker rmi sealingp/sing-box-easy:latest 2>/dev/null || true

echo "3. Pulling latest image..."
docker pull sealingp/sing-box-easy:latest

echo "4. Creating config file if not exists..."
cd /mnt/sata2-2/docker/singbox-easy
if [ ! -f app.yml ]; then
    cat > app.yml << 'EOF'
server:
  port: 8080

sing-box:
  config_path: /etc/sing-box/config.json
  binary_path: sing-box
  dashboard_path: /etc/sing-box/dashboard
  database_path: /etc/sing-box/sing-box-easy.db
EOF
fi

echo "5. Starting new container..."
docker run -d \
  --name sing-box-easy \
  --restart unless-stopped \
  -p 8001:8080 \
  -v /mnt/sata2-2/docker/singbox-easy:/etc/sing-box \
  -v /mnt/sata2-2/docker/singbox-easy/app.yml:/app/app.yml:ro \
  sealingp/sing-box-easy:latest

echo "6. Waiting for container to start..."
sleep 5

echo "7. Checking container status..."
docker ps | grep sing-box-easy

echo "8. Testing web interface..."
curl -I http://localhost:8001/ 2>/dev/null | head -n 1

echo "9. Container logs (last 10 lines):"
docker logs sing-box-easy 2>&1 | tail -10

ENDSSH

echo ""
echo "=== Deployment complete ==="
echo "Access the web interface at: http://192.168.31.200:8001/"