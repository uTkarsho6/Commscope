import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const BASE_URL = 'http://localhost:8080';

export class SSEAdapter {
  private eventSource: EventSource | null = null;

  // 1. Send Message to SSE Stream Broadcaster
  async send(clientId: string, messageText: string): Promise<any> {
    const timestamp = new Date().toLocaleTimeString();

    const outboundEvent: CommunicationEvent = {
      id: `evt-sse-send-${Date.now()}`,
      timestamp,
      protocol: 'sse',
      direction: 'outbound',
      summary: `POST /api/sse/send`,
      payload: { client_id: clientId, message: messageText },
      topologyState: 'client_to_server',
    };
    eventBus.emit(outboundEvent);

    const start = performance.now();
    try {
      const response = await fetch(`${BASE_URL}/api/sse/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, message: messageText }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      const inboundEvent: CommunicationEvent = {
        id: `evt-sse-send-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'inbound',
        summary: `SSE Send Published (${latencyMs}ms)`,
        payload: data,
        latencyMs,
        topologyState: 'server_to_client',
      };
      eventBus.emit(inboundEvent);

      return data;
    } catch (err: any) {
      const errorEvent: CommunicationEvent = {
        id: `evt-sse-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'error',
        summary: `SSE Send Failed`,
        payload: { error: err.message || 'Network error' },
      };
      eventBus.emit(errorEvent);
      throw err;
    }
  }

  // 2. Open Persistent SSE Stream Connection (EventSource)
  connect(): void {
    if (this.eventSource) return; // Already connected

    const start = performance.now();
    this.eventSource = new EventSource(`${BASE_URL}/api/sse/events`);

    this.eventSource.onopen = () => {
      eventBus.emit({
        id: `evt-sse-sys-conn-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'system',
        summary: `SSE EventSource Connected (text/event-stream)`,
        topologyState: 'streaming',
      });
    };

    // 3. Receive Streamed Data Chunks from Server
    this.eventSource.onmessage = (event) => {
      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;

      let payload: any;
      try {
        payload = JSON.parse(event.data);
      } catch {
        payload = event.data;
      }

      eventBus.emit({
        id: `evt-sse-msg-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'inbound',
        summary: `SSE Stream Chunk Received`,
        payload,
        latencyMs,
        topologyState: 'streaming',
      });
    };

    this.eventSource.onerror = () => {
      eventBus.emit({
        id: `evt-sse-sys-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'error',
        summary: `SSE EventSource Connection Error`,
      });
    };
  }

  // 4. Close SSE Stream Connection
  disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;

      eventBus.emit({
        id: `evt-sse-sys-disc-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'sse',
        direction: 'system',
        summary: `SSE EventSource Connection Closed`,
      });
    }
  }
}

// Export singleton instance
export const sseAdapter = new SSEAdapter();
