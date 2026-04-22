import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { ConversationService } from './conversation.service';

describe('ConversationService', () => {
  let service: ConversationService;
  let fetchSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(ConversationService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('createConversation()', () => {
    it('should POST /conversations/create with member_ids, room_key, and optional group_name', async () => {
      const mockResponse = { conversation_id: 'conv-abc', created: true };
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockResponse), { status: 200 })
      );
      const result = await service.createConversation(['bob'], 'secret-key', 'Study Group');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/conversations/create');
      expect(opts.method).toBe('POST');
      expect(JSON.parse(opts.body as string)).toEqual({ member_ids: ['bob'], room_key: 'secret-key', group_name: 'Study Group' });
      expect(result).toEqual(mockResponse);
    });

    it('should throw on non-ok response', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('error', { status: 400 }));
      await expect(service.createConversation(['bob'], 'key')).rejects.toThrow();
    });
  });

  describe('getConversations()', () => {
    it('should GET /conversations/get and return list', async () => {
      const mockConvs = [{ id: 'c1', name: 'Bob', last_message: '', last_message_time: 0 }];
      vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockConvs), { status: 200 })
      );
      const result = await service.getConversations();
      expect(result).toEqual(mockConvs);
    });

    it('should throw on error', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('error', { status: 500 }));
      await expect(service.getConversations()).rejects.toThrow();
    });
  });

  describe('getMessages()', () => {
    it('should GET /conversations/:id/messages', async () => {
      const mockMessages = [{ Username: 'alice', Ciphertext: 'abc', CreatedAt: 1000 }];
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockMessages), { status: 200 })
      );
      const result = await service.getMessages('conv-1');
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain('/conversations/conv-1/messages');
      expect(result).toEqual(mockMessages);
    });
  });

  describe('verifyRoomKey()', () => {
    it('should POST /conversations/:id/verify-room-key and resolve on ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await expect(service.verifyRoomKey('conv-1', 'mykey')).resolves.toBeUndefined();
    });

    it('should throw ROOM_KEY_VERIFIER_NOT_SET on 404', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }));
      await expect(service.verifyRoomKey('conv-1', 'mykey')).rejects.toThrow('ROOM_KEY_VERIFIER_NOT_SET');
    });

    it('should throw generic error on other failures', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('bad key', { status: 400 }));
      await expect(service.verifyRoomKey('conv-1', 'mykey')).rejects.toThrow();
    });
  });

  describe('claimRoomKey()', () => {
    it('should POST /conversations/:id/claim-room-key and return the room_key', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify({ room_key: 'claimed-key' }), { status: 200 })
      );
      const key = await service.claimRoomKey('conv-1');
      expect(key).toBe('claimed-key');
    });

    it('should throw ROOM_KEY_NOT_AVAILABLE on 404', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }));
      await expect(service.claimRoomKey('conv-1')).rejects.toThrow('ROOM_KEY_NOT_AVAILABLE');
    });

    it('should throw on other errors', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('forbidden', { status: 403 }));
      await expect(service.claimRoomKey('conv-1')).rejects.toThrow();
    });
  });

  describe('editMessage()', () => {
    it('should PATCH /messages/:id with ciphertext', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 204 }));
      await service.editMessage('msg-1', 'edited-ciphertext');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/messages/msg-1');
      expect(opts.method).toBe('PATCH');
      expect(JSON.parse(opts.body as string)).toEqual({ ciphertext: 'edited-ciphertext' });
    });
  });

  describe('leaveConversation()', () => {
    it('should POST /conversations/:id/leave', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 204 }));
      await service.leaveConversation('conv-1');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/conversations/conv-1/leave');
      expect(opts.method).toBe('POST');
    });

    it('should throw on error', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('error', { status: 500 }));
      await expect(service.leaveConversation('conv-1')).rejects.toThrow();
    });
  });

});
