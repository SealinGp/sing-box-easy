# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Set NODE_OPTIONS to increase memory limit and disable certain optimizations
# This helps prevent esbuild crashes in container environments
ENV NODE_OPTIONS="--max-old-space-size=4096"

# Build frontend with increased memory and single-threaded mode to avoid concurrency issues
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.25.5-alpine AS backend-builder

WORKDIR /app

# Install build dependencies including gcc for CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/dist ./dist

# Build the Go binary with CGO enabled for SQLite support
# Note: CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -ldflags='-w -s -extldflags "-static"' -o sing-box-easy ./main.go


# Stage 3: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests, tzdata for timezone support, and sqlite-libs for runtime
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

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

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${HTTP_PORT}/health || exit 1

# Run the application
CMD ["./sing-box-easy"]