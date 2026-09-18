import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const GRPC_PROXY_URL = 'http://localhost:8080/api/grpc';

export class GRPCAdapter {

  // 1. Unary RPC: 1 Request -> 1 Response

  async sendUnaryMessage(clientId: string, messageText: string): Promise<any> {
    const start = performance.now();

    eventBus.emit({
      id: `evt-grpc-unary-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'grpc',
      direction: 'outbound',
      summary: `gRPC Unary RPC: SendMessage`,
      payload: { method: 'SendMessage', client_id: clientId, message: messageText },
      topologyState: 'client_to_server',
    });

    try {
      const response = await fetch(`${GRPC_PROXY_URL}/send`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-gRPC-Call-Type': 'Unary',
        },
        body: JSON.stringify({ client_id: clientId, message: messageText }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      eventBus.emit({
        id: `evt-grpc-unary-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'grpc',
        direction: 'inbound',
        summary: `gRPC Unary OK (${latencyMs}ms)`,
        payload: data,
        latencyMs,
        topologyState: 'server_to_client',
      });

      return data;
    } catch (err: any) {
      eventBus.emit({
        id: `evt-grpc-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'grpc',
        direction: 'error',
        summary: `gRPC Unary Call Failed`,
        payload: { error: err.message || 'gRPC call error' },
      });
      throw err;
    }
  }


  // 2. Server Streaming RPC: 1 Request -> Stream of Responses

  async startServerStream(clientId: string, count: number = 5): Promise<void> {
    const start = performance.now();

    eventBus.emit({
      id: `evt-grpc-srv-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'grpc',
      direction: 'outbound',
      summary: `gRPC Server Streaming RPC: StreamMessages`,
      payload: { method: 'StreamMessages', client_id: clientId, count },
      topologyState: 'streaming',
    });

    try {
      const response = await fetch(`${GRPC_PROXY_URL}/stream?client_id=${clientId}&count=${count}`);
      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) return;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const end = performance.now();
        const latencyMs = Math.round((end - start) * 100) / 100;
        const chunkText = decoder.decode(value);

        eventBus.emit({
          id: `evt-grpc-srv-in-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'grpc',
          direction: 'inbound',
          summary: `gRPC Server Stream Chunk Received`,
          payload: { chunk: chunkText },
          latencyMs,
          topologyState: 'streaming',
        });
      }
    } catch (err: any) {
      eventBus.emit({
        id: `evt-grpc-srv-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'grpc',
        direction: 'error',
        summary: `gRPC Server Stream Error`,
        payload: { error: err.message || 'Stream error' },
      });
    }
  }


  // 3. Client Streaming RPC: Stream of Requests -> 1 Response

  async sendClientStreamBatch(clientId: string, messages: string[]): Promise<any> {
    const start = performance.now();

    eventBus.emit({
      id: `evt-grpc-cli-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'grpc',
      direction: 'outbound',
      summary: `gRPC Client Streaming RPC: SendBatchMessages (${messages.length} chunks)`,
      payload: { method: 'SendBatchMessages', client_id: clientId, batchSize: messages.length },
      topologyState: 'client_to_server',
    });

    try {
      const response = await fetch(`${GRPC_PROXY_URL}/batch`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-gRPC-Call-Type': 'ClientStreaming',
        },
        body: JSON.stringify({ client_id: clientId, messages }),
      });

      const end = performance.now();
      const latencyMs = Math.round((end - start) * 100) / 100;
      const data = await response.json();

      eventBus.emit({
        id: `evt-grpc-cli-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'grpc',
        direction: 'inbound',
        summary: `gRPC Client Stream Summary ACK (${latencyMs}ms)`,
        payload: data,
        latencyMs,
        topologyState: 'server_to_client',
      });

      return data;
    } catch (err: any) {
      eventBus.emit({
        id: `evt-grpc-cli-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'grpc',
        direction: 'error',
        summary: `gRPC Client Streaming Error`,
        payload: { error: err.message },
      });
      throw err;
    }
  }


  // 4. Bi-directional Streaming RPC: Stream <-> Stream Concurrent

  startBidiStream(clientId: string): { send: (msg: string) => void; close: () => void } {
    const start = performance.now();

    eventBus.emit({
      id: `evt-grpc-bidi-init-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'grpc',
      direction: 'system',
      summary: `gRPC Bi-directional Stream Initialized (ChatStream)`,
      payload: { method: 'ChatStream', client_id: clientId },
      topologyState: 'streaming',
    });

    return {
      send: async (msgText: string) => {
        const frameStart = performance.now();

        eventBus.emit({
          id: `evt-grpc-bidi-out-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'grpc',
          direction: 'outbound',
          summary: `gRPC Bi-di Upstream Frame Sent`,
          payload: { text: msgText },
          topologyState: 'streaming',
        });

        try {
          const response = await fetch(`${GRPC_PROXY_URL}/bidi`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ client_id: clientId, message: msgText }),
          });

          const frameEnd = performance.now();
          const latencyMs = Math.round((frameEnd - frameStart) * 100) / 100;
          const data = await response.json();

          eventBus.emit({
            id: `evt-grpc-bidi-in-${Date.now()}`,
            timestamp: new Date().toLocaleTimeString(),
            protocol: 'grpc',
            direction: 'inbound',
            summary: `gRPC Bi-di Downstream Frame Received`,
            payload: data,
            latencyMs,
            topologyState: 'streaming',
          });
        } catch (err: any) {
          eventBus.emit({
            id: `evt-grpc-bidi-err-${Date.now()}`,
            timestamp: new Date().toLocaleTimeString(),
            protocol: 'grpc',
            direction: 'error',
            summary: `gRPC Bi-di Stream Frame Error`,
            payload: { error: err.message },
          });
        }
      },
      close: () => {
        eventBus.emit({
          id: `evt-grpc-bidi-close-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'grpc',
          direction: 'system',
          summary: `gRPC Bi-directional Stream Closed (EOF)`,
        });
      },
    };
  }
}

export const grpcAdapter = new GRPCAdapter();
