#!/bin/bash

# BUILD_FRONTEND_LOCAL=true bash scripts/build-docker.sh

# Docker build script with optimizations for esbuild issues
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="${IMAGE_NAME:-sing-box-easy}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
BUILDER_NAME="${BUILDER_NAME:-multiarch-builder}"

echo -e "${GREEN}=== Docker Build Configuration ===${NC}"
echo "Image Name: $IMAGE_NAME"
echo "Tag: $TAG"
echo "Platform: $PLATFORM"

# Check if we should build frontend locally first
if [ "$BUILD_FRONTEND_LOCAL" = "true" ]; then
    echo -e "${YELLOW}=== Building Frontend Locally First ===${NC}"
    echo "This avoids esbuild issues in Docker containers"

    cd frontend
    npm ci
    npm run build
    cd ..

    echo -e "${GREEN}Frontend built successfully locally${NC}"

    # Use a different Dockerfile that skips frontend build
    DOCKERFILE="Dockerfile.prebuilt"

    # Create the alternate Dockerfile
    cat > $DOCKERFILE << 'EOF'
# Stage 1: Backend Builder
# Use --platform=$BUILDPLATFORM to run the builder natively on the host (e.g. Apple Silicon)
# regardless of the target platform. This prevents slow QEMU emulation for Go builds.
FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine AS backend-builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the Go binary
# Build the Go binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -installsuffix cgo -o sing-box-easy ./main.go

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests and tzdata for timezone support
RUN apk --no-cache add ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /etc/sing-box

# Copy the binary from builder
COPY --from=backend-builder /app/sing-box-easy .

# Copy the pre-built frontend
COPY --from=backend-builder /app/dist ./dist

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
else
    DOCKERFILE="Dockerfile"
fi

# Setup buildx if not exists
if ! docker buildx ls | grep -q $BUILDER_NAME; then
    echo -e "${YELLOW}Creating new buildx builder: $BUILDER_NAME${NC}"
    docker buildx create --name $BUILDER_NAME --use
    docker buildx inspect --bootstrap
else
    echo -e "${GREEN}Using existing buildx builder: $BUILDER_NAME${NC}"
    docker buildx use $BUILDER_NAME
fi

# Build the image
echo -e "${GREEN}=== Building Docker Image ===${NC}"
echo "Using Dockerfile: $DOCKERFILE"

if docker buildx build \
    --platform $PLATFORM \
    -t $IMAGE_NAME:$TAG \
    --load \
    -f $DOCKERFILE \
    . ; then
    echo -e "${GREEN}=== Build Successful ===${NC}"
    echo "Image: $IMAGE_NAME:$TAG"

    # Show image info
    docker images | grep $IMAGE_NAME

    # Cleanup temporary Dockerfile if created
    if [ "$BUILD_FRONTEND_LOCAL" = "true" ]; then
        rm -f $DOCKERFILE
    fi
else
    echo -e "${RED}=== Build Failed ===${NC}"

    # Cleanup temporary Dockerfile if created
    if [ "$BUILD_FRONTEND_LOCAL" = "true" ]; then
        rm -f $DOCKERFILE
    fi

    exit 1
fi

echo -e "${GREEN}=== Docker image built successfully ===${NC}"
echo ""
echo "To run the container:"
echo "  docker run -d -p 8080:8080 -v /path/to/config:/etc/sing-box $IMAGE_NAME:$TAG"
echo ""
echo "To push to registry:"
echo "  docker push $IMAGE_NAME:$TAG"