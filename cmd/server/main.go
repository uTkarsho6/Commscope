package main

import (
	"commscope/internal/handlers"
	"commscope/internal/metrics"
	"commscope/internal/registry"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
