# Build stage
FROM node:22-alpine AS builder

RUN corepack enable && corepack prepare pnpm@10 --activate

WORKDIR /app

# Copy package files for dependency caching
COPY web/mroki-hub/package.json web/mroki-hub/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy source code
COPY web/mroki-hub/ .

# Build the application
RUN pnpm build

# Runtime stage
FROM caddy:alpine

# Copy Caddyfile
COPY build/package/mroki-hub/Caddyfile /etc/caddy/Caddyfile

# Copy built assets from builder
COPY --from=builder /app/dist /srv

# Copy entrypoint script for runtime config injection
COPY build/package/mroki-hub/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Create a non-root user and grant it ownership of the serving directory so the
# entrypoint can write /srv/config.js when running with runAsNonRoot.
RUN addgroup -g 1000 mroki \
	&& adduser -u 1000 -G mroki -D mroki \
	&& chown -R mroki:mroki /srv
USER 1000

# Expose default port
EXPOSE 80

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile"]
