package grpcserver

import (
	"context"
	"fmt"
	"time"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"

)

// Server implements the CommScopeServiceServer gRPC interface.
// service handler struct
type Server struct {
	pb.UnimplementedCommScopeServiceServer // embedding to inherit default empty implementation by protobuf ( prevents compilation error) maintain forward compatibility if any new rpc is added in future
	Registry *registry.ConnectionRegistry
	Tracker *metrics.ProtocolTracker
}

//NewServer initializes a new gRPC Server instance
func NewServer(reg *registry.ConnectionRegistry, tracker *metrics.ProtocolTracker) *Server {
	return &Server{
		Registry: reg,
		Tracker: tracker,
	}
}

// SendMessage handles Unary gRPC mesage sending.

func(s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	start := time.Now()

	clientID := req.GetClientId()
	if clientID == ""{
     clientID = "grpc-unary-client"
	}

	s.Registry.Add(&registry.Client{
		ID: clientID,
		Protocol: "grpc-unary",
		ConnectedAt: time.Now(),
		LastSeen: time.Now(),

	})

	s.Registry.RecordMessage(clientID)
	duration := time.Since(start)
	s.Tracker.Record(duration)
	
	return &pb.SendMessageResponse{
		Status: "success",
		MessageId: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		CreatedAt: time.Now().Format(time.RFC3339Nano),
	},nil

}

// GetStats returns telemetry metrics over Unary gRPC.

func(s *Server) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.ProtocolStatsResponse, error){
      activeCount := s.Registry.Count()

	  var totalMessages int64
	  for _, client := range s.Registry.List(){
		if client.Protocol == "grpc-unary" {
			totalMessages += client.MessageCount
		}
	  }
	  snapshot := s.Tracker.Snapshot(activeCount, totalMessages)

	  return &pb.ProtocolStatsResponse {
		Protocol: snapshot.Protocol,
		ActiveConnections: int32(snapshot.ActiveConnections),
		MessageCount: snapshot.MessageCount,
		P50LatencyMs: snapshot.P50LatencyMs,
        P99LatencyMs: snapshot.P99LatencyMs,
		Throughput: snapshot.Throughput,
	  }, nil


}


