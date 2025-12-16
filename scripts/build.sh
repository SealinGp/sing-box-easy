#!/bin/bash

# Docker Build Script for x86_64 (amd64) Architecture
# This script builds the sing-box-easy Docker image for x86_64/amd64 platform
# Works on both Intel and Apple Silicon Macs

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
IMAGE_NAME="sing-box-easy"
TAG="latest"
PLATFORM="linux/amd64"
PUSH=false
LOAD=true

# Print usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build Docker image for x86_64 (amd64) architecture on macOS

OPTIONS:
    -t, --tag TAG           Image tag (default: latest)
    -n, --name NAME         Image name (default: sing-box-easy)
    -p, --push              Push image to registry after build
    -l, --load              Load image to local docker (default: true)
    --no-load               Don't load image to local docker
    --no-cache              Build without using cache
    -h, --help              Display this help message

EXAMPLES:
    # Build with default settings
    $0

    # Build with custom tag
    $0 -t v1.0.0

    # Build and push to registry
    $0 -t v1.0.0 -p

    # Build without cache
    $0 --no-cache

EOF
    exit 0
}

# Parse arguments
NO_CACHE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        -l|--load)
            LOAD=true
            shift
            ;;
        --no-load)
            LOAD=false
            shift
            ;;
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo -e "${GREEN}=== Docker Build Configuration ===${NC}"
echo "Image Name: $IMAGE_NAME"
echo "Tag: $TAG"
echo "Platform: $PLATFORM"
echo "Push: $PUSH"
echo "Load to local: $LOAD"
echo "Project Root: $PROJECT_ROOT"
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker Desktop.${NC}"
    exit 1
fi

# Check if buildx is available
if ! docker buildx version > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker buildx is not available. Please update Docker Desktop.${NC}"
    exit 1
fi

# Create buildx builder if it doesn't exist
BUILDER_NAME="multiarch-builder"
if ! docker buildx inspect "$BUILDER_NAME" > /dev/null 2>&1; then
    echo -e "${YELLOW}Creating buildx builder: $BUILDER_NAME${NC}"
    docker buildx create --name "$BUILDER_NAME" --use
else
    echo -e "${YELLOW}Using existing buildx builder: $BUILDER_NAME${NC}"
    docker buildx use "$BUILDER_NAME"
fi

# Bootstrap the builder
docker buildx inspect --bootstrap

# Prepare build arguments
BUILD_ARGS="--platform $PLATFORM"
BUILD_ARGS="$BUILD_ARGS -t ${IMAGE_NAME}:${TAG}"

if [ "$LOAD" = true ]; then
    BUILD_ARGS="$BUILD_ARGS --load"
fi

if [ "$PUSH" = true ]; then
    BUILD_ARGS="$BUILD_ARGS --push"
fi

if [ -n "$NO_CACHE" ]; then
    BUILD_ARGS="$BUILD_ARGS $NO_CACHE"
fi

# Build the image
echo -e "${GREEN}=== Building Docker Image ===${NC}"
echo "Command: docker buildx build $BUILD_ARGS ."
echo ""

if docker buildx build $BUILD_ARGS .; then
    echo ""
    echo -e "${GREEN}=== Build Successful ===${NC}"
    echo "Image: ${IMAGE_NAME}:${TAG}"
    echo "Platform: $PLATFORM"

    if [ "$LOAD" = true ]; then
        echo ""
        echo -e "${GREEN}Image loaded to local Docker.${NC}"
        echo "You can run it with:"
        echo "  docker run -d -p 8080:8080 \\"
        echo "    -v \$(pwd)/config:/etc/sing-box \\"
        echo "    -v \$(pwd)/app.yml:/app/app.yml:ro \\"
        echo "    ${IMAGE_NAME}:${TAG}"
    fi

    if [ "$PUSH" = true ]; then
        echo ""
        echo -e "${GREEN}Image pushed to registry.${NC}"
    fi
else
    echo ""
    echo -e "${RED}=== Build Failed ===${NC}"
    exit 1
fi
