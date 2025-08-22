# Updated Dockerfile for SQLC-based API
FROM golang:1.24.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Install SQLC
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate SQLC code
RUN sqlc generate

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags netgo \
    -ldflags '-w -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/api

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy any necessary files (like migrations if you want to run them in container)
COPY --from=builder /app/sql ./sql

# Expose port
EXPOSE 3333

# Run the binary
CMD ["./main"]