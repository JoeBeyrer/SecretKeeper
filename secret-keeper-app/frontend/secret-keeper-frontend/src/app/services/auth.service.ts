import { Injectable } from '@angular/core';

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
      return data;
    } catch (e) {
      console.error('[AuthService] Failed to load user profile:', e);
      return null;
    }
  }

  getCurrentUser(): UserProfile | null {
    return this.currentUser;
  }

  clearCurrentUser(): void {
    this.currentUser = null;
  }
}