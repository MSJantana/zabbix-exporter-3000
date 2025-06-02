# Etapa de build
FROM golang:1.14-alpine as builder

LABEL maintainer="rzrbld <razblade@gmail.com>"

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

# Instalando dependências
RUN apk add --no-cache git

# Clonando o repositório e compilando
WORKDIR /app
RUN git clone https://github.com/MSJantana/zabbix-exporter-3000 .
WORKDIR /app/zabbix

RUN go mod tidy
RUN go build -o /go/bin/ze3000 main.go

# Imagem final
FROM alpine:3.11

EXPOSE 9469

RUN mkdir /main && chmod 777 /main
WORKDIR /main

COPY --from=builder /go/bin/ze3000 /main/ze3000

RUN apk add --no-cache ca-certificates curl su-exec && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

CMD ["/main/ze3000"]
