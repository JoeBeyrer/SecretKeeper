import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { KeyService } from './key.service';

describe('KeyService', () => {
  let service: KeyService;
  let fetchSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(KeyService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('saveKeys()', () => {
    it('should POST /keys/save with the correct payload', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await service.saveKeys('pub-key-abc', 'enc-priv-key-xyz');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/keys/save');
      expect(opts.method).toBe('POST');
      expect(JSON.parse(opts.body as string)).toEqual({
        public_key: 'pub-key-abc',
        encrypted_private_key: 'enc-priv-key-xyz',
      });
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('error', { status: 500 }));
      await expect(service.saveKeys('pub', 'priv')).rejects.toThrow();
    });
  });

  describe('getKeys()', () => {
    it('should GET /keys/get and return public and private keys', async () => {
      const mockKeys = { public_key: 'pub-key', encrypted_private_key: 'enc-priv' };
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockKeys), { status: 200 })
      );
      const result = await service.getKeys();
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain('/keys/get');
      expect(result).toEqual(mockKeys);
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }));
      await expect(service.getKeys()).rejects.toThrow();
    });
  });

  describe('getPublicKey()', () => {
    it('should GET /users/:username/public-key and return public_key and user_id', async () => {
      const mockData = { public_key: 'pub-key', user_id: 'user-uuid-123' };
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockData), { status: 200 })
      );
      const result = await service.getPublicKey('alice');
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain('/users/alice/public-key');
      expect(result).toEqual(mockData);
    });

    it('should throw when the user is not found', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }));
      await expect(service.getPublicKey('ghost')).rejects.toThrow();
    });
  });

  describe('saveConversationKeys()', () => {
    it('should POST /conversations/:id/keys with the keys array', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      const keys = [{ user_id: 'u1', encrypted_key: 'enc1' }, { user_id: 'u2', encrypted_key: 'enc2' }];
      await service.saveConversationKeys('conv-abc', keys);
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/conversations/conv-abc/keys');
      expect(opts.method).toBe('POST');
      expect(JSON.parse(opts.body as string)).toEqual({ keys });
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('forbidden', { status: 403 }));
      await expect(service.saveConversationKeys('conv-1', [])).rejects.toThrow();
    });
  });

  describe('getConversationKey()', () => {
    it('should GET /conversations/:id/key and return the encrypted_key string', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify({ encrypted_key: 'my-enc-key' }), { status: 200 })
      );
      const result = await service.getConversationKey('conv-xyz');
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain('/conversations/conv-xyz/key');
      expect(result).toBe('my-enc-key');
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }));
      await expect(service.getConversationKey('conv-1')).rejects.toThrow();
    });
  });
});
