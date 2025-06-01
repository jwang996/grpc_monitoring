package service

import (
	monitoringpb "client/internal/pb/monitoring"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"net"
	"testing"
)

type testServer struct {
	monitoringpb.UnimplementedMonitoringServiceServer
}

func (s *testServer) Monitoring(
	ctx context.Context,
	req *monitoringpb.MonitoringClientRequest,
) (*monitoringpb.MonitoringServerResponse, error) {
	clientReq := req.GetClientRequest()
	if clientReq == nil {
		return nil, status.Errorf(codes.InvalidArgument, "client_request must not be nil")
	}

	msg := clientReq.GetMessage()
	tsProto := clientReq.GetRequestDate()
	if tsProto == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request_date must not be nil")
	}

	if msg != "ping" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid message: %q (expected \"ping\")", msg)
	}

	t := tsProto.AsTime().UTC()
	formatted := t.Format("2006-01-02 15:04:05")
	responseText := fmt.Sprintf("%s on %s, response: pong", msg, formatted)

	return &monitoringpb.MonitoringServerResponse{
		Message: responseText,
	}, nil
}

func startTestGRPCServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	monitoringpb.RegisterMonitoringServiceServer(server, &testServer{})

	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() {
		server.GracefulStop()
		lis.Close()
	}
}

func TestClientService_SendPingAndSendWrong(t *testing.T) {
	addr, cleanup := startTestGRPCServer(t)
	defer cleanup()

	var registered []prometheus.Collector

	unregisterAll := func() {
		for _, c := range registered {
			prometheus.Unregister(c)
		}
	}

	defer unregisterAll()

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(),
	}

	clientSvc, err := NewClientService(addr, dialOpts...)
	if err != nil {
		t.Fatalf("NewClientService(%q) error: %v", addr, err)
	}
	defer func() {
		if cerr := clientSvc.Close(); cerr != nil {
			t.Errorf("error closing client connection: %v", cerr)
		}
	}()

	registered = append(registered, clientSvc.totalCalls, clientSvc.successCalls, clientSvc.failureCalls)

	t.Run("SendPing_Success", func(t *testing.T) {
		msg, err := clientSvc.SendPing(context.Background())
		if err != nil {
			t.Fatalf("SendPing returned error: %v", err)
		}
		const prefix = "ping on "
		if len(msg) < len(prefix) || msg[:len(prefix)] != prefix {
			t.Errorf("unexpected response %q; must start with %q", msg, prefix)
		}
		if !endsWith(msg, ", response: pong") {
			t.Errorf("unexpected response %q; must end with %q", msg, ", response: pong")
		}
	})

	t.Run("SendWrong_InvalidArgument", func(t *testing.T) {
		if err := clientSvc.SendWrong(context.Background()); err != nil {
			t.Fatalf("SendWrong returned unexpected error: %v", err)
		}
	})
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}

func TestClientService_BadAddress(t *testing.T) {
	badAddr := "127.0.0.1:65535"

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	clientSvc, err := NewClientService(badAddr, dialOpts...)
	if err != nil {
		t.Fatalf("NewClientService(%q) returned unexpected error: %v", badAddr, err)
	}
	defer clientSvc.Close()

	_, pingErr := clientSvc.SendPing(context.Background())
	if pingErr == nil {
		t.Fatalf("SendPing to bad address %q succeeded unexpectedly", badAddr)
	}
	errMsg := pingErr.Error()
	if !contains(errMsg, "connection") && !contains(errMsg, "refused") && !contains(errMsg, "connect") {
		t.Errorf("unexpected error for SendPing to bad address: %v", pingErr)
	}
}

func TestClientService_InvalidArgs(t *testing.T) {
	addr, cleanup := startTestGRPCServer(t)
	defer cleanup()

	_, err := NewClientService(addr)
	if err == nil {
		t.Fatal("expected error when no dial options provided, got nil")
	}
	if !contains(err.Error(), "TransportCredentials") {
		t.Errorf("expected TransportCredentials‐related error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
