package main

import (
	"flag"
	"fmt"
	"log/slog"
	"lovco/chat"
	"lovco/config"
	"lovco/leftover"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config.InitDB(logger)
	defer config.CloseDB(logger)

	leftoverServer := leftover.NewLeftoverServer(config.DB)
	chatServer := chat.NewChatServer(config.DB)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	reflection.Register(srv)

	leftover.RegisterLeftoverServiceServer(srv, leftoverServer)
	chat.RegisterChatServiceServer(srv, chatServer)

	logger.Info("Server starting on port", "port", *port)

	panic(srv.Serve(lis))
}
