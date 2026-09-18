import { useEffect, useState } from 'react';
import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

interface EventLogConsoleProps {
  onSelectEvent: (event: CommunicationEvent | null) => void;
  selectedEventId?: string;
}

export function EventLogConsole({ onSelectEvent, selectedEventId }: EventLogConsoleProps) {
  const [events, setEvents] = useState<CommunicationEvent[]>([]);

  useEffect(() => {
    const unsubscribe = eventBus.subscribe((newEvent) => {
      setEvents((prev) => [newEvent, ...prev.slice(0, 99)]); // Keep last 100 events
    });

    return () => unsubscribe();
  }, []);

  const getBadgeStyle = (direction: CommunicationEvent['direction']) => {
    switch (direction) {
      case 'outbound':
        return { background: '#dbeafe', color: '#1e40af', border: '1px solid #bfdbfe' };
      case 'inbound':
        return { background: '#dcfce7', color: '#15803d', border: '1px solid #bbf7d0' };
      case 'system':
        return { background: '#f3e8ff', color: '#7e22ce', border: '1px solid #e9d5ff' };
      case 'error':
        return { background: '#ffe4e6', color: '#be123c', border: '1px solid #fecdd3' };
    }
  };

  return (
    <div className="workbench-panel" style={{ padding: '1rem', display: 'flex', flexDirection: 'column', height: '100%', minHeight: '300px' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.75rem' }}>
        <span style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--color-text-secondary)', textTransform: 'uppercase' }}>
          Real-time Event Console ({events.length} Events)
        </span>
        <button
          onClick={() => {
            setEvents([]);
            onSelectEvent(null);
          }}
          style={{ fontSize: '0.65rem', padding: '0.2rem 0.5rem', background: 'var(--color-surface)', border: '1px solid var(--color-border)', borderRadius: '4px', cursor: 'pointer' }}
        >
          CLEAR LOGS
        </button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '0.5rem', fontFamily: 'var(--font-mono)', fontSize: '0.75rem' }}>
        {events.length === 0 ? (
          <div style={{ color: 'var(--color-text-tertiary)', textAlign: 'center', padding: '2rem 0' }}>
            No network events recorded yet. Send a request to see telemetry frames.
          </div>
        ) : (
          events.map((evt) => {
            const isSelected = evt.id === selectedEventId;
            const badge = getBadgeStyle(evt.direction);

            return (
              <div
                key={evt.id}
                onClick={() => onSelectEvent(evt)}
                style={{
                  padding: '0.5rem 0.75rem',
                  borderRadius: '5px',
                  background: isSelected ? '#eff6ff' : 'var(--color-surface)',
                  border: isSelected ? '1px solid var(--color-primary)' : '1px solid var(--color-border)',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: '0.75rem',
                  transition: 'border 0.15s ease',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flex: 1, minWidth: 0 }}>
                  <span style={{ fontSize: '0.65rem', fontWeight: 700, padding: '0.1rem 0.4rem', borderRadius: '3px', textTransform: 'uppercase', ...badge }}>
                    {evt.direction}
                  </span>
                  <span style={{ color: 'var(--color-text-tertiary)', fontSize: '0.7rem' }}>{evt.timestamp}</span>
                  <span style={{ fontWeight: 600, color: 'var(--color-text-primary)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {evt.summary}
                  </span>
                </div>

                {evt.latencyMs != null && (
                  <span style={{ color: 'var(--color-live)', fontWeight: 600, fontSize: '0.7rem' }}>
                    {evt.latencyMs}ms
                  </span>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
