import { Injectable, signal } from '@angular/core';

export interface UserProfile {
  username: string;
  email: string;
  display_name: string;
  bio: string;
  profile_picture_url: string;
}

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private currentUser: UserProfile | null = null;
  // Reactive signal so components can track profile changes without polling.
  readonly currentUser$ = signal<UserProfile | null>(null);

  // In-memory RSA private key — set at login, cleared at logout.
  private _privateKey: CryptoKey | null = null;
  private _publicKey: CryptoKey | null = null;

  get privateKey(): CryptoKey | null { return this._privateKey; }
  get publicKey(): CryptoKey | null { return this._publicKey; }

  setKeyPair(publicKey: CryptoKey, privateKey: CryptoKey): void {
    this._publicKey = publicKey;
    this._privateKey = privateKey;
    crypto.subtle.exportKey('pkcs8', privateKey).then(pkcs8 => {
      sessionStorage.setItem('sk_private_key', btoa(String.fromCharCode(...new Uint8Array(pkcs8))));
    }).catch(() => {});
    crypto.subtle.exportKey('spki', publicKey).then(spki => {
      sessionStorage.setItem('sk_public_key', btoa(String.fromCharCode(...new Uint8Array(spki))));
    }).catch(() => {});
  }

  async restoreKeyPairFromSession(): Promise<void> {
    if (this._privateKey) return;
    const privB64 = sessionStorage.getItem('sk_private_key');
    const pubB64 = sessionStorage.getItem('sk_public_key');
    if (!privB64 || !pubB64) return;
    try {
      const privBuf = Uint8Array.from(atob(privB64), c => c.charCodeAt(0)).buffer;
      const pubBuf = Uint8Array.from(atob(pubB64), c => c.charCodeAt(0)).buffer;
      const [privateKey, publicKey] = await Promise.all([
        crypto.subtle.importKey('pkcs8', privBuf, { name: 'RSA-OAEP', hash: 'SHA-256' }, true, ['decrypt']),
        crypto.subtle.importKey('spki', pubBuf, { name: 'RSA-OAEP', hash: 'SHA-256' }, true, ['encrypt']),
      ]);
      this._privateKey = privateKey;
      this._publicKey = publicKey;
    } catch (e) {
      console.error('[AuthService] Failed to restore key pair from session:', e);
    }
  }

  async reloadCurrentUser(): Promise<UserProfile | null> {
    this.currentUser = null;
    return this.loadCurrentUser();
  }

  async loadCurrentUser(): Promise<UserProfile | null> {
    if (this.currentUser) {
      return this.currentUser; // already loaded, return cached value
    }

    try {
      const response = await fetch('http://localhost:8080/api/profile', {
        credentials: 'include', // sends sk_session cookie
      });

      if (!response.ok) {
        this.currentUser = null;
        return null;
      }

      const data: UserProfile = await response.json();
      this.currentUser = data;
      this.currentUser$.set(data);
      return data;
    } catch (e) {
      console.error('[AuthService] Failed to load user profile:', e);
      return null;
    }
  }

  getCurrentUser(): UserProfile | null {
    return this.currentUser;
  }

  // Patches cached user and notifies all signal consumers (e.g. nav avatars).
  updateCurrentUser(patch: Partial<UserProfile>): void {
    if (!this.currentUser) return;
    this.currentUser = { ...this.currentUser, ...patch };
    this.currentUser$.set({ ...this.currentUser });
  }

  clearCurrentUser(): void {
    this.currentUser = null;
    this.currentUser$.set(null);
    this._privateKey = null;
    this._publicKey = null;
    sessionStorage.removeItem('sk_private_key');
    sessionStorage.removeItem('sk_public_key');
  }
}
