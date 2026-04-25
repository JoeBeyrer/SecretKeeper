import { Component, OnInit, ChangeDetectorRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

import { FriendService, FriendEntry, UserSearchResult } from '../../services/friend.service';
import { AuthService } from '../../services/auth.service';

type Tab = 'friends' | 'requests' | 'add' | 'search';

const BLOCKED_STORAGE_KEY = 'sk_blocked_ids';

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

  // User search state
  searchQuery: string = '';
  searchResults: UserSearchResult[] = [];
  isSearching: boolean = false;
  searchError: string = '';
  private searchTimer: ReturnType<typeof setTimeout> | null = null;

  private blockedIds: Set<string> = new Set();

  // Confirmation dialog
  confirmDialog: { message: string; onConfirm: () => void } | null = null;

  private confirm(message: string, onConfirm: () => void): void {
    this.confirmDialog = { message, onConfirm };
    this.cdr.detectChanges();
  }

  dismissConfirm(): void {
    this.confirmDialog = null;
    this.cdr.detectChanges();
  }

  runConfirm(): void {
    const action = this.confirmDialog?.onConfirm;
    this.confirmDialog = null;
    this.cdr.detectChanges();
    action?.();
  }

  constructor(
    public friendService: FriendService,
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
    this.loadBlockedIds();
    await this.loadAll(true);
  }

  // ── localStorage helpers ───────────────────────────────────────────────────

  private loadBlockedIds(): void {
    try {
      const raw = localStorage.getItem(BLOCKED_STORAGE_KEY);
      this.blockedIds = new Set(raw ? JSON.parse(raw) : []);
    } catch {
      this.blockedIds = new Set();
    }
  }

  private saveBlockedIds(): void {
    localStorage.setItem(BLOCKED_STORAGE_KEY, JSON.stringify([...this.blockedIds]));
  }

  // ── Data loading ───────────────────────────────────────────────────────────

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
      const incoming = (requests ?? []).filter(r => r.direction === 'incoming');
      this.friendService.pendingCount.set(incoming.length);
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
    if (tab !== 'search') {
      this.searchQuery = '';
      this.searchResults = [];
      this.searchError = '';
    }
    this.loadAll();
  }

  // ── Derived lists ──────────────────────────────────────────────────────────

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

  avatarBg(name: string): string {
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return `hsl(${Math.abs(hash) % 360}, 55%, 38%)`;
  }

  // ── Friend actions ─────────────────────────────────────────────────────────

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

  async rescind(f: FriendEntry): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [f.username]: true };
    this.clearMessages();
    try {
      await this.friendService.rescindRequest(f.username);
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to rescind request.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[f.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  confirmRemove(f: FriendEntry): void {
    this.confirm(`Remove ${this.displayName(f)} as a friend?`, () => this.remove(f));
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

  confirmBlock(f: FriendEntry): void {
    this.confirm(`Block ${this.displayName(f)}? They won't be able to message you.`, () => this.block(f));
  }

  async block(f: FriendEntry): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [f.username]: true };
    this.clearMessages();
    try {
      await this.friendService.blockUser(f.user_id);
      this.blockedIds.add(f.user_id);
      this.saveBlockedIds();
      await this.loadAll();
    } catch (e: any) {
      this.errorMessage = e.message || 'Failed to block user.';
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

  startChat(f: FriendEntry): void {
    this.router.navigate(['/messaging'], { queryParams: { chatWith: f.username } });
  }

  // ── Search ─────────────────────────────────────────────────────────────────

  onSearchInput(): void {
    if (this.searchTimer) clearTimeout(this.searchTimer);
    const q = this.searchQuery.trim();
    if (!q) {
      this.searchResults = [];
      this.searchError = '';
      return;
    }
    this.searchTimer = setTimeout(() => this.runSearch(q), 300);
  }

  private async runSearch(query: string): Promise<void> {
    this.isSearching = true;
    this.searchError = '';
    try {
      const results = await this.friendService.searchUsers(query);
      this.searchResults = results.map(r =>
        this.blockedIds.has(r.user_id) ? { ...r, status: 'blocked' } : r
      );
      this.cdr.detectChanges();
    } catch (e: any) {
      this.searchError = e.message || 'Search failed.';
    } finally {
      this.isSearching = false;
      this.cdr.detectChanges();
    }
  }

  async sendRequestFromSearch(result: UserSearchResult): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [result.username]: true };
    this.searchError = '';
    try {
      await this.friendService.sendFriendRequest(result.username);
      result.status = 'pending_outgoing';
      this.cdr.detectChanges();
    } catch (e: any) {
      this.searchError = e.message || 'Failed to send friend request.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[result.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  confirmBlockFromSearch(result: UserSearchResult): void {
    this.confirm(`Block ${result.display_name || result.username}?`, () => this.blockFromSearch(result));
  }

  async blockFromSearch(result: UserSearchResult): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [result.username]: true };
    this.searchError = '';
    try {
      await this.friendService.blockUser(result.user_id);
      this.blockedIds.add(result.user_id);
      this.saveBlockedIds();
      result.status = 'blocked';
      this.cdr.detectChanges();
    } catch (e: any) {
      this.searchError = e.message || 'Failed to block user.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[result.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  async unblockFromSearch(result: UserSearchResult): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [result.username]: true };
    this.searchError = '';
    try {
      await this.friendService.unblockUser(result.user_id);
      this.blockedIds.delete(result.user_id);
      this.saveBlockedIds();
      result.status = 'none';
      this.cdr.detectChanges();
    } catch (e: any) {
      this.searchError = e.message || 'Failed to unblock user.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[result.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  async rescindFromSearch(result: UserSearchResult): Promise<void> {
    this.actionInProgress = { ...this.actionInProgress, [result.username]: true };
    this.searchError = '';
    try {
      await this.friendService.rescindRequest(result.username);
      result.status = 'none';
      this.cdr.detectChanges();
    } catch (e: any) {
      this.searchError = e.message || 'Failed to cancel request.';
    } finally {
      const u = { ...this.actionInProgress };
      delete u[result.username];
      this.actionInProgress = u;
      this.cdr.detectChanges();
    }
  }

  goTo(page: string): void { this.router.navigate(['/' + page]); }
  goToMessaging(): void { this.router.navigate(['/messaging']); }
  goToProfile(): void { this.router.navigate(['/profile']); }

  clearMessages(): void {
    this.errorMessage = '';
    this.successMessage = '';
  }
}
