FROM edoburu/pgbouncer:latest

USER root
RUN apk add --no-cache netcat-openbsd

COPY pgbouncer.ini /etc/pgbouncer/pgbouncer.ini
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

USER pgbouncer

EXPOSE 6432

ENTRYPOINT ["/entrypoint.sh"]
