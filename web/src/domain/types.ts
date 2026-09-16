// Union type representing our 7 backend communication protocols
export type ProtocolType =
    | 'http' | 'polling' | 'sse' | 'webrtc' | 'ws' | 'longpolling' | 'grpc';


//Matches the JSON response struct returned by our Go backend for the /api/metrics/all endpoint
export interface ProtocolMetrics {
    Protocol: string;
    ActiveConnections: number;
    MessageCount: number;
    P50LatencyMs: number;
    P99LatencyMs: number;
    Throughput: number;
}

//  Metadata used for rendering protocol selector tabs and badges:
export interface ProtocolMeta {
    id: ProtocolType;
    name: string;
    category: 'Request-Response' | 'Streaming' | 'P2P';
    description: string;
}