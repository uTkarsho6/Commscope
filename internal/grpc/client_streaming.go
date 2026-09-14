package grpcserver

import (
	"commscope/internal/registry"
	"commscope/pkg/pb"
	"fmt"
	"io"
	"time"
)

// The client streams multiple request messages to the server.
// The server reads until io.EOF and returns 1 summary response via stream.SendAndClose.

func (s *Server) SendBatchMessages(stream pb.CommScopeService_SendBatchMessagesServer) error {
	clientID := "grpc-client-stream-client"

	s.Registry.Add(&registry.Client{
		ID:          clientID,
		Protocol:    "grpcs-client-streaming",
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
	})
	count := 0
	start := time.Now()
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// client finished the stream just send one reponse summary and close the stream
			return stream.SendAndClose(&pb.SendMessageResponse{
				Status:    "success",
				MessageId: fmt.Sprintf("batch-summary-%d-msgs", count),
				CreatedAt: time.Now().Format(time.RFC3339Nano),
			})

		}

		if err != nil {
			return err
		}

		_ = req // Received message payload from client chunk
		count++

		s.Registry.RecordMessage(clientID)

		duration := time.Since(start)
		s.Tracker.Record(duration)

		start = time.Now()

	}

}
