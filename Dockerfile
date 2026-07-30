FROM golang:1.26 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -v -o /usr/local/bin/bank .

FROM alpine:3.23.3

WORKDIR /usr/local/bin

COPY --from=builder /usr/local/bin/bank /usr/local/bin/bank

ARG BANK_PORT=27462

EXPOSE ${BANK_PORT}

ENTRYPOINT ["/usr/local/bin/bank"]


