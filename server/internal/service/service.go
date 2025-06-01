package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	monitoringpb "server/internal/pb/monitoring"
)

type Service struct {
	monitoringpb.UnimplementedMonitoringServiceServer
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Monitoring(
	ctx context.Context, // match grpc generated server interface
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
