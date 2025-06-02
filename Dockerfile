# Etapa de build
FROM golang:1.21-alpine AS builder

LABEL maintainer="rzrbld <razblade@gmail.com>"

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=https://proxy.golang.org

WORKDIR /app

# Instala git e certifica-se de que todas dependências estão resolvidas
RUN apk add --no-cache git

# Clona o repositório corrigido
RUN git clone https://github.com/MSJantana/zabbix-exporter-3000 .

# Baixa dependências e compila o binário
RUN go mod tidy && go build -o ze3000 main.go

# Etapa final
FROM alpine:3.11

EXPOSE 8080

RUN apk add --no-cache \
      ca-certificates \
      curl \
      su-exec && \
    echo 'hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4' >> /etc/nsswitch.conf

# Cria diretório de execução
WORKDIR /main
COPY --from=builder /app/ze3000 /main/ze3000

CMD ["/main/ze3000"]
