FROM edoburu/pgbouncer:latest

USER root
RUN apk add --no-cache netcat-openbsd && \
    id pgbouncer 2>/dev/null || adduser -D -H -s /sbin/nologin pgbouncer && \
    mkdir -p /etc/pgbouncer && \
    chown pgbouncer:pgbouncer /etc/pgbouncer

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER pgbouncer

EXPOSE 6432

ENTRYPOINT ["/entrypoint.sh"]
