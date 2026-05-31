# Build stage
# cmd/caddy-mroki is a self-contained Caddy build (standard modules + the mroki
# gate handler), so a plain `go build` replaces xcaddy.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o caddy ./cmd/caddy-mroki

FROM caddy:latest
COPY --from=builder /src/caddy /usr/bin/caddy

# The freshly built binary carries no file capabilities, so grant
# cap_net_bind_service to keep the ability to bind privileged ports (e.g.
# :80/:443) when running as non-root. Also create a non-root user and grant it
# ownership of Caddy's config/data directories so storage and runtime config
# work without root.
RUN apk add --no-cache libcap \
    && setcap cap_net_bind_service=+ep /usr/bin/caddy \
    && apk del libcap \
    && addgroup -g 1000 mroki \
    && adduser -u 1000 -G mroki -D mroki \
    && chown -R mroki:mroki /config /data

# Run as a non-root user
USER 1000
