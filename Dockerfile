FROM golang:1.24.5-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o lovco server/main.go

FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget
    
WORKDIR /root/
COPY --from=builder /app/lovco .

# gRPC health probe (static)
RUN wget -qO /bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.26/grpc_health_probe-linux-amd64 && chmod +x /bin/grpc_health_probe

EXPOSE 50051

CMD ["./lovco"]