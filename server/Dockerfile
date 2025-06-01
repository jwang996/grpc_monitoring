FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server ./cmd/

FROM alpine:3.18
RUN apk add --no-cache ca-certificates

COPY --from=builder /server /usr/local/bin/server

RUN mkdir -p /etc/certs

USER nobody:nobody

ENTRYPOINT ["/usr/local/bin/server"]