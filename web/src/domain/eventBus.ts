import type { CommunicationEvent } from './event';

type EventListener = (event: CommunicationEvent) => void;

class EventBus {
  private listeners: Set<EventListener> = new Set();

  // Subscribe a component to receive live communication events
  // Returns an unsubscribe function to clean up when component unmounts!
  subscribe(listener: EventListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  // Emit a new communication event to all active listeners
  emit(event: CommunicationEvent): void {
    this.listeners.forEach((listener) => {
      try {
        listener(event);
      } catch (err) {
        console.error('Error in EventBus listener:', err);
      }
    });
  }

  // Remove all active listeners (useful for testing or cleanup)
  clear(): void {
    this.listeners.clear();
  }
}

// Export a single shared instance (Singleton Pattern)
export const eventBus = new EventBus();
