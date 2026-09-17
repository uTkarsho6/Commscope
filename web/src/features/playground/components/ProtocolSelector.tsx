import type { ProtocolType, ProtocolMeta } from '../../../domain/types';

// 1. Array of metadata definitions for our 7 backend protocols
export const PROTOCOL_LIST: ProtocolMeta[] = [
  {
    id: 'http',
    name: 'HTTP Baseline',
    category: 'Request-Response',
    description: 'Stateless request-response communication. Each request opens a short-lived connection.',
  },
  {
    id: 'polling',
    name: 'Short Polling',
    category: 'Request-Response',
    description: 'Client sends recurring HTTP GET requests on a timer to check for new messages.',
  },
  {
    id: 'longpolling',
    name: 'Long Polling',
    category: 'Request-Response',
    description: 'Server holds ("hangs") the GET request open until data arrives or a timeout expires.',
  },
  {
    id: 'sse',
    name: 'Server-Sent Events',
    category: 'Streaming',
    description: 'Persistent 1-way HTTP stream over which server pushes live event data chunks to client.',
  },
  {
    id: 'ws',
    name: 'WebSockets',
    category: 'Streaming',
    description: 'Full-duplex bi-directional TCP socket allowing asynchronous client & server frames.',
  },
  {
    id: 'grpc',
    name: 'gRPC Unary / Stream',
    category: 'Streaming',
    description: 'HTTP/2 binary transport with strongly-typed Protocol Buffers schema.',
  },
  {
    id: 'webrtc',
    name: 'WebRTC Peer-to-Peer',
    category: 'P2P',
    description: 'Direct peer-to-peer RTCDataChannel connection bypassing intermediate server relays.',
  },
];

interface ProtocolSelectorProps {
  selectedProtocol: ProtocolType;
  onSelectProtocol: (protocol: ProtocolType) => void;
}

export function ProtocolSelector({ selectedProtocol, onSelectProtocol }: ProtocolSelectorProps) {
  // Find current selected metadata object
  const activeMeta = PROTOCOL_LIST.find((p) => p.id === selectedProtocol) || PROTOCOL_LIST[0];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
      {/* 2. Compact 1-Row Navigation Tab Bar */}
      <div
        className="workbench-panel-subdued"
        style={{
          padding: '0.25rem',
          display: 'flex',
          gap: '0.25rem',
          overflowX: 'auto',
        }}
      >
        {PROTOCOL_LIST.map((item) => {
          const isSelected = item.id === selectedProtocol;
          return (
            <button
              key={item.id}
              onClick={() => onSelectProtocol(item.id)}
              style={{
                flex: 1,
                padding: '0.4rem 0.6rem',
                fontSize: '0.75rem',
                fontFamily: 'var(--font-sans)',
                fontWeight: isSelected ? 600 : 400,
                borderRadius: '3px',
                border: isSelected ? '1px solid var(--color-primary)' : '1px solid transparent',
                cursor: 'pointer',
                backgroundColor: isSelected ? 'var(--color-surface)' : 'transparent',
                color: isSelected ? 'var(--color-primary)' : 'var(--color-text-secondary)',
                whiteSpace: 'nowrap',
                transition: 'all 0.15s ease',
              }}
            >
              {item.name}
            </button>
          );
        })}
      </div>

      {/* 3. Selected Protocol Description Bar */}
      <div
        className="workbench-panel"
        style={{
          padding: '0.75rem 1rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
            <h3 style={{ fontSize: '0.85rem', fontWeight: 600, margin: 0, color: 'var(--color-text-primary)' }}>
              {activeMeta.name}
            </h3>
            {/* Category Badge */}
            <span
              style={{
                fontSize: '0.65rem',
                fontFamily: 'var(--font-mono)',
                textTransform: 'uppercase',
                padding: '0.1rem 0.4rem',
                borderRadius: '2px',
                backgroundColor: 'var(--color-primary-light)',
                color: 'var(--color-primary)',
              }}
            >
              {activeMeta.category}
            </span>
          </div>
          <p style={{ fontSize: '0.75rem', color: 'var(--color-text-secondary)', margin: 0 }}>
            {activeMeta.description}
          </p>
        </div>
      </div>
    </div>
  );
}
