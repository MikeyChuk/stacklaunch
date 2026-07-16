FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api .

FROM alpine:3.22

WORKDIR /app

RUN adduser -D -u 10001 appuser

COPY --from=builder /app/api /app/api

USER appuser

EXPOSE 8080

CMD ["/app/api"]