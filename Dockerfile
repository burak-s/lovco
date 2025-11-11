FROM golang:1.25.3-alpine3.22 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o lovco -ldflags="-s -w" server/cmd/main.go

RUN wget -qO /grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.26/grpc_health_probe-linux-amd64 && chmod +x /grpc_health_probe

FROM alpine:3.22
RUN apk add --no-cache ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    
WORKDIR /home/appuser/
COPY --from=builder /app/lovco .
COPY --from=builder /grpc_health_probe /bin/grpc_health_probe

RUN chown -R appuser:appgroup /home/appuser
USER appuser

EXPOSE 50051

CMD ["./lovco"]