# Technical Requirements Document (TRD)
## CommScope — Interactive Real-Time Communication Playground

**Version:** 1.0
**Companion to:** CommScope-PRD.md
**Status:** Draft — for Antigravity project context

---

## 1. Architecture Overview

```
                       [CommScope Architecture]
                                  │
    ┌─────────────────────────────┼─────────────────────────────┐
    ▼                             ▼                             ▼
[React + TS Dashboard]      [Go Central Hub]           [Telemetry Engine]
 • Protocol panels           • Protocol handlers         • Latency / RTT calc
 • Visualization layer       • Connection registry       • Throughput calc
 • Live log streams          • Event broadcast engine     • Metrics aggregation
 • Zustand state store       • gRPC server (HTTP/2)       • Exposed via /stats
                             • WebRTC signaling (via WS)
```

- **Frontend:** React + TypeScript SPA, communicates with backend via plain HTTP, EventSource (SSE), native WebSocket API, and Connect-Web (gRPC-over-HTTP bridge).
- **Backend:** Single Go binary exposing all protocol endpoints from one process (single instance, no microservices — see Non-Goals in PRD).
- **Reverse proxy (deployment only):** Nginx or Caddy terminating TLS and routing to the Go binary, with special handling for WebSocket `Upgrade` and HTTP/2 (gRPC) passthrough.

## 2. Technology Stack

| Layer | Choice | Notes |
|---|---|---|
| Backend language | Go (current stable version) | Single binary, strong concurrency primitives |
| HTTP routing | Standard library `net/http` (`http.ServeMux`) | No framework needed for this scope |
| WebSockets | `gorilla/websocket` or `nhooyr.io/websocket` | Pick one, document choice in code comments |
| gRPC | `google.golang.org/grpc` + `protoc`/`buf` for codegen | Standard official tooling |
| gRPC-Web bridge | `connectrpc.com` (Connect-Go/Connect-Web) | Verify still current recommendation at build time |
| WebRTC (Go side, signaling only) | Native `net/http` + WebSocket for signaling; browser handles actual `RTCPeerConnection` | No Go-side media/data termination needed since DataChannel is browser-to-browser |
| Frontend framework | React + TypeScript | Vite as build tool |
| Styling | Tailwind CSS | |
| Animation | Framer Motion | Packet movement animation |
| Charts | Recharts | Latency/throughput line charts |
| Node/topology diagrams | react-flow | Client↔server / peer↔peer visualization |
| State management | Zustand | Central store for per-protocol status/metrics |
| Containerization | Docker (multi-stage builds), docker-compose | One container for Go backend, one for React static build (served via Nginx) |
| Reverse proxy / TLS | Nginx or Caddy | Caddy preferred for simpler auto-HTTPS in a solo/demo context |
| Load testing (dev-time only) | k6 (HTTP/WS), ghz (gRPC) | Used to validate live metrics accuracy, not shipped in production |

## 3. Backend Design

### 3.1 Project structure
```
/cmd/server/main.go        — entrypoint, wiring, graceful shutdown
/internal/handlers/        — one file/package per protocol (http, polling, sse, ws, grpc, webrtc)
/internal/registry/        — thread-safe ConnectionRegistry (shared across protocols)
/internal/metrics/         — latency/throughput/duration calculators, shared struct format
/internal/middleware/      — logging, CORS
/proto/                    — .proto definitions + generated code
/web/                      — React + TS frontend app
/deploy/                   — Dockerfiles, docker-compose.yml, nginx/caddy config
```

### 3.2 Shared core components (built once, reused across all protocols)
- **ConnectionRegistry:** `map[string]*Client` guarded by `sync.RWMutex`. Tracks per-connection: ID, protocol type, connected-at timestamp, message count, last-seen timestamp. Exposes `Add`, `Remove`, `List`, `Count`.
- **Metrics struct (per protocol):** `{Latency (rolling p50/p99), ThroughputMsgPerSec, MessageCount, ConnectionDuration, ActiveConnections}`. Computed centrally so every protocol reports metrics in the same shape to the frontend.
- **Graceful shutdown:** `server.Shutdown(ctx)` on SIGINT/SIGTERM, closing all registry-tracked connections cleanly.
- **Logging middleware:** structured request logs (method, path, duration, status).

### 3.3 Protocol-specific technical requirements

**HTTP Request-Response:** Standard `net/http` handler; log start/end timestamps for latency measurement.

**Short Polling:** `GET /poll/messages?since=<timestamp>` — in-memory message store, returns anything newer than `since`.

**Long Polling:** `GET /poll/wait` — handler blocks using a channel + `select` with timeout (e.g. 30s); returns immediately if new data arrives, or empty response on timeout (client re-polls).

**SSE:** `GET /sse/stream` — sets `Content-Type: text/event-stream`, writes events in a loop, calls `http.Flusher.Flush()` after each write; supports `Last-Event-ID` header for basic reconnection.

