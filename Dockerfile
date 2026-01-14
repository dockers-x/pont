# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BUILD_TIME
ARG GIT_COMMIT

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with embedded assets and inject version info
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static' \
        -X 'pont/version.Version=${VERSION}' \
        -X 'pont/version.BuildTime=${BUILD_TIME}' \
        -X 'pont/version.GitCommit=${GIT_COMMIT}'" \
    -o pont .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder (assets are embedded)
COPY --from=builder /build/pont .

# Create data and logs directories with proper ownership
RUN mkdir -p /app/data /app/logs && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 13333

# Set environment variables
ENV PORT=13333
ENV DATA_DIR=/app/data
ENV LOG_DIR=/app/logs

# Run the application
CMD ["./pont"]
