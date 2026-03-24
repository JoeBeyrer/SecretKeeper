import { Injectable } from '@angular/core';

export interface FriendEntry {
  user_id: string;
  username: string;
  display_name: string;
  accepted: boolean;
  direction?: string; 
}

@Injectable({
  providedIn: 'root',
})
export class FriendService {
  private base = 'http://localhost:8080/api';

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

  async removeFriend(username: string): Promise<void> {
    const res = await fetch(`${this.base}/friends/remove`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ username }),
    });
    if (!res.ok) throw new Error(await res.text());
  }
}