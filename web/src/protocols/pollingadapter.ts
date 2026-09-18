import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const BASE_URL = 'http://localhost:8080';

export class PollingAdapter {
  private pollIntervalId: number | null = null;
  private lastSeenTimestamp: number = Date.now();

  // 1. Send Message to Short Polling Store
  async send(clientId: string, messageText: string): Promise<any> {
    const timestamp = new Date().toLocaleTimeString();

    const outboundEvent: CommunicationEvent = {
      id: `evt-poll-send-${Date.now()}`,
      timestamp,
      protocol: 'polling',
      direction: 'outbound',
      summary: `POST /api/polling/send`,
      payload: { client_id: clientId, message: messageText },
      topologyState: 'client_to_server',
    };
    eventBus.emit(outboundEvent);

    const start = performance.now();
    try {
      const response = await fetch(`${BASE_URL}/api/polling/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, message: messageText }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      const inboundEvent: CommunicationEvent = {
        id: `evt-poll-send-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'polling',
        direction: 'inbound',
        summary: `POLL Send OK (${latencyMs}ms)`,
        payload: data,
        latencyMs,
        topologyState: 'server_to_client',
      };
      eventBus.emit(inboundEvent);

      return data;
    } catch (err: any) {
      const errorEvent: CommunicationEvent = {
        id: `evt-poll-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'polling',
        direction: 'error',
        summary: `Polling Send Failed: ${err.message || err}`,
        payload: { error: err.message || 'Network error' },
      };
      eventBus.emit(errorEvent);
      throw err;
    }
  }

  // 2. Start Short Polling Loop (fires GET /api/polling/messages every 3 seconds)
  startPolling(intervalMs = 3000): void {
    this.stopPolling(); // Clear any existing timer
    this.lastSeenTimestamp = Date.now();

    this.pollIntervalId = window.setInterval(async () => {
      const start = performance.now();
      const timestampStr = new Date().toLocaleTimeString();

      // Emit Outbound Poll Trigger Event
      eventBus.emit({
        id: `evt-poll-get-out-${Date.now()}`,
        timestamp: timestampStr,
        protocol: 'polling',
        direction: 'outbound',
        summary: `GET /api/polling/messages?since=${this.lastSeenTimestamp}`,
        topologyState: 'client_to_server',
      });

      try {
        const response = await fetch(
          `${BASE_URL}/api/polling/messages?since=${this.lastSeenTimestamp}`
        );
        const end = performance.now();
        const latencyMs = Math.round((end - start) * 100) / 100;
        const messages = await response.json();

        if (Array.isArray(messages) && messages.length > 0) {
          this.lastSeenTimestamp = Date.now();
        }

        // Emit Inbound Poll Result Event
        eventBus.emit({
          id: `evt-poll-get-in-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'polling',
          direction: 'inbound',
          summary: `POLL GET (${messages.length || 0} msgs, ${latencyMs}ms)`,
          payload: messages,
          latencyMs,
          topologyState: 'server_to_client',
        });
      } catch (err: any) {
        eventBus.emit({
          id: `evt-poll-get-err-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'polling',
          direction: 'error',
          summary: `Polling GET Failed`,
          payload: { error: err.message },
        });
      }
    }, intervalMs);
  }

  // 3. Stop Polling Loop
  stopPolling(): void {
    if (this.pollIntervalId !== null) {
      clearInterval(this.pollIntervalId);
      this.pollIntervalId = null;
    }
  }
}

// Export singleton instance
export const pollingAdapter = new PollingAdapter();
