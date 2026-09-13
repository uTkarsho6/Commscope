package grpcserver

import (
	"context"
	"net"
	"testing"

	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func dialer(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	srv := NewServer(reg, tracker)
	pb.RegisterCommScopeServiceServer(s, srv)

	go func() {
		if err := s.Serve(lis); err != nil {
			return
		}
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	closer := func() {
		lis.Close()
		s.Stop()
		conn.Close()
	}

	return conn, closer
}

func TestUnaryGRPC_SendMessageAndStats(t *testing.T) {
	reg := registry.NewConnectionRegistry()
	tracker := metrics.NewProtocolTracker("grpc-unary")

	conn, closer := dialer(reg, tracker)
	defer closer()

	client := pb.NewCommScopeServiceClient(conn)

	// Test SendMessage RPC
	resp, err := client.SendMessage(context.Background(), &pb.SendMessageRequest{
		ClientId: "test-grpc-client",
		Text:     "Hello gRPC Unary!",
	})

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if resp.GetStatus() != "success" {
		t.Errorf("Expected status 'success', got %s", resp.GetStatus())
	}

	// Test GetStats RPC
	statsResp, err := client.GetStats(context.Background(), &pb.GetStatsRequest{})
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if statsResp.GetMessageCount() != 1 {
		t.Errorf("Expected message count 1, got %d", statsResp.GetMessageCount())
	}
}
