# Dockerfile in project root for the server
FROM golang:1.23-alpine3.19 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server /app/
COPY config.json /app/config.json
EXPOSE 8080
ENTRYPOINT ["/app/server", "--config", "/app/config.json"]
