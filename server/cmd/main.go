package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"lovco/server/chat"
	"lovco/server/config"
	"lovco/server/leftover"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	port    = flag.Int("port", 50051, "The server port")
	address = flag.String("address", "0.0.0.0", "The server address")

	sleep  = flag.Duration("sleep", 10*time.Minute, "The sleep time in minutes")
	system = ""
)

type server struct {
	grpc.ServerStream
	recv int
	sent int
}

func (s *server) RecvMsg(m any) error {
	slog.Info("Receive a message", "type", fmt.Sprintf("%T", m), "time", time.Now().Format(time.RFC3339))
	s.recv++
	return s.ServerStream.RecvMsg(m)
}

func (s *server) SendMsg(m any) error {
	slog.Info("Send a message", "type", fmt.Sprintf("%T", m), "time", time.Now().Format(time.RFC3339))
	s.sent++
	return s.ServerStream.SendMsg(m)
}

func loggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		var peerAddr string
		if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
			peerAddr = p.Addr.String()
		}
		logger.Info("grpc unary start", "method", info.FullMethod, "peer", peerAddr)

		resp, err := handler(ctx, req)
		st, _ := status.FromError(err)

		logger.Info("grpc unary end", "method", info.FullMethod, "code", st.Code().String(), "duration", time.Since(start), "peer", peerAddr, "err", err)
		return resp, err
	}
}

func loggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		var peerAddr string
		if p, ok := peer.FromContext(ss.Context()); ok && p.Addr != nil {
			peerAddr = p.Addr.String()
		}
		wrapped := &server{ServerStream: ss}
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
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config.InitDB(logger)
	defer config.DB.Close()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *address, *port))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(loggingUnaryInterceptor(logger)),
		grpc.ChainStreamInterceptor(loggingStreamInterceptor(logger)),
	)

	reflection.Register(srv)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(srv, healthcheck)

	leftoverServer := leftover.NewLeftoverServer(config.DB)
	leftover.RegisterLeftoverServiceServer(srv, leftoverServer)

	chatServer := chat.NewChatServer(config.DB)
	chat.RegisterChatServiceServer(srv, chatServer)

	go func() {
		next := healthpb.HealthCheckResponse_SERVING
		for {
			healthcheck.SetServingStatus(system, next)
			healthcheck.SetServingStatus("", next)

			if next == healthpb.HealthCheckResponse_SERVING {
				next = healthpb.HealthCheckResponse_NOT_SERVING
			} else {
				next = healthpb.HealthCheckResponse_SERVING
			}

			time.Sleep(*sleep)
		}
	}()

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	srv.GracefulStop()
	slog.Info("Server gracefully stopped")
}
