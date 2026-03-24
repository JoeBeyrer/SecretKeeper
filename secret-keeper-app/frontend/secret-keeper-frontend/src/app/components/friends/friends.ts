import { Component, OnInit, ChangeDetectorRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

import { FriendService, FriendEntry } from '../../services/friend.service';
import { AuthService } from '../../services/auth.service';

type Tab = 'friends' | 'requests' | 'add';

@Component({
  selector: 'app-friends',
  imports: [FormsModule],
  templateUrl: './friends.html',
  styleUrl: './friends.css',
})
export class Friends implements OnInit {
  activeTab: Tab = 'friends';

  friends: FriendEntry[] = [];
  pendingRequests: FriendEntry[] = [];

  addUsername: string = '';
  errorMessage: string = '';
  successMessage: string = '';

  initialLoading: boolean = true;
  refreshing: boolean = false;

  actionInProgress: Record<string, boolean> = {};

  constructor(
    private friendService: FriendService,
    private authService: AuthService,
    private router: Router,
    private cdr: ChangeDetectorRef,
  ) {}

  async ngOnInit(): Promise<void> {
    const user = await this.authService.loadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    await this.loadAll(true);
  }

  async loadAll(initial = false): Promise<void> {
    if (initial) {
      this.initialLoading = true;
    } else {
      this.refreshing = true;
    }

    try {
      const [friends, requests] = await Promise.all([
        this.friendService.getFriends(),
        this.friendService.getPendingRequests(),
      ]);
      this.friends = friends ?? [];
      this.pendingRequests = requests ?? [];
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to load friends.';
    } finally {
      this.initialLoading = false;
      this.refreshing = false;
      this.cdr.detectChanges();
    }
  }

  setTab(tab: Tab): void {
    this.activeTab = tab;
    this.clearMessages();
    this.loadAll();
  }

  get incomingRequests(): FriendEntry[] {
    return this.pendingRequests.filter(r => r.direction === 'incoming');
  }

  get outgoingRequests(): FriendEntry[] {
    return this.pendingRequests.filter(r => r.direction === 'outgoing');
  }

  get incomingCount(): number {
    return this.incomingRequests.length;
  }

  displayName(f: FriendEntry): string {
    return f.display_name || f.username;
  }

  async sendRequest(): Promise<void> {
    const username = this.addUsername.trim();
    if (!username) return;
    this.clearMessages();
    try {
      await this.friendService.sendFriendRequest(username);
      this.successMessage = `Friend request sent to @${username}!`;
      this.addUsername = '';
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to send friend request.';
      this.cdr.detectChanges();
    }
  }

  async accept(f: FriendEntry): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [f.username]: true };
    this.clearMessages();
    try {
      await this.friendService.acceptRequest(f.username);
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to accept request.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[f.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  async decline(f: FriendEntry): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [f.username]: true };
    this.clearMessages();
    try {
      await this.friendService.declineRequest(f.username);
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to decline request.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[f.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  async remove(f: FriendEntry): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [f.username]: true };
    this.clearMessages();
    try {
      await this.friendService.removeFriend(f.username);
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to remove friend.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[f.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  isActing(username: string): boolean {
    return !!this.actionInProgress[username];
  }

  goTo(page: string): void { this.router.navigate(['/' + page]); }
  goToMessaging(): void { this.router.navigate(['/messaging']); }
  goToProfile(): void { this.router.navigate(['/profile']); }

  clearMessages(): void {
    this.errorMessage = '';
    this.successMessage = '';
  }
}
