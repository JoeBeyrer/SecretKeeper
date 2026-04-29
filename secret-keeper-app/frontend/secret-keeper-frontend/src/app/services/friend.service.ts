import { Injectable, signal } from '@angular/core';

export interface FriendEntry {
  user_id: string;
  username: string;
  display_name: string;
  profile_picture_url: string;
  accepted: boolean;
  direction?: string;
}

export interface UserSearchResult {
  user_id: string;
  username: string;
  display_name: string;
  profile_picture_url: string;
  /** "none" | "friend" | "pending_outgoing" | "pending_incoming" | "blocked" */
  status: string;
}

export interface PublicProfile {
  username: string;
  display_name: string;
  bio: string;
  profile_picture_url: string;
  is_friend: boolean;
}

@Injectable({
  providedIn: 'root',
})
export class FriendService {
  private base = 'http://localhost:8080/api';

  readonly pendingCount = signal(0);

  async refreshPendingCount(): Promise<void> {
    try {
      const requests = await this.getPendingRequests();
      const incoming = (requests ?? []).filter(r => r.direction === 'incoming');
      this.pendingCount.set(incoming.length);
    } catch {
      // silently ignore — badge just won't update
    }
  }

  async getFriends(): Promise<FriendEntry[]> {
    const res = await fetch(`${this.base}/friends`, { credentials: 'include' });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  async getPendingRequests(): Promise<FriendEntry[]> {
    const res = await fetch(`${this.base}/friends/requests`, { credentials: 'include' });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  async sendFriendRequest(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async acceptRequest(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/accept`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async declineRequest(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/decline`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async searchUsers(query: string): Promise<UserSearchResult[]> {
    if (!query.trim()) return [];
    const res = await fetch(
      `${this.base}/users/search?q=${encodeURIComponent(query.trim())}`,
      { credentials: 'include' },
    );
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  async rescindRequest(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/rescind`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async removeFriend(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/remove`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async blockUser(blockeeId: string): Promise<void> {
    const res = await fetch(`${this.base}/blocks/block/${blockeeId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ blockee_id: blockeeId }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async unblockUser(blockeeId: string): Promise<void> {
    const res = await fetch(`${this.base}/blocks/unblock/${blockeeId}`, {
      method: 'DELETE',
      credentials: 'include',
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async getPublicProfile(username: string): Promise<PublicProfile> {
    const res = await fetch(`${this.base}/profile/by-username/${encodeURIComponent(username)}`, {
      credentials: 'include',
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  avatarBg(name: string): string {
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return `hsl(${Math.abs(hash) % 360}, 55%, 38%)`;
  }
}
