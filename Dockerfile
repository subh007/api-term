# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o api-term cli/main.go cli/utils.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (if any)
# ca-certificates is needed for HTTPS requests
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/api-term .

# Create assets directory if needed (though user provided URLs might skip this)
RUN mkdir -p assets

# Set entrypoint so user can pass flags
ENTRYPOINT ["/app/api-term"]
