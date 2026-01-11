#!/bin/bash

# Deploy script for Podman using Kubernetes YAML
# Usage: bash scripts/deploy-podman.sh

set -e

YAML_FILE="sing-box-easy.yaml"

# Colors for output
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Deploying to Podman ===${NC}"

# Check if the YAML file exists
if [ ! -f "$YAML_FILE" ]; then
    echo "Error: $YAML_FILE not found in current directory."
    echo "Please run this script from the project root."
    exit 1
fi

# Stop existing pod if it exists
if podman pod exists sing-box-easy; then
    echo "Stopping existing pod..."
    podman pod stop sing-box-easy
    echo "Removing existing pod..."
    podman pod rm sing-box-easy
fi

# Run the pod
echo "Playing kube YAML..."
podman play kube $YAML_FILE

echo -e "${GREEN}=== Deployment Successful ===${NC}"
echo "Pod 'sing-box-easy' is running."
echo "Check status with: podman pod ps"
echo "Check logs with: podman logs sing-box-easy-sing-box-easy" # Container name is usually podname-containername
