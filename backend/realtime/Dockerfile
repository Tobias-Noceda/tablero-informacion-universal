FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY ../go.mod ../go.sum ./
RUN go mod download

COPY ../common ../realtime ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/realtime ./realtime/main.go

FROM alpine:3.21

# Import CA certificates to allow outgoing HTTPS calls from your Go code
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

COPY --from=builder /app/realtime .

RUN adduser -D appuser
USER appuser

EXPOSE 62113

CMD ["./realtime"]
