FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server main.go

FROM alpine:3.19

RUN adduser -D -u 1000 appuser
WORKDIR /app
COPY --from=builder /app/server .
RUN chown -R appuser:appuser /app

USER appuser
EXPOSE 8080
CMD ["./server"]
