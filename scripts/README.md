# Build Scripts

This directory contains utility scripts for building and deploying sing-box-easy.

## build.sh

Docker build script for creating x86_64 (amd64) images on macOS.

### Usage

```bash
# Basic build
./scripts/build.sh

# Build with custom tag
./scripts/build.sh -t v1.0.0

# Build without using cache
./scripts/build.sh --no-cache

# Build and push to registry
./scripts/build.sh -t v1.0.0 -p
```

### Features

- **Cross-platform support**: Works on both Intel and Apple Silicon Macs
- **Automatic buildx setup**: Creates and configures Docker buildx builder
- **Platform targeting**: Builds for linux/amd64 (x86_64) architecture
- **Flexible options**: Support for custom tags, push to registry, cache control
- **Error handling**: Validates Docker availability and build status

### Options

| Option | Description |
|--------|-------------|
| `-t, --tag TAG` | Image tag (default: latest) |
| `-n, --name NAME` | Image name (default: sing-box-easy) |
| `-p, --push` | Push image to registry after build |
| `-l, --load` | Load image to local docker (default: true) |
| `--no-load` | Don't load image to local docker |
| `--no-cache` | Build without using cache |
| `-h, --help` | Display help message |

### Examples

```bash
# Development build
./scripts/build.sh -t dev

# Production build with push
./scripts/build.sh -t v1.2.3 -p

# Clean build without cache
./scripts/build.sh --no-cache

# Build but don't load locally (for CI/CD)
./scripts/build.sh -t ci-build --no-load -p
```

### Requirements

- Docker Desktop for Mac
- Docker buildx (included in Docker Desktop)

### Notes

- The script automatically creates a buildx builder named `multiarch-builder` if it doesn't exist
- Built images are loaded to local Docker by default (can be disabled with `--no-load`)
- The script validates that Docker is running before attempting to build
