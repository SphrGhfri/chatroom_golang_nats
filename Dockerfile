# ------------------------ Base Stage ------------------------
FROM golang:1.23 AS base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# ------------------------ Development Stage ------------------------
FROM base AS dev

RUN go install github.com/air-verse/air@v1.61.5
COPY . .

CMD ["air", "-c", ".air.toml"]

# ------------------------ Builder Stage ------------------------
FROM base AS builder

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# ------------------------ Production Stage ------------------------
FROM alpine:latest AS prod

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Tehran

WORKDIR /app
COPY --from=builder /app/server /app/
COPY config.json /app/config.json
EXPOSE 8080
ENTRYPOINT ["/app/server", "--config", "/app/config.json"]
