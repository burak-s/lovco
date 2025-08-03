package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"xvco/config"
	"xvco/leftover"

	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config.InitDB(logger)
	defer config.CloseDB(logger)

	leftoverServer := leftover.NewLeftoverServer(config.DB)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()

	leftover.RegisterLeftoverServiceServer(srv, leftoverServer)
	logger.Info("Server starting on port", "port", *port)

	panic(srv.Serve(lis))
}
