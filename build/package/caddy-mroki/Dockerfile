FROM caddy:builder AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN xcaddy build \
    --with github.com/pedrobarco/mroki/pkg/caddymodule=./pkg/caddymodule

FROM caddy:latest
COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# COPY drops file capabilities, so re-apply cap_net_bind_service to keep the
# ability to bind privileged ports (e.g. :80/:443) when running as non-root.
# Also create a non-root user and grant it ownership of Caddy's config/data
# directories so storage and runtime config work without root.
RUN apk add --no-cache libcap \
    && setcap cap_net_bind_service=+ep /usr/bin/caddy \
    && apk del libcap \
    && addgroup -g 1000 mroki \
    && adduser -u 1000 -G mroki -D mroki \
    && chown -R mroki:mroki /config /data

# Run as a non-root user
USER 1000
