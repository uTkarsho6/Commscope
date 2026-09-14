package grpcserver

import (
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"
	"context" //Controls request cancellation signals
	"io"      //Provides io.EOF (End of File), which gRPC streams return when the stream is closed by the server.
	"testing"
	"time"
)

func TestServerStreamingGRPC_StreamMessages(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("grpc-server-streaming")

	conn, closer := dialer(reg, tracker) // reuses the helper func from unary_test.go
	defer closer()

	client := pb.NewCommScopeServiceClient(conn)
	//Background returns a non-nil, empty Context. It is never canceled, has no values, and has no deadline. It is typically used by the main function, initialization, and tests, and as the top-level Context for incoming requests.
	// Set 3-second context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// open server streaming RPC
	stream, err := client.StreamMessages(ctx, &pb.GetStatsRequest{})
	if err != nil {
		t.Fatalf("Failed to open Server Streaming RPC: %v", err)
	}

	// Read incoming streamed messages in a loop
	receivedCount := 0

	for {
		resp, err := stream.Recv() // Recv receives the next response message from the server. The client may repeatedly call Recv to read messages from the response stream.
		if err == io.EOF {
			break // stream ended
		}
		if ctx.Err() != nil {
			break // context deadline expired
		}
		if err != nil {
			break
		}

		if resp.GetStatus() != "success" {
			t.Errorf("Expected status 'success', got %s", resp.GetStatus())
		}
		receivedCount++

		if receivedCount >= 2 {
			cancel()
			break
		}
	}

		if receivedCount < 2 {
		t.Errorf("Expected at least 2 stream messages, got %d", receivedCount)
	}
}
