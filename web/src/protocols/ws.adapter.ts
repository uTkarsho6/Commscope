import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const WS_URL = 'ws://localhost:8080/api/ws/connect';
const BASE_URL = 'http://localhost:8080';

export class WSAdapter {
  private socket: WebSocket | null = null;
  private isConnected = false;

  // 1. Open persistent WebSocket Connection
  connect(clientId: string = 'ws-client-browser'): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.socket && this.isConnected) {
        resolve();
        return;
      }

      const start = performance.now();
      const url = `${WS_URL}?client_id=${encodeURIComponent(clientId)}`;
      this.socket = new WebSocket(url);

      this.socket.onopen = () => {
        const end = performance.now();
        const latencyMs = Math.round((end - start) * 100) / 100;
        this.isConnected = true;

        eventBus.emit({
          id: `evt-ws-sys-conn-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'ws',
          direction: 'system',
          summary: `WebSocket Handshake 101 Switching Protocols (${latencyMs}ms)`,
          latencyMs,
          topologyState: 'streaming',
        });
        resolve();
      };

      this.socket.onmessage = (event) => {
        let payload: any;
        try {
          payload = JSON.parse(event.data);
        } catch {
          payload = event.data;
        }

        eventBus.emit({
          id: `evt-ws-msg-in-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'ws',
          direction: 'inbound',
          summary: `WebSocket Frame Received (Downstream)`,
          payload,
          topologyState: 'server_to_client',
        });
      };

      this.socket.onerror = (error) => {
        eventBus.emit({
          id: `evt-ws-err-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'ws',
          direction: 'error',
          summary: `WebSocket Connection Error`,
          payload: { error },
        });
        reject(error);
      };

      this.socket.onclose = () => {
        this.isConnected = false;
        this.socket = null;

        eventBus.emit({
          id: `evt-ws-sys-disc-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'ws',
          direction: 'system',
          summary: `WebSocket Connection Closed`,
        });
      };
    });
  }

  // 2. Send Upstream Frame directly over active WebSocket connection
  sendDirect(messageText: string): void {
    if (!this.socket || !this.isConnected) {
      throw new Error('WebSocket is not connected');
    }

    const payload = { text: messageText, timestamp: new Date().toISOString() };

    eventBus.emit({
      id: `evt-ws-send-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'ws',
      direction: 'outbound',
      summary: `WebSocket Frame Sent (Upstream)`,
      payload,
      topologyState: 'client_to_server',
    });

    this.socket.send(JSON.stringify(payload));
  }

  // 3. Send Message via HTTP POST to Backend WS Broadcast Hub
  async sendViaHTTP(clientId: string, messageText: string): Promise<any> {
    const start = performance.now();

    eventBus.emit({
      id: `evt-ws-post-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'ws',
      direction: 'outbound',
      summary: `POST /api/ws/send (Broadcast)`,
      payload: { client_id: clientId, text: messageText },
      topologyState: 'client_to_server',
    });

    try {
      const response = await fetch(`${BASE_URL}/api/ws/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ client_id: clientId, text: messageText }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      eventBus.emit({
        id: `evt-ws-post-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'ws',
        direction: 'inbound',
        summary: `WS Broadcast ACK (${latencyMs}ms)`,
        payload: data,
        latencyMs,
      });

      return data;
    } catch (err: any) {
      eventBus.emit({
        id: `evt-ws-post-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'ws',
        direction: 'error',
        summary: `WS Broadcast HTTP POST Failed`,
        payload: { error: err.message },
      });
      throw err;
    }
  }

  // 4. Close WebSocket Connection
  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
      this.isConnected = false;
    }
  }
}

export const wsAdapter = new WSAdapter();
