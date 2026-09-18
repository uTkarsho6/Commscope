import type { CommunicationEvent } from '../domain/event';
import { eventBus } from '../domain/eventBus';

const SIGNALING_URL = 'http://localhost:8080/api/webrtc/offer';

export class WebRTCAdapter {
  private pc: RTCPeerConnection | null = null;
  private dc: RTCDataChannel | null = null;
  private isConnected = false;

  // 1. Initiate WebRTC P2P Handshake (SDP Offer/Answer via Go Signaling Server)
  async connect(peerId: string = 'peer-browser-1'): Promise<void> {
    if (this.pc && this.isConnected) return;

    const start = performance.now();

    eventBus.emit({
      id: `evt-webrtc-sig-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'webrtc',
      direction: 'outbound',
      summary: `WebRTC Signaling SDP Offer Sent to Go Server`,
      payload: { peer_id: peerId },
      topologyState: 'client_to_server',
    });

    try {
      // Create RTCPeerConnection with Google Public STUN server
      this.pc = new RTCPeerConnection({
        iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
      });

      // Create DataChannel for low-latency binary/text data
      this.dc = this.pc.createDataChannel('commscope-dc');

      this.dc.onopen = () => {
        const end = performance.now();
        const latencyMs = Math.round((end - start) * 100) / 100;
        this.isConnected = true;

        eventBus.emit({
          id: `evt-webrtc-dc-open-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'webrtc',
          direction: 'system',
          summary: `WebRTC DataChannel OPEN (P2P Connected in ${latencyMs}ms)`,
          latencyMs,
          topologyState: 'p2p',
        });
      };

      this.dc.onmessage = (event) => {
        let payload: any;
        try {
          payload = JSON.parse(event.data);
        } catch {
          payload = event.data;
        }

        eventBus.emit({
          id: `evt-webrtc-msg-in-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'webrtc',
          direction: 'inbound',
          summary: `WebRTC DataChannel Message Received (Direct P2P)`,
          payload,
          topologyState: 'p2p',
        });
      };

      this.dc.onclose = () => {
        this.isConnected = false;
        eventBus.emit({
          id: `evt-webrtc-dc-close-${Date.now()}`,
          timestamp: new Date().toLocaleTimeString(),
          protocol: 'webrtc',
          direction: 'system',
          summary: `WebRTC DataChannel Closed`,
        });
      };

      // Create local SDP offer
      const offer = await this.pc.createOffer();
      await this.pc.setLocalDescription(offer);

      // Send SDP Offer to Go Backend Signaling endpoint
      const response = await fetch(SIGNALING_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type: 'offer',
          peer_id: peerId,
          sdp: offer.sdp,
        }),
      });

      const answerData = await response.json();

      eventBus.emit({
        id: `evt-webrtc-sig-in-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'webrtc',
        direction: 'inbound',
        summary: `WebRTC Signaling SDP Answer Received from Go Server`,
        payload: answerData,
        topologyState: 'server_to_client',
      });

      // Set Remote Description (Go server SDP answer)
      if (answerData.sdp) {
        await this.pc.setRemoteDescription(
          new RTCSessionDescription({ type: 'answer', sdp: answerData.sdp })
        );
      }
    } catch (err: any) {
      eventBus.emit({
        id: `evt-webrtc-err-${Date.now()}`,
        timestamp: new Date().toLocaleTimeString(),
        protocol: 'webrtc',
        direction: 'error',
        summary: `WebRTC Handshake Failed`,
        payload: { error: err.message },
      });
      throw err;
    }
  }

  // 2. Send Data directly over P2P DataChannel (No Server Relay!)
  sendDirectP2P(messageText: string): void {
    if (!this.dc || this.dc.readyState !== 'open') {
      throw new Error('WebRTC DataChannel is not open');
    }

    const payload = { text: messageText, timestamp: new Date().toISOString() };

    eventBus.emit({
      id: `evt-webrtc-send-out-${Date.now()}`,
      timestamp: new Date().toLocaleTimeString(),
      protocol: 'webrtc',
      direction: 'outbound',
      summary: `WebRTC Direct P2P Message Sent (Bypasses Server)`,
      payload,
      topologyState: 'p2p',
    });

    this.dc.send(JSON.stringify(payload));
  }

  // 3. Disconnect WebRTC session
  disconnect(): void {
    if (this.dc) {
      this.dc.close();
      this.dc = null;
    }
    if (this.pc) {
      this.pc.close();
      this.pc = null;
    }
    this.isConnected = false;
  }
}

export const webrtcAdapter = new WebRTCAdapter();
