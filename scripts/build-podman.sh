#!/bin/bash

# BUILD_FRONTEND_LOCAL=true bash scripts/build-podman.sh

# Podman build script with optimizations for esbuild issues
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

echo -e "${GREEN}=== Podman Build Configuration ===${NC}"
echo "Image Name: $IMAGE_NAME"
echo "Tag: $TAG"
echo "Platform: $PLATFORM"

# Check if we should build frontend locally first
if [ "$BUILD_FRONTEND_LOCAL" = "true" ]; then
    echo -e "${YELLOW}=== Building Frontend Locally First ===${NC}"
    echo "This avoids esbuild issues in container environments"

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
FROM golang:1.25.5-alpine AS backend-builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sing-box-easy ./main.go

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

# Build the image
echo -e "${GREEN}=== Building Podman Image ===${NC}"
echo "Using Dockerfile: $DOCKERFILE"

# Podman build command
if podman build \
    --platform $PLATFORM \
    -t $IMAGE_NAME:$TAG \
    -f $DOCKERFILE \
    . ; then
    echo -e "${GREEN}=== Build Successful ===${NC}"
    echo "Image: $IMAGE_NAME:$TAG"

    # Show image info
    podman images | grep $IMAGE_NAME

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

echo -e "${GREEN}=== Podman image built successfully ===${NC}"
echo ""
echo "To run the container:"
echo "  podman run -d -p 8080:8080 -v /path/to/config:/etc/sing-box $IMAGE_NAME:$TAG"
echo ""
echo "To push to registry:"
echo "  podman push $IMAGE_NAME:$TAG"
