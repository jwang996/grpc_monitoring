package service

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	monitoringpb "server/internal/pb/monitoring"
)

func TestMonitoring(t *testing.T) {
	svc := NewService()

	baseTime := time.Date(2025, 5, 31, 14, 23, 0, 0, time.UTC)
	tsProto := timestamppb.New(baseTime)

	tests := []struct {
		name        string
		message     string
		timestamp   *timestamppb.Timestamp
		wantMessage string
		wantCode    codes.Code
	}{
		{
			name:        "valid ping",
			message:     "ping",
			timestamp:   tsProto,
			wantMessage: "ping on 2025-05-31 14:23:00, response: pong",
			wantCode:    codes.OK,
		},
		{
			name:      "nil client_request",
			message:   "",
			timestamp: tsProto,
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "nil timestamp",
			message:   "ping",
			timestamp: nil,
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "wrong message",
			message:   "hello",
			timestamp: tsProto,
			wantCode:  codes.InvalidArgument,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req *monitoringpb.MonitoringClientRequest
			switch tc.name {
			case "nil client_request":
				req = &monitoringpb.MonitoringClientRequest{
					ClientRequest: nil,
				}
			default:
				req = &monitoringpb.MonitoringClientRequest{
					ClientRequest: &monitoringpb.Client{
						Message:     tc.message,
						RequestDate: tc.timestamp,
					},
				}
			}

			resp, err := svc.Monitoring(context.Background(), req)
			if tc.wantCode != codes.OK {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tc.wantCode)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected a gRPC status error, got: %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("expected error code %v, got %v (%v)", tc.wantCode, st.Code(), st.Message())
				}
				return
			}

			// Expect no error
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp == nil {
				t.Fatalf("expected non-nil response")
			}
			if resp.GetMessage() != tc.wantMessage {
				t.Errorf("unexpected message: got %q, want %q", resp.GetMessage(), tc.wantMessage)
			}
		})
	}
}
