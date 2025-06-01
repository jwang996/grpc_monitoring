package main

import (
	"client/internal/security"
	"context"
	"errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"client/internal/config"
	"client/internal/service"
)

func main() {
	cfg := config.LoadConfig()
	ctx := context.Background()

	tickerCtx, cancelTickers := context.WithCancel(context.Background())
	defer cancelTickers()

	tp, err := setupOpenTelemetry(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to set up tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	creds, err := security.LoadClientTLSCredentials(cfg)
	if err != nil {
		log.Fatalf("cannot load client TLS credentials: %v", err)
	}

	// Dial options:
	// - WithStatsHandler(otelgrpc.NewClientHandler()) → for tracing outgoing RPCs
	// - grpcprometheus interceptors → for Prometheus metrics
	otelClientHandler := otelgrpc.NewClientHandler()
	dialOpts := []grpc.DialOption{
		// mTLS
		grpc.WithTransportCredentials(creds),
		// OpenTelemetry interceptor
		grpc.WithStatsHandler(otelClientHandler),
		// Prometheus interceptors
		grpc.WithUnaryInterceptor(grpcprometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpcprometheus.StreamClientInterceptor),
	}

	clientSvc, err := service.NewClientService(cfg.GRPCServerAddress, dialOpts...)
	if err != nil {
		log.Fatalf("failed to create ClientService: %v", err)
	}
	defer func() {
		if err := clientSvc.Close(); err != nil {
			log.Printf("error closing gRPC client connection: %v", err)
		}
	}()

	metricAddr := ":" + cfg.MetricsPort
	httpSrv := &http.Server{
		Addr:    metricAddr,
		Handler: promhttp.Handler(),
	}
	go func() {
		log.Printf("[METRICS] listening on %s", metricAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[METRICS] ListenAndServe error: %v", err)
		}
	}()

	// Goroutine: send "ping" every 15 seconds
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-tickerCtx.Done():
				return
			case <-ticker.C:
				msg, err := clientSvc.SendPing(context.Background())
				if err != nil {
					log.Printf("[PING] error sending ping: %v", err)
				} else {
					log.Printf("[PING] received response: %s", msg)
				}
			}
		}
	}()

	// 9) Goroutine: send "wrong" every 2 minutes
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-tickerCtx.Done():
				return
			case <-ticker.C:
				err := clientSvc.SendWrong(context.Background())
				if err != nil {
					log.Printf("[WRONG] unexpected error: %v", err)
				} else {
					log.Printf("[WRONG] server correctly returned InvalidArgument")
				}
			}
		}
	}()

	<-stop
	log.Println("[MAIN] shutdown signal received, stopping all goroutines...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[METRICS] HTTP shutdown error: %v", err)
	} else {
		log.Println("[METRICS] HTTP server stopped")
	}

	cancelTickers()
	log.Println("[MAIN] all goroutines signaled to stop; exiting")
}