**WebSockets:** Upgrade handshake via chosen library; per-connection read pump + write pump goroutines; ping/pong heartbeat on an interval (e.g. 30s); broadcast via registry iteration or a pub/sub channel.

**gRPC (all 4 modes):** Single `.proto` file defining:
- `Echo(EchoRequest) returns (EchoResponse)` — Unary
- `Subscribe(SubscribeRequest) returns (stream Update)` — Server Streaming
- `Upload(stream Chunk) returns (UploadSummary)` — Client Streaming
- `Chat(stream ChatMessage) returns (stream ChatMessage)` — Bidirectional Streaming
Served over HTTP/2, bridged to the browser via Connect-Web.

**WebRTC:** Go WebSocket hub (reused from the WebSockets phase) relays SDP offer/answer and ICE candidates between two browser clients. No media/TURN server required for v1 (assume same-network/STUN-only is acceptable for demo purposes — flag TURN as a stretch goal if NAT traversal fails in testing).

## 4. Frontend Design

### 4.1 Structure
```
/web/src/
  hooks/            — useSSE, useWebSocket, useGrpcStream, useWebRTC
  store/            — Zustand store (per-protocol status/metrics)
  components/
    ProtocolPanel/  — reusable panel: node diagram + packet animation + chart + logs
    BattleMode/     — comparison view
    Dashboard/      — overview screen
  lib/              — protocol-specific client logic (e.g., gRPC client setup)
```

### 4.2 Key technical requirements
- All connection hooks must clean up properly on component unmount (close sockets/streams, cancel subscriptions) to avoid leaks when switching between panels.
- `<ProtocolPanel>` must be a single reusable component parameterized per protocol (props: protocol name, connection hook, node diagram variant [client-server vs peer-peer]) — not duplicated 7 times.
- Zustand store shape must be consistent across protocols so `<ProtocolPanel>` and `<BattleMode>` can consume any protocol's data identically:
  ```ts
  type ProtocolState = {
    status: 'idle' | 'connecting' | 'open' | 'closing' | 'closed';
    latency: number[];       // rolling window
    msgCount: number;
    throughput: number;
    connectionDuration: number;
    activeConnections: number;
    logs: LogEntry[];
  }
  ```
- Battle Mode must be able to run 2+ protocol hooks concurrently and read from the shared store without interference.

## 5. Non-Functional Requirements

- **Concurrency safety:** All shared backend state must be protected (mutex or channel-based); validated with `go test -race`.
- **Resource limits:** Backend should cap max concurrent connections per protocol (configurable constant) to prevent demo misuse from exhausting resources.
- **Observability:** Structured logs on the backend, correlated where possible with a connection/request ID.
- **Performance targets (demo-scale, not production-scale):** Should comfortably handle at least 100 concurrent simulated clients per protocol without degraded UI responsiveness (validated via k6/ghz load tests).
- **Deployability:** Entire stack must start via a single `docker-compose up` command in a fresh environment.

## 6. Testing Requirements

- Unit tests for `ConnectionRegistry` under concurrent load (`go test -race`).
- Manual verification per protocol via an independent tool (curl for HTTP/polling/SSE, a WS test client for WebSockets, grpcurl for gRPC, browser devtools console for WebRTC) before trusting the UI's display of it.
- Load testing (k6 for HTTP/WS, ghz for gRPC) at 10/100/1000 simulated clients to validate that displayed metrics match independently measured ones.

## 7. Deployment Requirements

- Multi-stage Dockerfile for the Go backend (build stage + minimal runtime image).
- Separate Dockerfile for the React build, served as static files via Nginx (or served directly by Caddy).
- docker-compose.yml wiring both containers plus the reverse proxy.
- Reverse proxy configuration must correctly pass through: WebSocket `Upgrade`/`Connection` headers, HTTP/2 for gRPC, and SSE (no buffering that would break the stream).
- TLS via Let's Encrypt (Caddy can automate this) for the public deployment.
- Target public host: Fly.io, Railway, or a low-cost VPS (decide during Phase 14, per PRD open questions).

## 8. Risks & Technical Mitigations

| Risk | Mitigation |
|---|---|
| WebRTC NAT traversal failing on public deployment | Test on same network first; document as known limitation if TURN server isn't added |
| gRPC-Web bridge tooling changing (Connect-Web is actively evolving) | Verify current docs at [connectrpc.com](https://connectrpc.com/) before implementing that phase |
| Shared mutable state causing race conditions as protocols are added incrementally | Every new protocol handler must route through the same tested `ConnectionRegistry`, not introduce a new ad-hoc shared map |
| Reverse proxy misconfiguration silently breaking WebSockets/gRPC only in production (works on localhost) | Explicitly test WS + gRPC through the proxy locally via docker-compose before deploying publicly |

---

*This TRD defines HOW CommScope is built. See the companion PRD for WHAT is being built and WHY.*
