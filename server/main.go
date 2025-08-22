package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"lovco/chat"
	"lovco/config"
	"lovco/leftover"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// server/main.go (add below imports, above func main)
func newLoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		var peerAddr string
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			peerAddr = p.Addr.String()
		}
		logger.Info(
			"\ngrpc unary start",
			"\nmethod", info.FullMethod,
			"\npeer", peerAddr,
		)
		resp, err := handler(ctx, req)
		st, _ := status.FromError(err)
		logger.Info(
			"\ngrpc unary end",
			"\nmethod", info.FullMethod,
			"\ncode", st.Code().String(),
			"\nduration", time.Since(start),
			"\npeer", peerAddr,
			"\nerr", err,
		)
		return resp, err
	}
}

type loggingServerStream struct {
	grpc.ServerStream
	recv int
	sent int
}

func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.recv++
	}
	return err
}

func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.sent++
	}
	return err
}

func newLoggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		var peerAddr string
		if p, ok := peer.FromContext(ss.Context()); ok && p.Addr != nil {
			peerAddr = p.Addr.String()
		}
		wrapped := &loggingServerStream{ServerStream: ss}
		logger.Info(
			"grpc stream start",
			"method", info.FullMethod,
			"peer", peerAddr,
			"client_stream", info.IsClientStream,
			"server_stream", info.IsServerStream,
		)
		err := handler(srv, wrapped)
		st, _ := status.FromError(err)
		logger.Info(
			"grpc stream end",
			"method", info.FullMethod,
			"code", st.Code().String(),
			"duration", time.Since(start),
			"peer", peerAddr,
			"recv", wrapped.recv,
			"sent", wrapped.sent,
			"err", err,
		)
		return err
	}
}

func main() {
	port := flag.Int("port", 50051, "The server port")
	address := flag.String("address", "0.0.0.0", "The server address")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config.InitDB(logger)
	defer config.CloseDB(logger)

	leftoverServer := leftover.NewLeftoverServer(config.DB)
	chatServer := chat.NewChatServer(config.DB)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *address, *port))
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(newLoggingUnaryInterceptor(logger)),
		grpc.ChainStreamInterceptor(newLoggingStreamInterceptor(logger)),
	)
	reflection.Register(srv)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthServer)
	leftover.RegisterLeftoverServiceServer(srv, leftoverServer)
	chat.RegisterChatServiceServer(srv, chatServer)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Server starting on port", "port", *port)
		if err := srv.Serve(lis); err != nil {
			logger.Error("Failed to serve", "error", err)
		}
	}()

	<-c
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		healthServer.SetServingStatus("grpc.health.v1.Health", healthpb.HealthCheckResponse_NOT_SERVING)
		srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Server shutdown completed")
	case <-ctx.Done():
		logger.Warn("Server shutdown timed out, forcing stop")
		srv.Stop()
	}
}
