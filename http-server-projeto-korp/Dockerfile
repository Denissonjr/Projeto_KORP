# ---- build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copia primeiro o go.mod para aproveitar cache de camadas do Docker
COPY go.mod ./

# Copia o código-fonte
COPY main.go ./

# Resolve dependências (gera/atualiza go.sum) e baixa os módulos necessários
RUN go mod tidy

# Compila um binário estático, sem CGO, para rodar em imagem mínima
RUN CGO_ENABLED=0 GOOS=linux go build -o http-server-projeto-korp .

# ---- runtime stage ----
FROM alpine:3.19

# Certificados para eventuais chamadas HTTPS futuras e curl para healthcheck
RUN apk add --no-cache ca-certificates curl

WORKDIR /app
COPY --from=builder /app/http-server-projeto-korp .

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./http-server-projeto-korp"]
