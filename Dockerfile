# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy frontend source
COPY frontend/ ./

# Build frontend (output to /app/dist)
RUN npm run build


# Stage 2: Build Backend
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

# Copy built frontend from previous stage
COPY --from=frontend-builder /app/dist ./dist

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sing-box-easy ./main.go


# Stage 3: Runtime
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests and tzdata for timezone support
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

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${HTTP_PORT}/health || exit 1

# Run the application
CMD ["./sing-box-easy"]
