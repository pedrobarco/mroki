# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mroki-api ./cmd/mroki-api

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/mroki-api .

# Skip .env file loading in container
ENV APP_ENV=production

# Expose default port
EXPOSE 8090

CMD ["./mroki-api"]
