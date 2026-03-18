FROM golang:1.25.8 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/vault .

FROM alpine:3.23.3

WORKDIR /usr/local/bin

RUN apk add --no-cache curl \
    && addgroup -S vault \
    && adduser -S vault -G vault

COPY --from=builder /usr/local/bin/vault /usr/local/bin/vault

RUN chown vault:vault /usr/local/bin/vault && chmod 0755 /usr/local/bin/vault

USER vault:vault

EXPOSE 27462

ENTRYPOINT ["/usr/local/bin/vault"]


