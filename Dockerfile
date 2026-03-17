FROM golang:1.25.8 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/vault .

FROM alpine:latest

WORKDIR /usr/local/bin

RUN apk add --no-cache curl

COPY --from=builder /usr/local/bin/vault /usr/local/bin/vault

EXPOSE 8080

ENTRYPOINT ["vault"]


