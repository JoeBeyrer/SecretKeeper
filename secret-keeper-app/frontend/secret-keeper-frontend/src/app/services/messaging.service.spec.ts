import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { MessagingService, IncomingMessage } from './messaging.service';

class MockWebSocket {
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState: number = MockWebSocket.OPEN;
  sentMessages: string[] = [];

  onopen: (() => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: (() => void) | null = null;

  constructor(public url: string) {}

  send(data: string): void {
    this.sentMessages.push(data);
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) this.onclose();
  }

  simulateMessage(data: object): void {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }));
    }
  }
}

describe('MessagingService', () => {
  let service: MessagingService;
  let mockSocket: MockWebSocket;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(MessagingService);

    vi.spyOn(window as any, 'WebSocket').mockImplementation(
      function (this: any, url: string) {
        mockSocket = new MockWebSocket(url);
        return mockSocket;
      } as any
    );
  });

  afterEach(() => {
    service.ngOnDestroy();
    vi.restoreAllMocks();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('connect()', () => {
    it('should create a WebSocket pointing to the correct URL', () => {
      service.connect();
      expect(mockSocket.url).toBe('ws://localhost:8080/ws');
    });

    it('should not create a second socket if already OPEN', () => {
      service.connect();
      const firstSocket = mockSocket;
      service.connect();
      expect(mockSocket).toBe(firstSocket);
    });

    it('should set socket to null on close', () => {
      service.connect();
      mockSocket.close();
      expect(service.isConnected()).toBe(false);
    });
  });

  describe('isConnected()', () => {
    it('should return false before connect() is called', () => {
      expect(service.isConnected()).toBe(false);
    });

    it('should return true when socket is OPEN', () => {
      service.connect();
      expect(service.isConnected()).toBe(true);
    });

    it('should return false after disconnect()', () => {
      service.connect();
      service.disconnect();
      expect(service.isConnected()).toBe(false);
    });
  });

  describe('sendMessage()', () => {
    it('should send a properly structured JSON payload', () => {
      service.connect();
      service.sendMessage('conv-1', 'cipher-abc');
      expect(mockSocket.sentMessages.length).toBe(1);
      const payload = JSON.parse(mockSocket.sentMessages[0]);
      expect(payload).toEqual({
        type: 'send_message',
        conversation_id: 'conv-1',
        ciphertext: 'cipher-abc',
      });
    });

    it('should include client_message_id when provided', () => {
      service.connect();
      service.sendMessage('conv-1', 'cipher-abc', 'temp-123');
      const payload = JSON.parse(mockSocket.sentMessages[0]);
      expect(payload).toEqual({
        type: 'send_message',
        conversation_id: 'conv-1',
        ciphertext: 'cipher-abc',
        client_message_id: 'temp-123',
      });
    });

    it('should not throw when socket is not open', () => {
      expect(() => service.sendMessage('conv-1', 'cipher')).not.toThrow();
    });

    it('should not send when socket has been disconnected', () => {
      service.connect();
      service.disconnect();
      service.sendMessage('conv-1', 'cipher');
      expect(mockSocket.sentMessages.length).toBe(0);
    });
  });

  describe('disconnect()', () => {
    it('should close the socket', () => {
      service.connect();
      service.disconnect();
      expect(mockSocket.readyState).toBe(MockWebSocket.CLOSED);
    });

    it('should not throw when called before connect()', () => {
      expect(() => service.disconnect()).not.toThrow();
    });
  });

  describe('messages$', () => {
    it('should emit an IncomingMessage when a new_message event is received', () => {
      return new Promise<void>((resolve) => {
        const expected: IncomingMessage = {
          type: 'new_message',
          conversation_id: 'conv-1',
          ciphertext: 'encrypted-blob',
          sender_id: 'alice',
          display_name: 'Alice',
          profile_picture_url: '',
          message_id: 'msg-1',
        };

        service.messages$.subscribe((msg) => {
          expect(msg).toEqual(expected);
          resolve();
        });

        service.connect();
        mockSocket.simulateMessage(expected);
      });
    });

    it('should not emit for messages with a type other than new_message', () => {
      return new Promise<void>((resolve) => {
        let received = false;
        service.messages$.subscribe(() => { received = true; });

        service.connect();
        mockSocket.simulateMessage({ type: 'ping', conversation_id: 'x', ciphertext: '', sender_id: '', display_name: '' });

        setTimeout(() => {
          expect(received).toBe(false);
          resolve();
        }, 0);
      });
    });

    it('should not throw when the incoming message is malformed JSON', () => {
      service.connect();
      expect(() => {
        if (mockSocket.onmessage) {
          mockSocket.onmessage(new MessageEvent('message', { data: 'not-valid-json{{{' }));
        }
      }).not.toThrow();
    });

    it('should emit multiple messages in order', () => {
      return new Promise<void>((resolve) => {
        const received: string[] = [];
        const messages = ['first', 'second', 'third'];

        service.messages$.subscribe((msg) => {
          received.push(msg.ciphertext);
          if (received.length === messages.length) {
            expect(received).toEqual(messages);
            resolve();
          }
        });

        service.connect();
        messages.forEach(c =>
          mockSocket.simulateMessage({ type: 'new_message', conversation_id: 'c', ciphertext: c, sender_id: 'u', display_name: '' })
        );
      });
    });
  });

  describe('ngOnDestroy()', () => {
    it('should disconnect and complete the subject without errors', () => {
      service.connect();
      expect(() => service.ngOnDestroy()).not.toThrow();
      expect(service.isConnected()).toBe(false);
    });
  });
});
