import { Component, OnInit, OnDestroy, ChangeDetectorRef, HostListener, effect } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { FriendService, FriendEntry, UserSearchResult, PublicProfile } from '../../services/friend.service';
import { AuthService } from '../../services/auth.service';
import { MessagingService } from '../../services/messaging.service';

type Tab = 'friends' | 'requests' | 'add' | 'search';

const BLOCKED_STORAGE_KEY = 'sk_blocked_ids';

@Component({
  selector: 'app-friends',
  imports: [FormsModule],
  templateUrl: './friends.html',
  styleUrl: './friends.css',
})
export class Friends implements OnInit, OnDestroy {
  activeTab: Tab = 'friends';

  friends: FriendEntry[] = [];
  pendingRequests: FriendEntry[] = [];

  addUsername: string = '';
  errorMessage: string = '';
  successMessage: string = '';

  initialLoading: boolean = true;
  refreshing: boolean = false;

  actionInProgress: Record<string, boolean> = {};

  // Own profile picture for nav avatar — kept reactive via effect().
  currentUserPictureUrl: string = '';
  currentUsername: string = '';

  // User search state
  searchQuery: string = '';
  searchResults: UserSearchResult[] = [];
  isSearching: boolean = false;
  searchError: string = '';
  private searchTimer: ReturnType<typeof setTimeout> | null = null;

  private blockedIds: Set<string> = new Set();

  // Confirmation dialog
  confirmDialog: { message: string; onConfirm: () => void } | null = null;

  profileModal: PublicProfile | null = null;

  private wsSub: Subscription | null = null;

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
    public router: Router,
    private cdr: ChangeDetectorRef,
    private messagingService: MessagingService,
  ) {
    // Keep nav avatar in sync with any profile change made on this or another tab.
    effect(() => {
      const u = this.authService.currentUser$();
      this.currentUserPictureUrl = u?.profile_picture_url ?? '';
      this.currentUsername = u?.username ?? '';
    });
  }

  async ngOnInit(): Promise<void> {
    const user = await this.authService.loadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    this.loadBlockedIds();
    await this.loadAll(true);

    this.messagingService.connect();
    this.wsSub = this.messagingService.messages$.subscribe(incoming => {
      if (incoming.type !== 'profile_updated') return;
      const username = incoming.username ?? '';
      const pictureURL = incoming.profile_picture_url ?? '';
      const displayName = incoming.display_name ?? '';

      // Update own nav avatar via signal (effect() handles the re-render).
      if (username === this.authService.getCurrentUser()?.username) {
        this.authService.updateCurrentUser({
          profile_picture_url: pictureURL,
          display_name: displayName || undefined,
        });
      }

      // Update friends list: picture AND display name.
      const friend = this.friends.find(f => f.username === username);
      if (friend) {
        friend.profile_picture_url = pictureURL;
        if (displayName) friend.display_name = displayName;
        this.cdr.detectChanges();
      }

      // Update pending requests list.
      const pending = this.pendingRequests.find(f => f.username === username);
      if (pending) {
        pending.profile_picture_url = pictureURL;
        if (displayName) pending.display_name = displayName;
        this.cdr.detectChanges();
      }

      // Update search results.
      const result = this.searchResults.find(r => r.username === username);
      if (result) {
        result.profile_picture_url = pictureURL;
        if (displayName) result.display_name = displayName;
        this.cdr.detectChanges();
      }
    });
  }

  ngOnDestroy(): void {
    this.wsSub?.unsubscribe();
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

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.confirmDialog) {
      this.dismissConfirm();
    } else if (this.profileModal) {
      this.closeProfile();
    }
  }

  displayName(f: FriendEntry): string {
    return f.display_name || f.username;
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

  async openProfile(username: string): Promise<void> {
    try {
      this.profileModal = await this.friendService.getPublicProfile(username);
      this.cdr.detectChanges();
    } catch {
      // silently ignore
    }
  }

  closeProfile(): void {
    this.profileModal = null;
    this.cdr.detectChanges();
  }

  goTo(page: string): void { this.router.navigate(['/' + page]); }
  goToMessaging(): void { this.router.navigate(['/messaging']); }
  goToProfile(): void { this.router.navigate(['/profile']); }

  clearMessages(): void {
    this.errorMessage = '';
    this.successMessage = '';
  }
}
