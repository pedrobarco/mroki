FROM arigaio/atlas:1.2.0-community AS atlas

FROM alpine:3.21

COPY --from=atlas /atlas /usr/local/bin/atlas

COPY ent/migrate/migrations/ /migrations/

ENTRYPOINT ["atlas", "migrate", "apply", "--dir", "file:///migrations"]
