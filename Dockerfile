FROM golang:1.22.4 AS builder

WORKDIR /app

COPY ./lineup-generation/v2 .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o exec

FROM alpine:latest AS certs

RUN apk --no-cache add ca-certificates

FROM scratch

WORKDIR /app

COPY --from=builder /app/exec /app/exec

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY ./lineup-generation/v2/static/schedule25-26.json /app/static/schedule25-26.json

CMD ["./exec"]