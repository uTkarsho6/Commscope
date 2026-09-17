// 1. Import React Hooks for managing state (memory) and side-effects (fetching data)
import { useState, useEffect } from 'react';
import type { ProtocolType } from './domain/types';
import { ProtocolSelector } from './features/playground/components/ProtocolSelector';


export function App() {
  // 2. STATE 1: Controls which view mode is visible ('playground' or 'battle')
  //    - activeTab: current value ('playground' by default)
  //    - setActiveTab: function to update activeTab when user clicks a navigation button
  const [activeTab, setActiveTab] = useState<'playground' | 'battle'>('playground');

  // 3. STATE 2: Stores whether the Go Backend server (:8080) is responding
  //    - serverOnline: boolean (true/false)
  //    - setServerOnline: function to update serverOnline status
  const [serverOnline, setServerOnline] = useState<boolean>(false);

  const [selectedProtocol, setSelectedProtocol] = useState<ProtocolType>('http');


  // 4. SIDE EFFECT HOOK: Runs ONCE when this component first renders on screen ([] dependency array)
  useEffect(() => {
    // Make HTTP GET request to Go server health endpoint
    fetch('http://localhost:8080/health')
      .then((res) => setServerOnline(res.ok))  // If 200 OK -> serverOnline = true
      .catch(() => setServerOnline(false));     // If network error -> serverOnline = false
  }, []); // [] means "run only once on page load"

  return (
    // 5. MAIN CONTAINER: Centered layout with maximum width of 1200px
    <div style={{ maxWidth: '1200px', margin: '0 auto', padding: '1.5rem' }}>

      {/* 6. HEADER BAR: Top panel with title on left and controls on right */}
      <header
        className="workbench-panel" // Uses our custom CSS rule from theme.css
        style={{
          padding: '1rem 1.5rem',
          marginBottom: '1.5rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        {/* Left Side: Branding Title & Subtitle */}
        <div>
          <h1 style={{ fontSize: '1.25rem', fontWeight: 600, margin: 0, color: 'var(--color-text-primary)' }}>
            CommScope
          </h1>
          <p style={{ fontSize: '0.75rem', color: 'var(--color-text-secondary)', margin: 0 }}>
            Interactive Real-Time Communication Playground
          </p>
        </div>

        {/* Right Side: Health Status Indicator & Navigation Buttons */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '1.5rem' }}>

          {/* Health Status Indicator */}
          <div style={{ fontSize: '0.75rem', fontFamily: 'var(--font-mono)', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
            {/* Dynamic Colored Dot: Green if online, Red if offline */}
            <span
              style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: serverOnline ? 'var(--color-live)' : 'var(--color-error)',
              }}
            />
            {/* Text showing ONLINE or OFFLINE dynamically */}
            <span>
              Go Server (:8080): <strong>{serverOnline ? 'ONLINE' : 'OFFLINE'}</strong>
            </span>
          </div>

          {/* Mode Switcher Buttons */}
          <div className="workbench-panel-subdued" style={{ padding: '0.25rem', display: 'flex', gap: '0.25rem' }}>

            {/* Button 1: Switch to Playground */}
            <button
              onClick={() => setActiveTab('playground')}
              style={{
                padding: '0.35rem 0.85rem',
                fontSize: '0.75rem',
                fontWeight: 600,
                borderRadius: '3px',
                border: 'none',
                cursor: 'pointer',
                // Highlight button background blue if Playground is selected
                backgroundColor: activeTab === 'playground' ? 'var(--color-primary)' : 'transparent',
                color: activeTab === 'playground' ? '#ffffff' : 'var(--color-text-secondary)',
              }}
            >
              Playground
            </button>

            {/* Button 2: Switch to Battle Mode */}
            <button
              onClick={() => setActiveTab('battle')}
              style={{
                padding: '0.35rem 0.85rem',
                fontSize: '0.75rem',
                fontWeight: 600,
                borderRadius: '3px',
                border: 'none',
                cursor: 'pointer',
                // Highlight button background blue if Battle Mode is selected
                backgroundColor: activeTab === 'battle' ? 'var(--color-primary)' : 'transparent',
                color: activeTab === 'battle' ? '#ffffff' : 'var(--color-text-secondary)',
              }}
            >
              Battle Mode
            </button>
          </div>
        </div>
      </header>

      {/* 7. MAIN VIEW AREA: Renders Playground OR Battle Mode dynamically */}
      <main>
        {/* Ternary Operator: Condition ? (Show If True) : (Show If False) */}
                {activeTab === 'playground' ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
            {/* Compact Protocol Selector Bar */}
            <ProtocolSelector
              selectedProtocol={selectedProtocol}
              onSelectProtocol={setSelectedProtocol}
            />

            {/* Placeholder for upcoming Packet Visualizer Canvas & Metrics */}
            <div className="workbench-panel" style={{ padding: '1.5rem' }}>
              <h4 style={{ fontSize: '0.85rem', margin: 0 }}>
                Selected Protocol Engine: <strong style={{ color: 'var(--color-primary)' }}>{selectedProtocol.toUpperCase()}</strong>
              </h4>
            </div>
          </div>
        ) : (
          <div className="workbench-panel" style={{ padding: '1.5rem' }}>
            <h2 style={{ fontSize: '1rem', margin: 0 }}>Battle Mode</h2>
            <p style={{ fontSize: '0.85rem', color: 'var(--color-text-secondary)' }}>
              Compare two protocols side-by-side under identical workload.
            </p>
          </div>
        )}

      </main>
    </div>
  );
}

export default App;
