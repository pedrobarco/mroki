FROM caddy:builder AS builder

RUN xcaddy build \
    --with github.com/pedrobarco/mroki/cmd/caddy-mroki

FROM caddy:latest

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
