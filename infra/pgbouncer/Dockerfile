FROM edoburu/pgbouncer:latest

USER root
RUN apk add --no-cache netcat-openbsd && \
    adduser -D -H -u 1000 -s /sbin/nologin pgbouncer 2>/dev/null || true && \
    mkdir -p /etc/pgbouncer && \
    chown -R pgbouncer:pgbouncer /etc/pgbouncer && \
    chmod 700 /etc/pgbouncer

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && chown pgbouncer:pgbouncer /entrypoint.sh

USER pgbouncer

EXPOSE 6432

ENTRYPOINT ["/entrypoint.sh"]
