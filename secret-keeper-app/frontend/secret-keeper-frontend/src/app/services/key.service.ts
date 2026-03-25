import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class KeyService {
  private base = 'http://localhost:8080/api';

  async saveKeys(publicKey: string, encryptedPrivateKey: string): Promise<void> {
    const res = await fetch(`${this.base}/keys/save`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ public_key: publicKey, encrypted_private_key: encryptedPrivateKey }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async getKeys(): Promise<{ public_key: string; encrypted_private_key: string }> {
    const res = await fetch(`${this.base}/keys/get`, { credentials: 'include' });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }

  async getPublicKey(username: string): Promise<{ public_key: string; user_id: string }> {
    const res = await fetch(`${this.base}/users/${username}/public-key`, { credentials: 'include' });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
    }

  async saveConversationKeys(convId: string, keys: { user_id: string; encrypted_key: string }[]): Promise<void> {
    const res = await fetch(`${this.base}/conversations/${convId}/keys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ keys }),
    });
    if (!res.ok) throw new Error(await res.text());
  }

  async getConversationKey(convId: string): Promise<string> {
    const res = await fetch(`${this.base}/conversations/${convId}/key`, { credentials: 'include' });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    return data.encrypted_key;
  }
}