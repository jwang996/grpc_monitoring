FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /client ./cmd/

FROM alpine:3.18
RUN apk add --no-cache ca-certificates

COPY --from=builder /client /usr/local/bin/client

RUN mkdir -p /etc/certs

USER nobody:nobody
ENTRYPOINT ["/usr/local/bin/client"]