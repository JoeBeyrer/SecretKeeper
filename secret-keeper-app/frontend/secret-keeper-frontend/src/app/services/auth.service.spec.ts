import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { AuthService } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(AuthService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('getCurrentUser() should return null before any load', () => {
    expect(service.getCurrentUser()).toBeNull();
  });

  it('clearCurrentUser() should set currentUser back to null', async () => {
    const mockProfile = {
      username: 'alice', email: 'alice@example.com',
      display_name: 'Alice', bio: '', profile_picture_url: '',
    };
    vi.spyOn(window, 'fetch').mockResolvedValue(new Response(JSON.stringify(mockProfile), { status: 200 }));

    await service.loadCurrentUser();
    expect(service.getCurrentUser()).toBeTruthy();

    service.clearCurrentUser();
    expect(service.getCurrentUser()).toBeNull();
  });

  it('loadCurrentUser() should return cached user on second call without re-fetching', async () => {
    const mockProfile = {
      username: 'alice', email: 'alice@example.com',
      display_name: 'Alice', bio: '', profile_picture_url: '',
    };
    const fetchSpy = vi.spyOn(window, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(mockProfile), { status: 200 })
    );

    await service.loadCurrentUser();
    await service.loadCurrentUser();

    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it('loadCurrentUser() should return null when response is not ok', async () => {
    vi.spyOn(window, 'fetch').mockResolvedValue(new Response('Unauthorized', { status: 401 }));
    const user = await service.loadCurrentUser();
    expect(user).toBeNull();
  });

  it('loadCurrentUser() should return null when fetch throws', async () => {
    vi.spyOn(window, 'fetch').mockRejectedValue(new Error('Network error'));
    const user = await service.loadCurrentUser();
    expect(user).toBeNull();
  });

  it('loadCurrentUser() should return and cache the user profile on success', async () => {
    const mockProfile = {
      username: 'bob', email: 'bob@example.com',
      display_name: 'Bob', bio: 'Hi', profile_picture_url: 'http://img.url',
    };
    vi.spyOn(window, 'fetch').mockResolvedValue(new Response(JSON.stringify(mockProfile), { status: 200 }));

    const user = await service.loadCurrentUser();
    expect(user).toEqual(mockProfile);
    expect(service.getCurrentUser()).toEqual(mockProfile);
  });
});
