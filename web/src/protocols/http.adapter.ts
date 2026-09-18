import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const BASE_URL = 'http://localhost:8080';

export class HTTPAdapter {
  async send(clientId: string, messageText: string): Promise<any> {
    const timestamp = new Date().toLocaleTimeString();

    // 1. Emit OUTBOUND Event (Triggers Client -> Server particle animation)
    const outboundEvent: CommunicationEvent = {
      id: `evt-out-${Date.now()}`,
      timestamp,
      protocol: 'http',
      direction: 'outbound',
      summary: `POST /api/http/send`,
      payload: { client_id: clientId, message: messageText },
      topologyState: 'client_to_server',
    };
    eventBus.emit(outboundEvent);

    const start = performance.now();

    try {
      // 2. Transmit HTTP Request over wire to Go Backend
      const response = await fetch(`${BASE_URL}/api/http/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, message: messageText }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      // 3. Emit INBOUND Event (Triggers Server -> Client particle animation & log)
      const inboundEvent: CommunicationEvent = {
        id: `evt-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'http',
        direction: 'inbound',
        summary: `HTTP 200 OK (${latencyMs}ms)`,
        payload: data,
        latencyMs,
        topologyState: 'server_to_client',
      };
      eventBus.emit(inboundEvent);

      return data;
    } catch (err: any) {
      // 4. Emit ERROR Event
      const errorEvent: CommunicationEvent = {
        id: `evt-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'http',
        direction: 'error',
        summary: `HTTP Request Failed: ${err.message || err}`,
        payload: { error: err.message || 'Network error' },
      };
      eventBus.emit(errorEvent);
      throw err;
    }
  }
}

// Export singleton instance
export const httpAdapter = new HTTPAdapter();
