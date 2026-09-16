import type { ProtocolType } from './types';

// Direction of the network traffic
export type EventDirection = 'outbound' | 'inbound' | 'system' | 'error';

// Visual communication pattern for Canvas particle animation
export type TopologyState =
  | 'client_to_server'
  | 'server_to_client'
  | 'waiting'
  | 'streaming'
  | 'p2p';

  // Protocol-agnostic communication event emitted by adapters
export interface CommunicationEvent {
  id: string;
  timestamp: string;
  protocol: ProtocolType;
  direction: EventDirection;
  summary: string;
  payload?: any;
  latencyMs?: number;
  payloadSizeBytes?: number;
  topologyState?: TopologyState;
}