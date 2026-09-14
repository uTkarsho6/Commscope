package grpcserver

import (
	"context"
	"fmt"
	"testing"

	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"
)

func TestClientStreamingGRPC_SendBatchMessages(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("grpc-client-streaming")

	conn, closer := dialer(reg, tracker)
	defer closer()

	client := pb.NewCommScopeServiceClient(conn)

	stream, err := client.SendBatchMessages(context.Background())
	if err != nil {
		t.Fatalf("Failed to open Client Streaming RPC: %v", err)
	}

	// Stream 3 request messages to the server
	for i := 1; i <= 3; i++ {
		req := &pb.SendMessageRequest{
			ClientId: "test-client-stream",
			Text:     fmt.Sprintf("Batch Message #%d", i),
		}
		if err := stream.Send(req); err != nil {
			t.Fatalf("Failed to send chunk #%d: %v", i, err)
		}
	}

	// Close stream and receive the single summary response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv failed: %v", err)
	}

	if resp.GetStatus() != "success" {
		t.Errorf("Expected status 'success', got %s", resp.GetStatus())
	}
}
