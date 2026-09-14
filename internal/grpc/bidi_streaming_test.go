package grpcserver

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"
	"context"
	"fmt"
	"io"
	"testing"
)

func TestBidiStreamingGRPC_ChatStream(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("grpc-bidi-streaming")

	conn, closer := dialer(reg, tracker)
	defer closer()

	client := pb.NewCommScopeServiceClient(conn)

	stream, err := client.ChatStream(context.Background())
	if err != nil {
		t.Fatalf("Failed to open ChatStream RPC: %v", err)
	}

	// 1. Send 3 request messages
	for i := 1; i <= 3; i++ {
		req := &pb.SendMessageRequest{
			ClientId: "test-bidi-client",
			Text:     fmt.Sprintf("Bidi Message #%d", i),
		}
		if err := stream.Send(req); err != nil {
			t.Fatalf("Failed to send frame #%d: %v", i, err)
		}

		// 2. Read concurrent real-time response
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed on frame #%d: %v", i, err)
		}

		if resp.GetStatus() != "success" {
			t.Errorf("Expected status 'success', got %s", resp.GetStatus())
		}
	}

	stream.CloseSend()
}
