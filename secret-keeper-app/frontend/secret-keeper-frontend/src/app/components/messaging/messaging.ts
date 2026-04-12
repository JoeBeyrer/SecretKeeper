import { Component, NgZone, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewChecked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { MessagingService } from '../../services/messaging.service';
import { ConversationService } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';
import { CryptoService } from '../../services/crypto.service';

interface Message {
  id: string;
  username: string;
  time: string;
  content: string;
  isMine: boolean;
}

interface Conversation {
  id: string;
  name: string;
  lastMessage: string;
  lastMessageTime: string;
  messageLifetime?:number;
}

type ModalState =
  | { type: 'none' }
  | { type: 'create-room-key'; username: string }
  | { type: 'show-room-key'; convId: string; key: string }
  | { type: 'enter-room-key'; convId: string };

@Component({
  selector: 'app-messaging',
  imports: [FormsModule],
  templateUrl: './messaging.html',
  styleUrl: './messaging.css',
})
export class Messaging implements OnInit, OnDestroy, AfterViewChecked {
  messages: Message[] = [];
  newMessage: string = '';
  errorMessage: string = '';

  conversationId: string = '';
  newConversationMemberId: string = '';
  isConnected: boolean = false;

  currentUsername: string = '';
  currentDisplayName: string = '';
  conversations: Conversation[] = [];
  messageLifetime: number = 0;

  // modal state
  modal: ModalState = { type: 'none' };
  roomKeyInput: string = '';
  roomKeyError: string = '';
  roomKeyCopied: boolean = false;

  // in-memory key cache for the session
  conversationKeys = new Map<string, CryptoKey>();

  private messageSub: Subscription | null = null;
  private shouldScrollToBottom = false;

  @ViewChild('messagesArea') private messagesArea?: ElementRef;

  constructor(
    private ngZone: NgZone,
    private router: Router,
    private route: ActivatedRoute,
    private messagingService: MessagingService,
    private conversationService: ConversationService,
    private authService: AuthService,
    private cryptoService: CryptoService,
  ) {}

  async ngOnInit(): Promise<void> {
    const user = await this.authService.loadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    this.currentUsername = user.username;
    this.currentDisplayName = user.display_name || user.username;

    await this.refreshConversationList();

    this.messagingService.connect();

    this.messageSub = this.messagingService.messages$.subscribe({next: async (incoming) => {
      if (incoming.type === 'messages_updated') {
        console.log('[Frontend] messages_updated received for', incoming.conversation_id);
        if (incoming.conversation_id === this.conversationId) {
          await this.loadMessages(this.conversationId);
        }
        return;
      }
      const knownConversation = !!this.conversations.find(c => c.id === incoming.conversation_id);
      if (!knownConversation) {
        void this.refreshConversationList();
      }

      if (incoming.sender_id !== this.currentUsername) {
        this.updateConversationName(incoming.conversation_id, incoming.display_name || incoming.sender_id);
      }

      this.ngZone.run(() => {
        this.updateConversationPreview(incoming.conversation_id, incoming.ciphertext);
      });

      if (incoming.conversation_id !== this.conversationId) return;
      if (incoming.sender_id === this.currentUsername) return;

      const convKey = this.conversationKeys.get(incoming.conversation_id);
      if (!convKey) return;

      this.cryptoService.decryptMessage(incoming.ciphertext, convKey).then(plaintext => {
        const msg: Message = {
          id: '',
          username: incoming.display_name || incoming.sender_id,
          time: this.formatTime(new Date()),
          content: plaintext,
          isMine: false,
        };
        this.ngZone.run(() => {
          this.messages.push(msg);
          this.shouldScrollToBottom = true;
        });
      }).catch(() => {
        this.ngZone.run(() => {
          this.messages.push({
            id: '',
            username: incoming.display_name || incoming.sender_id,
            time: this.formatTime(new Date()),
            content: '🔒 Could not decrypt message',
            isMine: false,
          });
          this.shouldScrollToBottom = true;
        });
      });
    }});

    const chatWith = this.route.snapshot.queryParamMap.get('chatWith');
    if (chatWith) {
      this.openCreateConversationModal(chatWith);
    }
  }

  ngAfterViewChecked(): void {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  // Called when clicking a conversation in the sidebar
  async selectConversation(convId: string): Promise<void> {
    if (
      this.conversationId === convId &&
      this.conversationKeys.has(convId) &&
      this.modal.type === 'none'
    ) {
      return;
    }

    this.messages = [];
    this.errorMessage = '';
    const conv = this.conversations.find(c => c.id === convId);
    this.messageLifetime = conv?.messageLifetime ?? 0;
    if (!this.messagingService.isConnected()) {
      this.messagingService.connect();
    }

    // If we already have the key cached, open immediately
    if (this.conversationKeys.has(convId)) {
      this.conversationId = convId;
      this.isConnected = true;
      await this.loadMessages(convId);
      return;
    }

    // Try one-time server claim for the recipient
    const claimed = await this.tryClaimRoomKey(convId);
    if (claimed) {
      return;
    }

    // Otherwise prompt for the room key
    this.modal = { type: 'enter-room-key', convId };
    this.roomKeyInput = '';
    this.roomKeyError = '';
  }

  async submitCreateConversation(): Promise<void> {
    if (this.modal.type !== 'create-room-key') return;

    const passphrase = this.roomKeyInput.trim();
    if (passphrase.length <= 6) {
      this.roomKeyError = 'Room key must be longer than 6 characters.';
      return;
    }

    await this.startNewConversationWith(this.modal.username, passphrase);
  }

  // Submit room key from the enter-key modal
  async submitRoomKey(): Promise<void> {
    if (this.modal.type !== 'enter-room-key') return;
    const convId = this.modal.convId;
    const passphrase = this.roomKeyInput.trim();

    if (!passphrase) {
      this.roomKeyError = 'Please enter the room key.';
      return;
    }

    try {
      const key = await this.cryptoService.deriveConversationKey(passphrase, convId);
      const verified = await this.verifyRoomKeyForConversation(convId, passphrase, key);
      if (!verified) {
        return;
      }

      this.conversationKeys.set(convId, key);
      this.conversationId = convId;
      this.isConnected = true;
      this.modal = { type: 'none' };
      this.roomKeyInput = '';
      this.roomKeyError = '';
      await this.loadMessages(convId);
    } catch (e: any) {
      this.roomKeyError = 'Incorrect room key. Please try again.';
    }
  }

  closeModal(): void {
    this.modal = { type: 'none' };
    this.roomKeyInput = '';
    this.roomKeyError = '';
  }

  copyRoomKey(): void {
    if (this.modal.type !== 'show-room-key') return;
    navigator.clipboard.writeText(this.modal.key);
    this.roomKeyCopied = true;
    setTimeout(() => this.roomKeyCopied = false, 2000);
  }

  async startNewConversation(): Promise<void> {
    const username = this.newConversationMemberId.trim();
    if (!username) {
      this.errorMessage = 'Please enter a username to start a conversation with.';
      return;
    }

    this.openCreateConversationModal(username);
  }

  async sendMessage(): Promise<void> {
    if (!this.newMessage.trim()) return;
    if (!this.conversationId) {
      this.errorMessage = 'Join or create a conversation first.';
      return;
    }
    if (!this.messagingService.isConnected()) {
      this.errorMessage = 'Not connected. Try rejoining the conversation.';
      return;
    }

    const convKey = this.conversationKeys.get(this.conversationId);
    if (!convKey) {
      this.errorMessage = 'No room key — please re-enter the room key.';
      return;
    }

    const plaintext = this.newMessage.trim();
    const ciphertext = await this.cryptoService.encryptMessage(plaintext, convKey);

    this.messagingService.sendMessage(this.conversationId, ciphertext);

    this.messages.push({
      id: '',
      username: this.currentDisplayName,
      time: this.formatTime(new Date()),
      content: plaintext,
      isMine: true,
    });
    this.updateConversationPreview(this.conversationId, ciphertext);
    this.newMessage = '';
    this.shouldScrollToBottom = true;
  }

  async deleteMessage(messageId: string, index: number): Promise<void> {
    if (!messageId) return;
    try {
      await this.conversationService.DeleteMessage(messageId);
      this.ngZone.run(() => {
        this.messages.splice(index, 1);
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to delete message:', e);
    }
  }

  async onMessageLifetimeChange(event: Event): Promise<void> {
    console.log('[Lifetime] change event fired');
    const value = Number((event.target as HTMLInputElement).value);
    console.log('[Lifetime] value:', value, 'conversationId:', this.conversationId);
    if (!this.conversationId) return;
    try {
      await this.conversationService.setMessageLifetime(this.conversationId, value);
      this.messageLifetime = value;
    } catch (e: any) {
      console.error('[Messaging] Failed to set message lifetime:', e);
    }
  }

  goTo(page: string): void {
    this.router.navigate(['/' + page]);
  }

  getActiveConversationName(): string {
    const conv = this.conversations.find(c => c.id === this.conversationId);
    return conv ? conv.name : this.conversationId.substring(0, 8);
  }

  ngOnDestroy(): void {
    this.messageSub?.unsubscribe();
  }

  // Private helpers

  private openCreateConversationModal(username: string): void {
    this.modal = { type: 'create-room-key', username };
    this.roomKeyInput = this.cryptoService.generateRoomKey();
    this.roomKeyError = '';
    this.roomKeyCopied = false;
    this.errorMessage = '';
  }

  private async refreshConversationList(): Promise<void> {
    try {
      const convs = await this.conversationService.getConversations();
      this.ngZone.run(() => {
        this.conversations = convs.map(c => ({
          id: c.id,
          name: c.name,
          lastMessage: c.last_message ? '🔒 Encrypted message' : '',
          lastMessageTime: c.last_message_time
            ? this.formatTimeShort(new Date(c.last_message_time * 1000))
            : '',
          message_lifetime: c.message_lifetime ?? 0,
        }));
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to load conversations:', e);
    }
  }

  private async tryClaimRoomKey(convId: string): Promise<boolean> {
    try {
      const roomKey = await this.conversationService.claimRoomKey(convId);
      const key = await this.cryptoService.deriveConversationKey(roomKey, convId);
      this.conversationKeys.set(convId, key);

      this.ngZone.run(() => {
        this.conversationId = convId;
        this.isConnected = true;
        this.modal = { type: 'show-room-key', convId, key: roomKey };
        this.roomKeyCopied = false;
        this.roomKeyInput = '';
        this.roomKeyError = '';
      });

      await this.loadMessages(convId);
      return true;
    } catch (e: any) {
      if (e?.message === 'ROOM_KEY_NOT_AVAILABLE') {
        return false;
      }

      console.error('[Messaging] Failed to claim room key:', e);
      this.ngZone.run(() => {
        this.errorMessage = 'Failed to retrieve the one-time room key.';
      });
      return false;
    }
  }

  private async startNewConversationWith(username: string, passphrase: string): Promise<void> {
    try {
      const result = await this.conversationService.createConversation([username], passphrase);

      this.ngZone.run(() => {
        this.conversationId = result.conversation_id;
        this.messages = [];
        this.errorMessage = '';
        this.isConnected = true;
        this.newConversationMemberId = '';
        this.addConversationToList(result.conversation_id, username);
      });

      if (result.created) {
        const key = await this.cryptoService.deriveConversationKey(passphrase, result.conversation_id);
        this.conversationKeys.set(result.conversation_id, key);

        this.ngZone.run(() => {
          this.modal = { type: 'show-room-key', convId: result.conversation_id, key: passphrase };
          this.roomKeyCopied = false;
        });
        return;
      }

      if (this.conversationKeys.has(result.conversation_id)) {
        await this.loadMessages(result.conversation_id);
        return;
      }

      this.ngZone.run(() => {
        this.modal = { type: 'enter-room-key', convId: result.conversation_id };
        this.roomKeyInput = '';
        this.roomKeyError = '';
      });
    } catch (e: any) {
      this.ngZone.run(() => {
        this.errorMessage = e.message || 'Failed to create conversation.';
      });
    }
  }

  private async verifyRoomKeyForConversation(convId: string, passphrase: string, key: CryptoKey): Promise<boolean> {
    try {
      await this.conversationService.verifyRoomKey(convId, passphrase);
      return true;
    } catch (e: any) {
      if (e?.message !== 'ROOM_KEY_VERIFIER_NOT_SET') {
        this.roomKeyError = 'Incorrect room key. Please try again.';
        return false;
      }
    }

    // Backward compatibility for older conversations created before verifier support.
    const history = await this.conversationService.getMessages(convId);
    if (history.length === 0) {
      this.roomKeyError = 'This older conversation has no saved room-key verifier yet. Please create a new conversation.';
      return false;
    }

    try {
      await this.cryptoService.decryptMessage(history[history.length - 1].Ciphertext, key);
      return true;
    } catch {
      this.roomKeyError = 'Incorrect room key. Please try again.';
      return false;
    }
  }

  private async loadMessages(convId: string): Promise<void> {
    const convKey = this.conversationKeys.get(convId);
    if (!convKey) return;

    try {
      const history = await this.conversationService.getMessages(convId);
      const decrypted: Message[] = await Promise.all(
        history.map(async (m: any) => {
          try {
            const content = await this.cryptoService.decryptMessage(m.Ciphertext, convKey);
            return {
              id: m.ID,
              username: m.DisplayName || m.Username,
              time: this.formatTime(new Date(m.CreatedAt * 1000)),
              content,
              isMine: m.Username === this.currentUsername,
            };
          } catch {
            return {
              id: m.ID,
              username: m.DisplayName || m.Username,
              time: this.formatTime(new Date(m.CreatedAt * 1000)),
              content: '🔒 Could not decrypt message',
              isMine: m.Username === this.currentUsername,
            };
          }
        })
      );
      this.ngZone.run(() => {
        this.messages = decrypted;
        this.shouldScrollToBottom = true;
      });
    } catch (e) {
      console.error('[Messaging] Failed to load messages:', e);
    }
  }

  private addConversationToList(id: string, name: string): void {
    if (this.conversations.find(c => c.id === id)) return;
    this.conversations.unshift({ id, name, lastMessage: '', lastMessageTime: '' });
  }

  private updateConversationName(convId: string, displayName: string): void {
    const conv = this.conversations.find(c => c.id === convId);
    if (conv && conv.name === convId.substring(0, 8)) {
      conv.name = displayName;
    }
  }

  private updateConversationPreview(convId: string, _ciphertext: string): void {
    const conv = this.conversations.find(c => c.id === convId);
    if (conv) {
      conv.lastMessage = '🔒 Encrypted message';
      conv.lastMessageTime = this.formatTimeShort(new Date());
    }
  }

  private scrollToBottom(): void {
    if (this.messagesArea) {
      const el = this.messagesArea.nativeElement;
      el.scrollTop = el.scrollHeight;
    }
  }

  private formatTime(d: Date): string {
    const hours = d.getHours();
    const minutes = String(d.getMinutes()).padStart(2, '0');
    const ampm = hours >= 12 ? 'pm' : 'am';
    const h = hours % 12 || 12;
    return `Today, ${h}.${minutes}${ampm}`;
  }

  private formatTimeShort(d: Date): string {
    const hours = d.getHours();
    const minutes = String(d.getMinutes()).padStart(2, '0');
    const ampm = hours >= 12 ? 'pm' : 'am';
    const h = hours % 12 || 12;
    return `${h}:${minutes}${ampm}`;
  }
}
