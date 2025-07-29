FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o skimatik ./cmd/skimatic

# Final stage
FROM alpine:latest

# No runtime dependencies needed

# Create non-root user
RUN addgroup -g 1001 -S skimatik && \
    adduser -u 1001 -S skimatik -G skimatik

# Copy binary from builder
COPY --from=builder /app/skimatik /usr/local/bin/skimatik

# Switch to non-root user
USER skimatik

# Default command
ENTRYPOINT ["skimatik"] 