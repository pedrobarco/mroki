# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Cross-compile the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o mroki-api ./cmd/mroki-api

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates \
    && addgroup -g 1000 mroki \
    && adduser -u 1000 -G mroki -D mroki

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/mroki-api .

# Skip .env file loading in container
ENV APP_ENV=production

# Run as a non-root user
USER 1000

# Expose default port
EXPOSE 8090

CMD ["./mroki-api"]
