import { Injectable, OnDestroy } from '@angular/core';
import { Subject, Observable } from 'rxjs';
 
export interface IncomingMessage {
  type: string;
  conversation_id: string;
  ciphertext: string;
  sender_id: string;
  display_name: string;
  profile_picture_url: string;
  message_id: string;
  client_message_id?: string;
  user_id?: string;
  username?: string;
  // expires_at is the Unix timestamp (seconds) at which the message will be deleted,
  // present only on new_message events when a lifetime is set on the conversation.
  expires_at?: number;
}
 
export interface OutgoingMessage {
  type: string;
  conversation_id: string;
  ciphertext: string;
  client_message_id?: string;
}
 
@Injectable({
  providedIn: 'root',
})
export class MessagingService implements OnDestroy {
  private socket: WebSocket | null = null;
  private messageSubject = new Subject<IncomingMessage>();
  messages$: Observable<IncomingMessage> = this.messageSubject.asObservable();
 
  connect(): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      return; // already connected
    }
 
    this.socket = new WebSocket('ws://localhost:8080/ws');
 
    this.socket.onopen = () => {
      console.log('[MessagingService] WebSocket connected');
    };
 
    this.socket.onmessage = (event: MessageEvent) => {
      try {
        const msg: IncomingMessage = JSON.parse(event.data);
        if (msg.type === 'new_message' || msg.type === 'messages_updated' || msg.type === 'message_ack' || msg.type === 'profile_updated') {
          this.messageSubject.next(msg);
        }
      } catch (e) {
        console.error('[MessagingService] Failed to parse message:', e);
      }
    };
 
    this.socket.onerror = (err) => {
      console.error('[MessagingService] WebSocket error:', err);
    };
 
    this.socket.onclose = () => {
      console.log('[MessagingService] WebSocket closed');
      this.socket = null;
    };
  }
 
  sendMessage(conversationId: string, ciphertext: string, clientMessageId?: string): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      console.error('[MessagingService] Socket is not open. Cannot send.');
      return;
    }
 
    const payload: OutgoingMessage = {
      type: 'send_message',
      conversation_id: conversationId,
      ciphertext,
      client_message_id: clientMessageId,
    };
 
    this.socket.send(JSON.stringify(payload));
  }
 
  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
  }
 
  isConnected(): boolean {
    return this.socket !== null && this.socket.readyState === WebSocket.OPEN;
  }
 
  ngOnDestroy(): void {
    this.disconnect();
    this.messageSubject.complete();
  }
}
