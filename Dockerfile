# ---- build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o url-shortener ./cmd/url-shortener

# ---- runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app && \
    mkdir -p /app/data /app/config

WORKDIR /app
COPY --from=builder /app/url-shortener .
RUN chown -R app:app /app

USER app
EXPOSE 8082

# Абсолютный путь - не зависит от того, что смонтировано в /app
ENTRYPOINT ["/app/url-shortener"]