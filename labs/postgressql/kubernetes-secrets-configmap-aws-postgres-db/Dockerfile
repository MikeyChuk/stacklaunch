FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

FROM alpine:3.22

WORKDIR /app

RUN addgroup \
      --gid 10001 \
      stacklaunch \
    && adduser \
      --disabled-password \
      --no-create-home \
      --uid 10001 \
      --ingroup stacklaunch \
      stacklaunch

COPY --from=builder /app/api /app/api

USER appuser

EXPOSE 8080

CMD ["/app/api"]