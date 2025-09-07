FROM golang:1.25.1 AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o order .

FROM alpine:3.19

WORKDIR /root/
COPY --from=builder /app/order .

CMD ["./order"]
