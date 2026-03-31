import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { FriendService } from './friend.service';

describe('FriendService', () => {
  let service: FriendService;
  let fetchSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(FriendService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getFriends()', () => {
    it('should GET /friends and return the list', async () => {
      const mockFriends = [{ user_id: 'u1', username: 'bob', display_name: '', accepted: true }];
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockFriends), { status: 200 })
      );
      const result = await service.getFriends();
      expect(result).toEqual(mockFriends);
      expect(fetchSpy.mock.calls[0][0]).toContain('/friends');
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('Forbidden', { status: 403 }));
      await expect(service.getFriends()).rejects.toThrow();
    });
  });

  describe('getPendingRequests()', () => {
    it('should GET /friends/requests and return list', async () => {
      const mockRequests = [{ user_id: 'u2', username: 'carol', display_name: '', accepted: false, direction: 'incoming' }];
      vi.spyOn(window, 'fetch').mockResolvedValue(
        new Response(JSON.stringify(mockRequests), { status: 200 })
      );
      const result = await service.getPendingRequests();
      expect(result).toEqual(mockRequests);
    });
  });

  describe('sendFriendRequest()', () => {
    it('should POST /friends/request with correct username', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await service.sendFriendRequest('dave');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/friends/request');
      expect(opts.method).toBe('POST');
      expect(JSON.parse(opts.body as string)).toEqual({ username: 'dave' });
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('Not found', { status: 404 }));
      await expect(service.sendFriendRequest('ghost')).rejects.toThrow();
    });
  });

  describe('acceptRequest()', () => {
    it('should POST /friends/accept with correct username', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await service.acceptRequest('carol');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/friends/accept');
      expect(JSON.parse(opts.body as string)).toEqual({ username: 'carol' });
    });
  });

  describe('declineRequest()', () => {
    it('should POST /friends/decline with correct username', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await service.declineRequest('carol');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/friends/decline');
      expect(JSON.parse(opts.body as string)).toEqual({ username: 'carol' });
    });
  });

  describe('removeFriend()', () => {
    it('should DELETE /friends/remove with correct username', async () => {
      fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(new Response('', { status: 200 }));
      await service.removeFriend('bob');
      const [url, opts] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toContain('/friends/remove');
      expect(opts.method).toBe('DELETE');
      expect(JSON.parse(opts.body as string)).toEqual({ username: 'bob' });
    });

    it('should throw when response is not ok', async () => {
      vi.spyOn(window, 'fetch').mockResolvedValue(new Response('error', { status: 500 }));
      await expect(service.removeFriend('bob')).rejects.toThrow();
    });
  });
});
