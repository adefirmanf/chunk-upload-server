# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files if they exist
COPY go.* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Run stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Create upload directory
RUN mkdir -p /root/tmp

# Expose port
EXPOSE 8090

# Run the application
CMD ["./main"]
