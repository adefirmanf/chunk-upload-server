# Build stage
FROM --platform=linux/amd64 golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files if they exist
COPY go.* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Run stage
FROM --platform=linux/amd64 alpine:latest

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
