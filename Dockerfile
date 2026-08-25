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

# Bundle every schedule under static/ so a season rollover is a change of SCHEDULE_FILE,
# not a Dockerfile edit. The server fails at startup if the file is missing.
COPY ./lineup-generation/v2/static/ /app/static/

ENV SCHEDULE_FILE=/app/static/schedule26-27.json

CMD ["./exec"]
