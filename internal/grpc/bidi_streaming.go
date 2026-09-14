package grpcserver

import(
	"fmt"
	"io"
	"time"
	"commscope/pkg/pb"
	"commscope/internal/registry"
)

// Both client and server concurrently stream messages over HTTP/2.

func (s *Server) ChatStream(stream pb.CommScopeService_ChatStreamServer) error {
       clientID  := "grpc-bidi-stream-client"

	   s.Registry.Add(&registry.Client{
		ID: clientID,
		Protocol: "grpc-bidi-streaming",
		ConnectedAt: time.Now(),
		LastSeen: time.Now(),
	   })

	   count := 0

	   for{
         // Read incoming request frame from client stream

		 req, err := stream.Recv()
		 if err == io.EOF {
			// client closed write stream
			return nil
		 }
		 if err != nil {
			return err
		 }

         count++
		 start := time.Now()
		 s.Registry.RecordMessage(clientID)
		 // Send real-time response frame over server stream

		 resp := &pb.SendMessageResponse{
			Status: "success",
			MessageId: fmt.Sprintf("bidi-msg-%d", count),
			CreatedAt: time.Now().Format(time.RFC3339Nano),
		 }

		 if err := stream.Send(resp); err != nil {
			return err
		 }

		 duration := time.Since(start)
		 s.Tracker.Record(duration) 
		 _=req // to avoid std go compiler error of defining and not using variable

	   }
}