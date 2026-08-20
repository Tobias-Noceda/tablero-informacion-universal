FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY ../go.mod ../go.sum ./
RUN go mod download

COPY ../common ../api ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api ./api/main.go

FROM alpine:3.21

# Import CA certificates to allow outgoing HTTPS calls from your Go code
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

COPY --from=builder /app/api .

RUN adduser -D appuser
USER appuser

EXPOSE 31126

CMD ["./api"]
