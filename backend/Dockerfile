FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server ./main.go

FROM alpine:3.21

# Import CA certificates to allow outgoing HTTPS calls from your Go code
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /app/server .

RUN adduser -D appuser
USER appuser

EXPOSE 31126

CMD ["./server"]
