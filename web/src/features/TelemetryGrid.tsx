import { useEffect, useState } from 'react';
import type { ProtocolMetrics, ProtocolType } from '../domain/types';

interface TelemetryGridProps {
  selectedProtocol: ProtocolType;
}

export function TelemetryGrid({ selectedProtocol }: TelemetryGridProps) {
  const [metrics, setMetrics] = useState<ProtocolMetrics | null>(null);

  useEffect(() => {
    let isMounted = true;

    const fetchMetrics = async () => {
      try {
        const res = await fetch(`http://localhost:8080/api/${selectedProtocol}/stats`);
        if (!res.ok) return;
        const data: ProtocolMetrics = await res.json();
        if (isMounted) setMetrics(data);
      } catch {
        // Backend offline or endpoint silent
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 1000);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [selectedProtocol]);

  return (
    <div className="workbench-panel" style={{ padding: '1rem' }}>
      <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--color-text-secondary)', textTransform: 'uppercase', marginBottom: '0.75rem' }}>
        Real-time Telemetry Metrics ({selectedProtocol.toUpperCase()})
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: '0.75rem' }}>
        {/* Active Connections */}
        <div style={{ padding: '0.75rem', borderRadius: '6px', background: 'var(--color-surface)', border: '1px solid var(--color-border)' }}>
          <span style={{ fontSize: '0.65rem', color: 'var(--color-text-tertiary)', textTransform: 'uppercase' }}>Active Conns</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 700, fontFamily: 'var(--font-mono)', color: 'var(--color-primary)' }}>
            {metrics?.ActiveConnections ?? 0}
          </div>
        </div>

        {/* Total Messages */}
        <div style={{ padding: '0.75rem', borderRadius: '6px', background: 'var(--color-surface)', border: '1px solid var(--color-border)' }}>
          <span style={{ fontSize: '0.65rem', color: 'var(--color-text-tertiary)', textTransform: 'uppercase' }}>Message Volume</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 700, fontFamily: 'var(--font-mono)' }}>
            {metrics?.MessageCount ?? 0}
          </div>
        </div>

        {/* P50 Latency */}
        <div style={{ padding: '0.75rem', borderRadius: '6px', background: 'var(--color-surface)', border: '1px solid var(--color-border)' }}>
          <span style={{ fontSize: '0.65rem', color: 'var(--color-text-tertiary)', textTransform: 'uppercase' }}>P50 Latency</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 700, fontFamily: 'var(--font-mono)', color: 'var(--color-live)' }}>
            {metrics?.P50LatencyMs != null ? `${metrics.P50LatencyMs}ms` : '--'}
          </div>
        </div>

        {/* P99 Latency */}
        <div style={{ padding: '0.75rem', borderRadius: '6px', background: 'var(--color-surface)', border: '1px solid var(--color-border)' }}>
          <span style={{ fontSize: '0.65rem', color: 'var(--color-text-tertiary)', textTransform: 'uppercase' }}>P99 Latency</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 700, fontFamily: 'var(--font-mono)', color: '#e11d48' }}>
            {metrics?.P99LatencyMs != null ? `${metrics.P99LatencyMs}ms` : '--'}
          </div>
        </div>

        {/* Throughput */}
        <div style={{ padding: '0.75rem', borderRadius: '6px', background: 'var(--color-surface)', border: '1px solid var(--color-border)' }}>
          <span style={{ fontSize: '0.65rem', color: 'var(--color-text-tertiary)', textTransform: 'uppercase' }}>Throughput</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 700, fontFamily: 'var(--font-mono)' }}>
            {metrics?.Throughput != null ? `${metrics.Throughput} msg/s` : '0 msg/s'}
          </div>
        </div>
      </div>
    </div>
  );
}
