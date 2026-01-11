# Stage 1: Build Frontend
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder

WORKDIR /app

# Copy frontend package files
COPY frontend/package*.json ./frontend/

# Install dependencies
WORKDIR /app/frontend
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Set NODE_OPTIONS to increase memory limit and disable certain optimizations
# This helps prevent esbuild crashes in container environments
ENV NODE_OPTIONS="--max-old-space-size=4096"

# Build frontend with increased memory and single-threaded mode to avoid concurrency issues
# This will output to /app/dist (../dist from /app/frontend)
RUN npm run build

# Stage 2: Build Backend
FROM --platform=$BUILDPLATFORM golang:alpine AS backend-builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies including gcc for CGO
# Install git for valid go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/dist ./dist

# Build the Go binary with CGO enabled for SQLite support
# Build the Go binary with CGO disabled (using pure Go sqlite)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -a -ldflags='-w -s -extldflags "-static"' -o sing-box-easy ./main.go


# Stage 3: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests and tzdata for timezone support
# sqlite-libs is not needed for pure Go sqlite
RUN apk --no-cache add ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /etc/sing-box

# Copy the binary from builder
COPY --from=backend-builder /app/sing-box-easy .

# Copy the built frontend
COPY --from=backend-builder /app/dist ./dist

# Copy example config
COPY app.yml.example ./app.yml.example

# Expose port (can be overridden by HTTP_PORT env var)
EXPOSE 8080

# Set environment variables with defaults
ENV HTTP_PORT=8080
ENV ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${HTTP_PORT}/health || exit 1

# Run the application
CMD ["./sing-box-easy"]