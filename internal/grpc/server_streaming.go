package grpcserver

import (
	"fmt"
	"time"

	"commscope/internal/registry"
	"commscope/pkg/pb"
)

// StreamMessages implements gRPC Server Streaming RPC.
// The client sends 1 request, and the server streams multiple messages over HTTP/2.
func (s *Server) StreamMessages(req *pb.GetStatsRequest, stream pb.CommScopeService_StreamMessagesServer) error {
	clientID := "grpc-server-stream-client"

	s.Registry.Add(&registry.Client{
		ID:          clientID,
		Protocol:    "grpc-server-streaming",
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
	})

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-stream.Context().Done():
			// Client disconnected or stram context cancelled
			return nil

		case t := <-ticker.C:
			count++
			start := time.Now()
			resp := &pb.SendMessageResponse{
				Status:    "success",
				MessageId: fmt.Sprintf("stream-msg-%d", count),
				CreatedAt: t.Format(time.RFC3339Nano),
			}

			// Send protobuf message over the gRPC stream
			if err := stream.Send(resp); err != nil {
				return err
			}
			duration := time.Since(start)
			s.Tracker.Record(duration)
			s.Registry.RecordMessage(clientID)

		}
	}

}
