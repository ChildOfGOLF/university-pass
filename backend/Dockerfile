FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN swag init -g cmd/api/main.go -o ./docs

RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/api .

EXPOSE 8080

CMD ["./api"]