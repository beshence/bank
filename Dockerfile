FROM golang:1.26

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /usr/local/bin/vault .

EXPOSE 80

ENTRYPOINT ["vault"]


