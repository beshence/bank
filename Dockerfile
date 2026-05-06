FROM golang:1.25.8 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /usr/local/bin/bank .

FROM alpine:3.23.3

WORKDIR /usr/local/bin

RUN apk add --no-cache curl \
    && addgroup -S bank \
    && adduser -S bank -G bank

COPY --from=builder /usr/local/bin/bank /usr/local/bin/bank

RUN chown bank:bank /usr/local/bin/bank && chmod 0755 /usr/local/bin/bank

USER bank:bank

EXPOSE 27462

ENTRYPOINT ["/usr/local/bin/bank"]


