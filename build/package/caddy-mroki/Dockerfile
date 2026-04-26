FROM caddy:builder AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN xcaddy build \
    --with github.com/pedrobarco/mroki/pkg/caddymodule=./pkg/caddymodule

FROM caddy:latest
COPY --from=builder /usr/bin/caddy /usr/bin/caddy
