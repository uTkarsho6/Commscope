package main

import (
	grpcserver "commscope/internal/grpc"
	"commscope/internal/handlers"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"commscope/pkg/pb"
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	reg := registry.NewConnectionRegistry()

	httpTracker := metrics.NewProtocolTracker("http")

	httpHandler := handlers.NewHTTPHandler(reg, httpTracker)

	log.Printf("Initialized Connection Registry (active connections: %d)", reg.Count())

	mux := http.NewServeMux()

	mux.HandleFunc(
		"/health",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

	server := &http.Server{ // pointer to http server
		Addr:    ":8080",
		Handler: mux,
	}

	// http route wiring
	mux.HandleFunc("/api/http/send", httpHandler.HandleSend)
	mux.HandleFunc("/api/http/stats", httpHandler.HandleStats)

	pollingTracker := metrics.NewProtocolTracker("polling")
	pollingStore := handlers.NewMessageStore()
	pollingHandler := handlers.NewPollingHandler(reg, pollingTracker, pollingStore)

	mux.HandleFunc("/api/polling/send", pollingHandler.HandleSend)
	mux.HandleFunc("/api/polling/messages", pollingHandler.HandleGetMessage)
	mux.HandleFunc("/api/polling/stats", pollingHandler.HandleStats)

	longPollTracker := metrics.NewProtocolTracker("longpolling")
	longPollStore := handlers.NewMessageStore()
	longPollBroadcaster := handlers.NewLongPollBroadcaster()
	longPollHandler := handlers.NewLongPollingHandler(reg, longPollTracker, longPollStore, longPollBroadcaster)

	mux.HandleFunc("/api/longpolling/send", longPollHandler.HandleSend)
	mux.HandleFunc("/api/longpolling/messages", longPollHandler.HandleGetMessages)
	mux.HandleFunc("/api/longpolling/stats", longPollHandler.HandleStats)

	sseTracker := metrics.NewProtocolTracker("sse")
	sseStore := handlers.NewMessageStore()
	sseBroadcaster := handlers.NewSSEBroadcaster()
	sseHandler := handlers.NewSSEHandler(reg, sseTracker, sseStore, sseBroadcaster)

	mux.HandleFunc("/api/sse/send", sseHandler.HandleSend)
	mux.HandleFunc("/api/sse/events", sseHandler.HandleEvents)
	mux.HandleFunc("/api/sse/stats", sseHandler.HandleStats)

	wsTracker := metrics.NewProtocolTracker("ws")
	wsHub := handlers.NewWSHub()
	go wsHub.Run() // start central hub event loop

	wsHandler := handlers.NewWSHandler(wsHub, reg, wsTracker)

	mux.HandleFunc("/api/ws/connect", wsHandler.HandleConnect)
	mux.HandleFunc("/api/ws/send", wsHandler.HandleSend)
	mux.HandleFunc("/api/ws/stats", wsHandler.HandleStats)

	// gRPC Server Setup:

	//tcp listener on 50051 port
	grpcListener, err := net.Listen("tcp", ":50051")

	//check if listener creation failed
	if err != nil {
		log.Fatalf("failed to listen on: %v", err)

	}

	grpcTracker := metrics.NewProtocolTracker("grpc-unary")

	// application lvl gRPC handler, application logic
	grpcSrv := grpcserver.NewServer(reg, grpcTracker)

	// actual server (gRPC framework)
	gServer := grpc.NewServer()

	pb.RegisterCommScopeServiceServer(gServer, grpcSrv) // connects your implementation to the gRPC framework.

	reflection.Register(gServer) // adding for development purpose only in production dont use this, allow tools to automatically know about the service provided by gRPC

	go func() {
		log.Printf("Starting gRPC server on %s", grpcListener.Addr().String())
		// The current goroutine is occupied waiting for/serving connections, so execution doesn't proceed past that function call.
		// This line is effectively a blocking call that would prevent the rest of the main function from executing if it were in the main goroutine.
		// However, since it is inside a goroutine, the main function can continue to execute concurrently.
		if err := gServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ServerListenAndServe error :%v", err)
		}
	}()

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Server is ready. Press ctrl+C to shut down")
	<-stop // block or wait main goroutine until any signal is sent on stop channel , here we are receiving the stop signal but not storing it in var

	log.Printf("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)

	}
	log.Printf("Server gracefully shutdown")
}
