package main

import (
	"context"
	"errors"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"server/internal/security"
	"syscall"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"server/internal/config"
	monitoringpb "server/internal/pb/monitoring"
	"server/internal/service"
)

func main() {
	cfg := config.LoadConfig()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	creds, err := security.LoadTLSCredentials(cfg)
	if err != nil {
		log.Fatalf("cannot load TLS credentials: %v", err)
	}

	metricAddr := ":" + cfg.MetricsPort
	httpSrv := &http.Server{
		Addr:    metricAddr,
		Handler: promhttp.Handler(), // exposes /metrics
	}

	go func() {
		log.Printf("[METRICS] listening on %s", metricAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Fatalf("[METRICS] failed to start metrics endpoint: %v", err)
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpcprometheus.UnaryServerInterceptor),
	)

	svc := service.NewService()
	monitoringpb.RegisterMonitoringServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	grpcprometheus.Register(grpcServer)
	grpcprometheus.EnableHandlingTimeHistogram()

	grpcAddr := ":" + cfg.GRPCPort
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	go func() {
		log.Printf("[GRPC] gRPC server listening on %s", grpcAddr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("[GRPC] gRPC Serve stopped: %v", err)
		}
	}()

	<-stop
	log.Println("[MAIN] shutdown signal received, stopping servers...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[METRICS] HTTP shutdown error: %v", err)
	} else {
		log.Println("[METRICS] HTTP server stopped")
	}

	grpcServer.GracefulStop()
	log.Println("[GRPC] gRPC server stopped gracefully")

	log.Println("[MAIN] All servers have shut down. Exiting.")
}
