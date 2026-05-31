FROM arigaio/atlas:1.2.0-community AS atlas

FROM alpine:3.21

RUN addgroup -g 1000 mroki \
    && adduser -u 1000 -G mroki -D mroki

COPY --from=atlas /atlas /usr/local/bin/atlas

COPY ent/migrate/migrations/ /migrations/

# Run as a non-root user
USER 1000

ENTRYPOINT ["atlas", "migrate", "apply", "--dir", "file:///migrations"]
