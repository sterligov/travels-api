FROM golang:1.13 as go-container

ARG WORKPATH=/app
RUN mkdir -p $WORKPATH
WORKDIR $WORKPATH

COPY go.mod $WORKPATH
COPY go.sum $WORKPATH
RUN go mod download
COPY . $WORKPATH

RUN go build -o travels-api

FROM mysql:8 as builder

RUN ["sed", "-i", "s/exec \"$@\"/echo \"not running $@\"/", "/usr/local/bin/docker-entrypoint.sh"]

ENV MYSQL_DATABASE api_db
ENV MYSQL_USER api_user
ENV MYSQL_PASSWORD secret_password
ENV MYSQL_ROOT_PASSWORD root

COPY ./tables.sql /docker-entrypoint-initdb.d/

RUN ["/usr/local/bin/docker-entrypoint.sh", "mysqld", "--datadir", "/initialized-db"]

FROM mysql:8

COPY --from=builder /initialized-db /var/lib/mysql
COPY ./.env .
# RUN mkdir -p /tmp/data
# COPY ./data.zip /tmp/data/data.zip
COPY --from=go-container /app/travels-api .

EXPOSE 80

COPY ./docker-entrypoint.sh /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

