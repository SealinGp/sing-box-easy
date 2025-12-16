# Docker Deployment Guide

This guide explains how to build and deploy sing-box-easy using Docker.

## Quick Start with Docker Compose

1. **Prepare configuration directory**:
   ```bash
   mkdir -p config
   ```

2. **Create app.yml** from the example:
   ```bash
   cp app.yml.example app.yml
   ```

3. **Edit app.yml** to configure paths for the containerized environment:
   ```yaml
   server:
     port: "8080"

   sing_box:
     config_path: "/etc/sing-box/config.json"
     binary_path: "sing-box"  # Will need to install sing-box in container
     subscription_path: "/etc/sing-box/subscriptions.json"
     init_state_path: "/etc/sing-box/init_state.json"
   ```

4. **Build and run**:
   ```bash
   docker-compose up -d
   ```

5. **Access the application**:
   - Open http://localhost:8080 in your browser
   - API endpoint: http://localhost:8080/api/1.12.12/

## Manual Docker Build

### Using the Build Script (Recommended for macOS)

The project includes a build script that handles cross-platform builds for x86_64 architecture on macOS:

```bash
# Basic build
./scripts/build.sh

# Build with custom tag
./scripts/build.sh -t v1.0.0

# Build without cache
./scripts/build.sh --no-cache

# Build and push to registry
./scripts/build.sh -t v1.0.0 -p

# Show all options
./scripts/build.sh --help
```

The script automatically:
- Sets up Docker buildx for cross-platform builds
- Builds for linux/amd64 (x86_64) platform
- Loads the image to your local Docker
- Works on both Intel and Apple Silicon Macs

### Direct Docker Build

If you prefer to build directly without the script:

```bash
# Build for x86_64/amd64 platform
docker buildx build --platform linux/amd64 -t sing-box-easy:latest --load .

# Or use regular docker build
docker build -t sing-box-easy:latest .
```

### Run the container:
```bash
docker run -d \
  --name sing-box-easy \
  -p 8080:8080 \
  -v $(pwd)/config:/etc/sing-box \
  -v $(pwd)/app.yml:/app/app.yml:ro \
  -e HTTP_PORT=8080 \
  sing-box-easy:latest
```

## Architecture

The Docker build uses a multi-stage process:

1. **Frontend Builder** (node:22-alpine)
   - Installs frontend dependencies
   - Builds Vue.js app
   - Outputs to `/app/dist`

2. **Backend Builder** (golang:1.25-alpine)
   - Downloads Go dependencies
   - Copies built frontend from stage 1
   - Compiles Go binary

3. **Runtime** (alpine:latest)
   - Minimal runtime environment
   - Contains only the binary and built frontend
   - Exposes port 8080

## Environment Variables

- `HTTP_PORT`: HTTP server port (default: 8080)

## Volumes

- `/etc/sing-box`: Configuration directory for sing-box
  - `config.json`: Main sing-box configuration
  - `subscriptions.json`: Subscription data
  - `init_state.json`: Initialization state
- `/app/app.yml`: Application configuration

## Health Check

The container includes a health check that pings `/health` endpoint every 30 seconds.

## Notes

- sing-box binary needs to be installed separately in the container or mounted as a volume
- The frontend is built during Docker image creation and served by the Go backend
- All static files are served from the `/dist` directory
- Configuration files should be mounted as volumes for persistence

## Production Considerations

1. **sing-box Installation**: You may need to extend the Dockerfile to install sing-box:
   ```dockerfile
   RUN wget https://github.com/SagerNet/sing-box/releases/download/vX.Y.Z/sing-box-X.Y.Z-linux-amd64.tar.gz \
       && tar -xzf sing-box-X.Y.Z-linux-amd64.tar.gz \
       && mv sing-box-X.Y.Z-linux-amd64/sing-box /usr/local/bin/ \
       && rm -rf sing-box-X.Y.Z-linux-amd64*
   ```

2. **Security**: Consider running as non-root user
3. **Reverse Proxy**: Use nginx or traefik in front for HTTPS
4. **Persistent Storage**: Ensure `/etc/sing-box` is properly backed up
