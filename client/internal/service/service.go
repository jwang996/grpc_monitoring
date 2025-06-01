package service

import (
	monitoringpb "client/internal/pb/monitoring"
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ClientService struct {
	conn         *grpc.ClientConn
	client       monitoringpb.MonitoringServiceClient
	totalCalls   prometheus.Counter
	successCalls prometheus.Counter
	failureCalls prometheus.Counter
}

func NewClientService(serverAddr string, dialOpts ...grpc.DialOption) (*ClientService, error) {
	target := "dns:///" + serverAddr

	grpcConn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient(%q) returned error: %w", serverAddr, err)
	}

	client := monitoringpb.NewMonitoringServiceClient(grpcConn)

	total := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grpc_client_total_requests",
		Help: "Total number of gRPC requests sent by the client",
	})
	success := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grpc_client_success_requests",
		Help: "Number of successful gRPC requests",
	})
	failure := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "grpc_client_failed_requests",
		Help: "Number of failed gRPC requests",
	})

	prometheus.MustRegister(total, success, failure)

	return &ClientService{
		conn:         grpcConn,
		client:       client,
		totalCalls:   total,
		successCalls: success,
		failureCalls: failure,
	}, nil
}

func (cs *ClientService) Close() error {
	return cs.conn.Close()
}

func (cs *ClientService) SendPing(ctx context.Context) (string, error) {
	cs.totalCalls.Inc()

	req := &monitoringpb.MonitoringClientRequest{
		ClientRequest: &monitoringpb.Client{
			Message:     "ping",
			RequestDate: timestamppb.Now(),
		},
	}
	resp, err := cs.client.Monitoring(ctx, req)
	if err != nil {
		cs.failureCalls.Inc()
		return "", err
	}
	cs.successCalls.Inc()
	return resp.GetMessage(), nil
}

func (cs *ClientService) SendWrong(ctx context.Context) error {
	cs.totalCalls.Inc()

	req := &monitoringpb.MonitoringClientRequest{
		ClientRequest: &monitoringpb.Client{
			Message:     "wrong",
			RequestDate: timestamppb.Now(),
		},
	}
	_, err := cs.client.Monitoring(ctx, req)
	if err != nil {
		cs.failureCalls.Inc()
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			return nil
		}
		return err
	}

	cs.successCalls.Inc()
	return nil
}
