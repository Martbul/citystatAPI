# Test Dockerfile to simulate Render's build environment
FROM golang:1.24.4-alpine

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Test the exact build command
RUN go run github.com/steebchen/prisma-client-go generate && \
    go build -tags netgo -ldflags '-s -w' -o app

# Test that the binary was created and works
RUN ls -la app && \
    file app

CMD ["./app"]