# --- Stage 1: Build the binary ---
FROM golang:1.25-alpine AS builder

# Install build dependencies if needed
RUN apk add --no-cache git

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application
# CGO_ENABLED=0 ensures a static binary for smaller image compatibility and reduce c deps
RUN CGO_ENABLED=0 go build -o main .

# --- Stage 2: Production runtime ---
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/main .

# Command to run the executable
CMD ["./main"]
