FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /zhibo-server ./cmd/server

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /zhibo-server /usr/local/bin/zhibo-server

USER app
EXPOSE 8081

CMD ["zhibo-server"]
