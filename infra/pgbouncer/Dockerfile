FROM edoburu/pgbouncer:latest

USER root
RUN apk add --no-cache netcat-openbsd && \
    mkdir -p /etc/pgbouncer && \
    chmod 777 /etc/pgbouncer

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 6432

ENTRYPOINT ["/entrypoint.sh"]
