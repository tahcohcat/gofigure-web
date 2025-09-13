# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies for CGO and SQLite
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Create data directory
RUN mkdir -p /data
ENV DATABASE_PATH=/data/mydb.sqlite

# Build the application WITH CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o main cmd/server/main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install ca-certificates and sqlite runtime
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy static files and data
COPY --from=builder /app/web ./web
COPY --from=builder /app/data ./data
COPY --from=builder /app/config.yaml ./config.yaml

# Create a non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

# Create data directory and set permissions
RUN mkdir -p /data && \
    chown -R appuser:appgroup /app /data

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]