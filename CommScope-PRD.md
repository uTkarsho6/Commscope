# Product Requirements Document (PRD)
## CommScope — Interactive Real-Time Communication Playground

**Version:** 1.0
**Owner:** [Your name]
**Status:** Draft — for Antigravity project context

---

## 1. Purpose & Vision

CommScope is an educational, portfolio-grade web application that lets a user visually explore, run, and compare real (not simulated) implementations of every major client-server and peer-to-peer communication protocol used in modern backend systems. The product's core value is **making invisible network behavior visible** — showing, in real time, how HTTP, Polling, Long Polling, SSE, WebSockets, gRPC (all 4 modes), and WebRTC actually move data, so the difference between them stops being theoretical and becomes something you can watch happen.

## 2. Problem Statement

Engineers learning backend/networking concepts usually encounter these protocols as disconnected theory: a diagram in a blog post, a paragraph in docs, or a one-off toy example. There's no single tool where you can:
- Trigger the *same kind* of message across 7 different transport mechanisms,
- Watch the packet/message actually move between client and server (or peer and peer),
- See real, live-measured latency/throughput/connection-state differences side by side,
- Understand *why* one protocol is chosen over another for a given real-world use case (chat vs. live sports scores vs. video calls).

CommScope solves this by being a real, running system — not slides.

## 3. Target Users

- **Primary:** The builder themself — a backend engineer preparing for interviews / deepening systems knowledge, using this as a hands-on learning project.
- **Secondary:** Anyone reviewing the builder's portfolio (recruiters, interviewers, other engineers) who wants to quickly assess practical understanding of networking/backend concepts.
- **Tertiary:** Other learners who discover the deployed tool and use it to understand protocol differences visually.

## 4. Goals

- G1: Provide real, working backend implementations of all 7 protocol categories (10 total modes counting gRPC's 4).
- G2: Visually animate the actual movement of requests/responses/messages per protocol.
- G3: Display live, accurate metrics per protocol: latency, throughput, message count, connection duration, active connections.
- G4: Allow direct side-by-side comparison of 2+ protocols under identical conditions ("Battle Mode").
- G5: Be deployable and usable publicly, not just on localhost.

## 5. Non-Goals (explicitly out of scope for v1)

- Authentication / user accounts / multi-tenant usage.
- Persisting historical data beyond a session (no database requirement for v1 — in-memory is acceptable).
- Mobile-native apps (web-responsive is sufficient, not a dedicated mobile app).
- Video/audio media streaming over WebRTC (only DataChannel/text, to keep it comparable to other protocols).
- Horizontal scaling / multi-instance backend (single-instance is fine for a demo tool).

## 6. Core Features (Functional Requirements)

### 6.1 Protocol Implementations
| # | Protocol | Requirement |
|---|---|---|
| 1 | HTTP Request-Response | Standard stateless request/response, real handler, real timing |
| 2 | Short Polling | Client repeatedly requests "what's new since X" |
| 3 | Long Polling | Server holds request open until new data or timeout |
| 4 | SSE | Server pushes events over a persistent one-way connection |
| 5 | WebSockets | Full-duplex persistent connection, broadcast capability |
| 6a | gRPC Unary | Single request, single response over HTTP/2 |
| 6b | gRPC Server Streaming | One request, multiple streamed responses |
| 6c | gRPC Client Streaming | Multiple streamed requests, one response |
| 6d | gRPC Bidirectional Streaming | Both sides stream independently |
| 7 | WebRTC DataChannel | Peer-to-peer connection established via signaling, direct data exchange |

### 6.2 Per-Protocol Dashboard Panel
Each protocol must have a UI panel showing:
- Animated visual representation of message/packet movement (client→server, server→client, or peer→peer)
- Current connection state (e.g., CONNECTING / OPEN / CLOSING / CLOSED, or equivalent per protocol)
- Request/response or message timeline (chronological log of events)
- Live server-side log stream relevant to that protocol
- Live client-side log stream
- Live metrics: latency (current + rolling), message count, throughput (msg/sec or bytes/sec), connection duration
- Count of currently active connections for that protocol

### 6.3 Comparison / "Battle Mode"
- User can select 2 or more protocols to run simultaneously under an equivalent workload (e.g., same message size/frequency).
- Results displayed as a side-by-side live chart (latency and/or throughput comparison).

### 6.4 Global Dashboard
- Overview screen listing all protocols with at-a-glance status (active/inactive, last latency reading).
- Navigation to each protocol's detailed panel.

## 7. User Flows

1. **Explore a single protocol:** User opens the app → selects a protocol panel → clicks "Connect/Start" → sends a test message → watches animation + logs + metrics update live → disconnects.
2. **Compare protocols:** User opens Battle Mode → selects 2+ protocols → starts identical load → watches comparative charts update in real time → reviews summary at the end.
3. **Review architecture:** User (e.g., an interviewer) opens the deployed app, tries a couple of protocols, and can understand at a glance what's different about each without reading external docs.

## 8. Success Metrics (how we know v1 is "done" and good)

- All 10 protocol modes are demonstrably working with real backend logic (verifiable via curl/grpcurl/devtools, not just UI).
- Each protocol panel shows live-updating, accurate metrics (validated against an external tool like k6/ghz during testing).
- The full app runs via a single `docker-compose up` locally and is also deployed to a public URL.
- The builder can explain, from memory, the connection lifecycle and duplex model of every protocol implemented (this is a learning-project success metric, not just a technical one).

## 9. Constraints & Assumptions

- Solo builder project; no team, no external stakeholders driving scope.
- Built incrementally over an estimated 14–20 week part-time learning timeline.
- Should work in modern evergreen browsers (Chrome/Firefox/Edge); no legacy browser support required.
- Public deployment budget assumed to be free/hobby-tier (Fly.io, Railway, or a low-cost VPS).

## 10. Open Questions / Decisions Needed During Build

- Exact hosting target for public deployment (decide before Phase 14 / deployment task).
- Whether to add authentication later if the tool is shared more broadly (currently: no).
- Whether historical metrics persistence (e.g., SQLite) is worth adding post-v1.

---

*This PRD defines WHAT is being built and WHY. See the companion TRD for HOW it is built (architecture, technology choices, and technical constraints).*
