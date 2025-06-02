# Etapa de build
FROM golang:1.20-alpine as builder

LABEL maintainer="rzrbld <razblade@gmail.com>"

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

RUN apk add --no-cache git

WORKDIR /app
RUN git clone https://github.com/MSJantana/zabbix-exporter-3000 .

RUN go mod tidy
RUN go build -o /go/bin/ze3000 main.go

FROM alpine:3.11

EXPOSE 9469

RUN mkdir /main && chmod 777 /main
WORKDIR /main

COPY --from=builder /go/bin/ze3000 /main/ze3000

RUN apk add --no-cache ca-certificates curl su-exec && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

CMD ["/main/ze3000"]
