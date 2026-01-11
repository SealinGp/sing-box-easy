#!/bin/bash
#DOCKER_USERNAME=sealingp ADDITIONAL_TAGS="v1.0.0" bash scripts/deploy-docker.sh

# Docker deployment script - builds and pushes image to Docker Hub
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration with defaults
DOCKER_USERNAME="${DOCKER_USERNAME:-}"
IMAGE_NAME="${IMAGE_NAME:-sing-box-easy}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
BUILD_FRONTEND_LOCAL="${BUILD_FRONTEND_LOCAL:-true}"
PUSH_TO_HUB="${PUSH_TO_HUB:-true}"
ADDITIONAL_TAGS="${ADDITIONAL_TAGS:-}"  # Space-separated list of additional tags

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo -e "${BLUE}=== Docker Deploy Configuration ===${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Image Name: $IMAGE_NAME"
echo "Tag: $TAG"
echo "Platform: $PLATFORM"
echo "Build Frontend Locally: $BUILD_FRONTEND_LOCAL"
echo "Push to Hub: $PUSH_TO_HUB"
echo ""

# Check Docker Hub username
if [ "$PUSH_TO_HUB" = "true" ]; then
    if [ -z "$DOCKER_USERNAME" ]; then
        echo -e "${YELLOW}Docker Hub username not set. Please provide it:${NC}"
        read -p "Docker Hub username: " DOCKER_USERNAME
        if [ -z "$DOCKER_USERNAME" ]; then
            echo -e "${RED}Error: Docker Hub username is required${NC}"
            exit 1
        fi
    fi
    echo "Docker Hub Username: $DOCKER_USERNAME"

    # Check if logged in to Docker Hub
    if ! docker info 2>/dev/null | grep -q "Username: $DOCKER_USERNAME"; then
        echo -e "${YELLOW}Not logged in to Docker Hub. Attempting login...${NC}"
        docker login --username "$DOCKER_USERNAME"
        if [ $? -ne 0 ]; then
            echo -e "${RED}Docker Hub login failed${NC}"
            exit 1
        fi
    else
        echo -e "${GREEN}Already logged in to Docker Hub${NC}"
    fi
fi

# Change to project root
cd "$PROJECT_ROOT"

# Step 1: Build the image
echo ""
echo -e "${BLUE}=== Step 1: Building Docker Image ===${NC}"

if [ "$BUILD_FRONTEND_LOCAL" = "true" ]; then
    echo -e "${YELLOW}Building frontend locally first (avoids container issues)...${NC}"

    # Build frontend
    cd frontend
    if [ ! -d "node_modules" ]; then
        echo "Installing frontend dependencies..."
        npm ci
    fi

    echo "Building frontend..."
    npm run build

    if [ $? -ne 0 ]; then
        echo -e "${RED}Frontend build failed${NC}"
        exit 1
    fi

    cd ..
    echo -e "${GREEN}Frontend built successfully${NC}"

    # Create temporary Dockerfile for pre-built frontend
    cat > Dockerfile.prebuilt << 'EOF'
# Stage 1: Backend Builder
FROM golang:1.25.5-alpine AS backend-builder

WORKDIR /app

# Install git for valid go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary with CGO disabled (using pure Go sqlite)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags='-w -s -extldflags "-static"' -o sing-box-easy ./main.go

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests and tzdata for timezone support
# sqlite-libs is not needed for pure Go sqlite
RUN apk --no-cache add ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /etc/sing-box

# Copy the binary from builder
COPY --from=backend-builder /app/sing-box-easy .

# Copy the pre-built frontend
COPY dist ./dist

# Copy example config
COPY app.yml.example ./app.yml.example

# Expose port (can be overridden by HTTP_PORT env var)
EXPOSE 8080

# Set environment variables with defaults
ENV HTTP_PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${HTTP_PORT}/health || exit 1

# Run the application
CMD ["./sing-box-easy"]
EOF
    DOCKERFILE="Dockerfile.prebuilt"
else
    DOCKERFILE="Dockerfile"
fi

# Build with buildx
echo "Building Docker image with buildx..."
docker buildx build \
    --platform "$PLATFORM" \
    -t "$IMAGE_NAME:$TAG" \
    --load \
    -f "$DOCKERFILE" \
    .

if [ $? -ne 0 ]; then
    echo -e "${RED}Docker build failed${NC}"
    [ -f "Dockerfile.prebuilt" ] && rm -f Dockerfile.prebuilt
    exit 1
fi

# Cleanup temporary Dockerfile if created
[ -f "Dockerfile.prebuilt" ] && rm -f Dockerfile.prebuilt

echo -e "${GREEN}Docker image built successfully: $IMAGE_NAME:$TAG${NC}"

# Step 2: Tag and push to Docker Hub
if [ "$PUSH_TO_HUB" = "true" ]; then
    echo ""
    echo -e "${BLUE}=== Step 2: Pushing to Docker Hub ===${NC}"

    # Tag with Docker Hub username
    FULL_IMAGE_NAME="$DOCKER_USERNAME/$IMAGE_NAME"

    echo "Tagging image as $FULL_IMAGE_NAME:$TAG..."
    docker tag "$IMAGE_NAME:$TAG" "$FULL_IMAGE_NAME:$TAG"

    # Push the main tag
    echo "Pushing $FULL_IMAGE_NAME:$TAG to Docker Hub..."
    docker push "$FULL_IMAGE_NAME:$TAG"

    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to push image to Docker Hub${NC}"
        exit 1
    fi

    # Push additional tags if specified
    if [ -n "$ADDITIONAL_TAGS" ]; then
        for extra_tag in $ADDITIONAL_TAGS; do
            echo "Tagging and pushing $FULL_IMAGE_NAME:$extra_tag..."
            docker tag "$IMAGE_NAME:$TAG" "$FULL_IMAGE_NAME:$extra_tag"
            docker push "$FULL_IMAGE_NAME:$extra_tag"
        done
    fi

    echo -e "${GREEN}Successfully pushed to Docker Hub!${NC}"
    echo ""
    echo -e "${GREEN}=== Deployment Complete ===${NC}"
    echo "Image available at: https://hub.docker.com/r/$DOCKER_USERNAME/$IMAGE_NAME"
    echo ""
    echo "To pull and run the image:"
    echo "  docker pull $FULL_IMAGE_NAME:$TAG"
    echo "  docker run -d -p 8080:8080 -v /path/to/config:/etc/sing-box $FULL_IMAGE_NAME:$TAG"
else
    echo ""
    echo -e "${GREEN}=== Build Complete ===${NC}"
    echo "Image available locally as: $IMAGE_NAME:$TAG"
    echo ""
    echo "To run locally:"
    echo "  docker run -d -p 8080:8080 -v /path/to/config:/etc/sing-box $IMAGE_NAME:$TAG"
fi

# Optional: Show image details
echo ""
echo -e "${BLUE}=== Image Details ===${NC}"
docker images | head -n 1
docker images | grep "$IMAGE_NAME" | grep "$TAG"

# Show image size
IMAGE_SIZE=$(docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep "$IMAGE_NAME:$TAG" | awk '{print $2}')
echo "Image Size: $IMAGE_SIZE"

echo ""
echo -e "${GREEN}Done!${NC}"