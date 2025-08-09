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
RUN apk add --no-cache ca-certificates
    
WORKDIR /root/
COPY --from=builder /app/lovco .

EXPOSE 50051

CMD ["./lovco"]