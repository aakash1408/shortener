# Stage 1 - Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o shortener ./cmd/server/

# Stage 2 - Run
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/shortener .

EXPOSE 8080

CMD ["./shortener"]
