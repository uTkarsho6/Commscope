import { useEffect, useRef, useState } from 'react';
import type { ProtocolType } from '../../../domain/types';
import type { CommunicationEvent } from '../../../domain/event';
import { eventBus } from '../../../domain/eventBus';

interface VisualParticle {
  id: string;
  direction: 'outbound' | 'inbound';
  progress: number; // 0.0 (start) to 1.0 (end)
  color: string;
}

interface PacketVisualizerProps {
  selectedProtocol: ProtocolType;
  isServerOnline: boolean;
}

export function PacketVisualizer({ selectedProtocol, isServerOnline }: PacketVisualizerProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [particles, setParticles] = useState<VisualParticle[]>([]);
  const [isWaiting, setIsWaiting] = useState<boolean>(false);

  // 1. Subscribe to EventBus & handle incoming CommunicationEvents
  useEffect(() => {
    const unsubscribe = eventBus.subscribe((event: CommunicationEvent) => {
      // Long Polling hanging state
      if (event.topologyState === 'waiting') {
        setIsWaiting(true);
        return;
      } else {
        setIsWaiting(false);
      }

      // Add new particle to canvas animation queue
      const particle: VisualParticle = {
        id: event.id,
        direction: event.direction === 'outbound' ? 'outbound' : 'inbound',
        progress: 0,
        color: event.direction === 'outbound' ? 'var(--color-primary)' : 'var(--color-live)',
      };

      setParticles((prev) => [...prev, particle]);
    });

    // Cleanup listener on unmount to prevent memory leaks!
    return () => unsubscribe();
  }, []);

  // 2. 60 FPS HTML5 Canvas Animation Loop
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animId: number;

    const render = () => {
      ctx.clearRect(0, 0, canvas.width, canvas.height);

      const padding = 70;
      const startX = padding;
      const endX = canvas.width - padding;
      const centerY = canvas.height / 2;

      // Draw Topology Connection Line
      ctx.beginPath();
      ctx.moveTo(startX, centerY);
      ctx.lineTo(endX, centerY);
      ctx.strokeStyle = isServerOnline ? '#cbd5e1' : '#f87171';
      ctx.lineWidth = 2;
      ctx.stroke();

      // Draw Client Node (Left)
      ctx.beginPath();
      ctx.arc(startX, centerY, 16, 0, Math.PI * 2);
      ctx.fillStyle = '#ffffff';
      ctx.strokeStyle = '#2f5de3';
      ctx.lineWidth = 2;
      ctx.fill();
      ctx.stroke();

      ctx.fillStyle = '#0f172a';
      ctx.font = '600 10px var(--font-sans)';
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(selectedProtocol === 'webrtc' ? 'PEER A' : 'CLIENT', startX, centerY);

      // Draw Server / Peer Node (Right)
      ctx.beginPath();
      ctx.arc(endX, centerY, 18, 0, Math.PI * 2);
      ctx.fillStyle = '#ffffff';
      ctx.strokeStyle = selectedProtocol === 'webrtc' ? '#9333ea' : '#2f5de3';
      ctx.lineWidth = 2;
      ctx.fill();
      ctx.stroke();

      ctx.fillStyle = '#0f172a';
      ctx.font = '600 10px var(--font-sans)';
      ctx.fillText(selectedProtocol === 'webrtc' ? 'PEER B' : 'GO SRV', endX, centerY);

      // Draw Particles moving along wire
      setParticles((prev) =>
        prev
          .map((p) => {
            const nextProgress = p.progress + 0.04;
            const currentX =
              p.direction === 'outbound'
                ? startX + (endX - startX) * nextProgress
                : endX - (endX - startX) * nextProgress;

            // Draw glowing particle dot
            ctx.beginPath();
            ctx.arc(currentX, centerY, 6, 0, Math.PI * 2);
            ctx.fillStyle = p.direction === 'outbound' ? '#2f5de3' : '#10b981';
            ctx.fill();

            return { ...p, progress: nextProgress };
          })
          .filter((p) => p.progress <= 1)
      );

      animId = requestAnimationFrame(render);
    };

    render();

    return () => cancelAnimationFrame(animId);
  }, [selectedProtocol, isServerOnline]);

  return (
    <div className="workbench-panel" style={{ padding: '1rem', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--color-text-secondary)', textTransform: 'uppercase' }}>
          Packet Flow Visualizer (60 FPS)
        </span>

        {isWaiting && (
          <span style={{ fontSize: '0.65rem', fontFamily: 'var(--font-mono)', padding: '0.1rem 0.5rem', borderRadius: '3px', backgroundColor: '#fef3c7', color: '#b45309' }}>
            WAITING FOR SERVER RESPONSE...
          </span>
        )}
      </div>

      <canvas
        ref={canvasRef}
        width={750}
        height={90}
        style={{
          width: '100%',
          height: '90px',
          backgroundColor: 'var(--color-surface-subdued)',
          borderRadius: '4px',
        }}
      />
    </div>
  );
}
