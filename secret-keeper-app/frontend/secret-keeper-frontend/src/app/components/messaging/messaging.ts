import { Component, NgZone, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewChecked, ChangeDetectorRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { MessagingService } from '../../services/messaging.service';
import { ConversationService } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';
import { CryptoService } from '../../services/crypto.service';

interface MessageAttachment {
  id: string;
  fileName: string;
  mimeType: string;
  size: number;
  downloadUrl: string;
  isImage: boolean;
  isVideo: boolean;
}

interface Message {
  id: string;
  username: string;
  time: string;
  content: string;
  isMine: boolean;
  attachments: MessageAttachment[];
}

interface Conversation {
  id: string;
  name: string;
  lastMessage: string;
  lastMessageTime: string;
  messageLifetime?: number;
}

interface LifetimeOption {
  label: string;
  value: number;
}

interface PendingAttachment {
  id: string;
  file: File;
  fileName: string;
  mimeType: string;
  size: number;
  isImage: boolean;
  isVideo: boolean;
}

interface RichMessageAttachmentPayload {
  file_name: string;
  mime_type: string;
  size: number;
  data_b64: string;
}

interface RichMessagePayload {
  version: 1;
  type: 'rich_message';
  text: string;
  attachments: RichMessageAttachmentPayload[];
}

type ModalState =
  | { type: 'none' }
  | { type: 'create-room-key'; username: string }
  | { type: 'show-room-key'; convId: string; key: string }
  | { type: 'enter-room-key'; convId: string }
  | { type: 'conversation-settings'; convId: string };

@Component({
  selector: 'app-messaging',
  imports: [FormsModule],
  templateUrl: './messaging.html',
  styleUrl: './messaging.css',
})
export class Messaging implements OnInit, OnDestroy, AfterViewChecked {
  readonly lifetimeOptions: LifetimeOption[] = [
    { label: '1 hour', value: 60 },
    { label: '1 day', value: 1440 },
    { label: '1 week', value: 10080 },
    { label: '1 month', value: 43200 },
    { label: '1 year', value: 525600 },
    { label: 'Never', value: 0 },
  ];

  messages: Message[] = [];
  newMessage: string = '';
  errorMessage: string = '';
  composerError: string = '';
  pendingAttachments: PendingAttachment[] = [];
  isSendingMessage: boolean = false;

  conversationId: string = '';
  newConversationMemberId: string = '';
  isConnected: boolean = false;

  currentUsername: string = '';
  currentDisplayName: string = '';
  conversations: Conversation[] = [];
  messageLifetime: number = 0;
  selectedMessageLifetime: number = 0;
  settingsError: string = '';

  modal: ModalState = { type: 'none' };
  roomKeyInput: string = '';
  roomKeyError: string = '';
  roomKeyCopied: boolean = false;

  conversationKeys = new Map<string, CryptoKey>();

  private messageSub: Subscription | null = null;
  private routeQuerySub: Subscription | null = null;
  private shouldScrollToBottom = false;

  @ViewChild('messagesArea') private messagesArea?: ElementRef;
  @ViewChild('attachmentInput') private attachmentInput?: ElementRef<HTMLInputElement>;

  constructor(
    private ngZone: NgZone,
    private cdr: ChangeDetectorRef,
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

    this.routeQuerySub = this.route.queryParamMap.subscribe(params => {
      const chatWith = params.get('chatWith')?.trim();
      if (!chatWith) return;

      this.openCreateConversationModalImmediately(chatWith);

      void this.router.navigate([], {
        relativeTo: this.route,
        queryParams: { chatWith: null },
        queryParamsHandling: 'merge',
        replaceUrl: true,
      });
    });

    await this.refreshConversationList();

    this.messagingService.connect();

    this.messageSub = this.messagingService.messages$.subscribe({ next: async (incoming) => {
      if (incoming.type === 'messages_updated') {
        console.log('[Frontend] messages_updated received for', incoming.conversation_id);
        await this.refreshConversationList();
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
        const msg = this.buildMessageFromDecryptedContent(
          '',
          incoming.display_name || incoming.sender_id,
          this.formatTime(new Date()),
          false,
          plaintext,
        );
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
            attachments: [],
          });
          this.shouldScrollToBottom = true;
        });
      });
    }});
  }

  ngAfterViewChecked(): void {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  async selectConversation(convId: string): Promise<void> {
    if (
      this.conversationId === convId &&
      this.conversationKeys.has(convId) &&
      this.modal.type === 'none'
    ) {
      return;
    }

    this.releaseMessageResources(this.messages);
    this.messages = [];
    this.pendingAttachments = [];
    this.newMessage = '';
    this.composerError = '';
    this.errorMessage = '';
    this.settingsError = '';
    const conv = this.conversations.find(c => c.id === convId);
    this.messageLifetime = conv?.messageLifetime ?? 0;
    this.selectedMessageLifetime = this.messageLifetime;
    if (!this.messagingService.isConnected()) {
      this.messagingService.connect();
    }

    if (this.conversationKeys.has(convId)) {
      this.conversationId = convId;
      this.isConnected = true;
      await this.loadMessages(convId);
      return;
    }

    const claimed = await this.tryClaimRoomKey(convId);
    if (claimed) {
      return;
    }

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
    this.settingsError = '';
  }

  copyRoomKey(): void {
    if (this.modal.type !== 'show-room-key') return;
    navigator.clipboard.writeText(this.modal.key);
    this.roomKeyCopied = true;
    setTimeout(() => this.roomKeyCopied = false, 2000);
  }

  openConversationSettings(): void {
    if (!this.conversationId) return;

    this.selectedMessageLifetime = this.messageLifetime;
    this.settingsError = '';
    this.modal = { type: 'conversation-settings', convId: this.conversationId };
  }

  async saveConversationSettings(): Promise<void> {
    if (this.modal.type !== 'conversation-settings' || !this.conversationId) {
      return;
    }

    try {
      await this.conversationService.setMessageLifetime(this.conversationId, this.selectedMessageLifetime);
      this.messageLifetime = this.selectedMessageLifetime;
      const conv = this.conversations.find(c => c.id === this.conversationId);
      if (conv) {
        conv.messageLifetime = this.selectedMessageLifetime;
      }
      this.modal = { type: 'none' };
      this.settingsError = '';
      await Promise.all([
        this.refreshConversationList(),
        this.loadMessages(this.conversationId),
      ]);
    } catch (e: any) {
      console.error('[Messaging] Failed to save conversation settings:', e);
      this.settingsError = e?.message || 'Failed to update conversation settings.';
    }
  }

  async startNewConversation(): Promise<void> {
    const username = this.newConversationMemberId.trim();
    if (!username) {
      this.errorMessage = 'Please enter a username to start a conversation with.';
      return;
    }

    this.openCreateConversationModal(username);
  }

  triggerAttachmentPicker(): void {
    this.composerError = '';
    this.attachmentInput?.nativeElement.click();
  }

  onAttachmentsSelected(event: Event): void {
    const input = event.target as HTMLInputElement | null;
    const files = Array.from(input?.files ?? []);
    if (files.length === 0) {
      return;
    }

    this.composerError = '';
    this.pendingAttachments = [
      ...this.pendingAttachments,
      ...files.map(file => this.createPendingAttachment(file)),
    ];

    if (input) {
      input.value = '';
    }
  }

  removePendingAttachment(attachmentId: string): void {
    this.pendingAttachments = this.pendingAttachments.filter(attachment => attachment.id !== attachmentId);
  }

  async sendMessage(): Promise<void> {
    const text = this.newMessage.trim();
    const hasAttachments = this.pendingAttachments.length > 0;
    if (!text && !hasAttachments) {
      return;
    }

    const convKey = this.ensureReadyForOutgoingContent();
    if (!convKey) {
      return;
    }

    this.composerError = '';
    this.isSendingMessage = true;

    try {
      let ciphertext: string;
      let optimisticMessage: Message;

      if (hasAttachments) {
        const payload = await this.buildRichMessagePayload(text, this.pendingAttachments);
        ciphertext = await this.cryptoService.encryptMessage(JSON.stringify(payload), convKey);
        optimisticMessage = this.createRichMessageFromFiles(this.currentDisplayName, this.formatTime(new Date()), true, text, this.pendingAttachments);
      } else {
        ciphertext = await this.cryptoService.encryptMessage(text, convKey);
        optimisticMessage = {
          id: '',
          username: this.currentDisplayName,
          time: this.formatTime(new Date()),
          content: text,
          isMine: true,
          attachments: [],
        };
      }

      this.messagingService.sendMessage(this.conversationId, ciphertext);
      this.messages.push(optimisticMessage);
      this.updateConversationPreview(this.conversationId, ciphertext);
      this.newMessage = '';
      this.pendingAttachments = [];
      this.shouldScrollToBottom = true;
    } catch (e) {
      console.error('[Messaging] Failed to encrypt and send outgoing content:', e);
      this.composerError = hasAttachments
        ? 'Failed to encrypt and send file.'
        : 'Failed to encrypt and send message.';
    } finally {
      this.isSendingMessage = false;
    }
  }

  async deleteMessage(messageId: string, index: number): Promise<void> {
    if (!messageId) return;
    try {
      await this.conversationService.DeleteMessage(messageId);
      this.ngZone.run(() => {
        this.releaseMessageResources([this.messages[index]]);
        this.messages.splice(index, 1);
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to delete message:', e);
    }
  }

  getMessageLifetimeLabel(value: number = this.messageLifetime): string {
    return this.lifetimeOptions.find(option => option.value === value)?.label ?? 'Never';
  }

  getAttachmentBadgeLabel(attachment: PendingAttachment): string {
    return `${this.getAttachmentEmoji(attachment.mimeType)} ${attachment.fileName}`;
  }

  getAttachmentMeta(attachment: MessageAttachment): string {
    return `${this.formatFileSize(attachment.size)} • ${attachment.mimeType || 'application/octet-stream'}`;
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
    this.routeQuerySub?.unsubscribe();
    this.releaseMessageResources(this.messages);
  }

  private openCreateConversationModal(username: string): void {
    this.modal = { type: 'create-room-key', username };
    this.roomKeyInput = this.cryptoService.generateRoomKey();
    this.roomKeyError = '';
    this.roomKeyCopied = false;
    this.errorMessage = '';
  }

  private openCreateConversationModalImmediately(username: string): void {
    this.ngZone.run(() => {
      this.openCreateConversationModal(username);
      this.cdr.detectChanges();
    });
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
          messageLifetime: c.message_lifetime ?? 0,
        }));

        const activeConversation = this.conversations.find(c => c.id === this.conversationId);
        if (activeConversation) {
          this.messageLifetime = activeConversation.messageLifetime ?? 0;
          this.selectedMessageLifetime = this.messageLifetime;
        }
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
        this.releaseMessageResources(this.messages);
        this.messages = [];
        this.pendingAttachments = [];
        this.newMessage = '';
        this.composerError = '';
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
            return this.buildMessageFromDecryptedContent(
              m.ID ?? m.id ?? '',
              m.DisplayName || m.Username,
              this.formatTime(new Date(m.CreatedAt * 1000)),
              m.Username === this.currentUsername,
              content,
            );
          } catch {
            return {
              id: m.ID ?? m.id ?? '',
              username: m.DisplayName || m.Username,
              time: this.formatTime(new Date(m.CreatedAt * 1000)),
              content: '🔒 Could not decrypt message',
              isMine: m.Username === this.currentUsername,
              attachments: [],
            };
          }
        })
      );
      this.ngZone.run(() => {
        this.releaseMessageResources(this.messages);
        this.messages = decrypted;
        this.shouldScrollToBottom = true;
      });
    } catch (e) {
      console.error('[Messaging] Failed to load messages:', e);
    }
  }

  private ensureReadyForOutgoingContent(): CryptoKey | null {
    if (!this.conversationId) {
      this.composerError = 'Join or create a conversation first.';
      return null;
    }
    if (!this.messagingService.isConnected()) {
      this.composerError = 'Not connected. Try rejoining the conversation.';
      return null;
    }

    const convKey = this.conversationKeys.get(this.conversationId);
    if (!convKey) {
      this.composerError = 'No room key — please re-enter the room key.';
      return null;
    }

    return convKey;
  }

  private createPendingAttachment(file: File): PendingAttachment {
    return {
      id: this.createUniqueId(),
      file,
      fileName: file.name,
      mimeType: file.type || 'application/octet-stream',
      size: file.size,
      isImage: file.type.startsWith('image/'),
      isVideo: file.type.startsWith('video/'),
    };
  }

  private async buildRichMessagePayload(text: string, attachments: PendingAttachment[]): Promise<RichMessagePayload> {
    const serializedAttachments = await Promise.all(
      attachments.map(async attachment => ({
        file_name: attachment.fileName,
        mime_type: attachment.mimeType,
        size: attachment.size,
        data_b64: this.cryptoService.bytesToBase64(new Uint8Array(await attachment.file.arrayBuffer())),
      }))
    );

    return {
      version: 1,
      type: 'rich_message',
      text,
      attachments: serializedAttachments,
    };
  }

  private buildMessageFromDecryptedContent(id: string, username: string, time: string, isMine: boolean, plaintext: string): Message {
    const payload = this.tryParseRichMessagePayload(plaintext);
    if (!payload) {
      return {
        id,
        username,
        time,
        content: plaintext,
        isMine,
        attachments: [],
      };
    }

    return {
      id,
      username,
      time,
      content: payload.text,
      isMine,
      attachments: payload.attachments.map(attachment => this.createMessageAttachmentFromPayload(attachment)),
    };
  }

  private createRichMessageFromFiles(username: string, time: string, isMine: boolean, text: string, attachments: PendingAttachment[]): Message {
    return {
      id: '',
      username,
      time,
      content: text,
      isMine,
      attachments: attachments.map(attachment => this.createMessageAttachmentFromFile(attachment.file)),
    };
  }

  private tryParseRichMessagePayload(plaintext: string): RichMessagePayload | null {
    try {
      const parsed = JSON.parse(plaintext) as Partial<RichMessagePayload>;
      if (
        parsed?.type !== 'rich_message' ||
        parsed.version !== 1 ||
        typeof parsed.text !== 'string' ||
        !Array.isArray(parsed.attachments)
      ) {
        return null;
      }

      const validAttachments = parsed.attachments.every((attachment: any) => (
        attachment &&
        typeof attachment.file_name === 'string' &&
        typeof attachment.mime_type === 'string' &&
        typeof attachment.size === 'number' &&
        typeof attachment.data_b64 === 'string'
      ));

      if (!validAttachments) {
        return null;
      }

      return parsed as RichMessagePayload;
    } catch {
      return null;
    }
  }

  private createMessageAttachmentFromPayload(payload: RichMessageAttachmentPayload): MessageAttachment {
    const buffer = this.cryptoService.base64ToArrayBuffer(payload.data_b64);
    const blob = new Blob([buffer], { type: payload.mime_type || 'application/octet-stream' });
    const downloadUrl = URL.createObjectURL(blob);
    const mimeType = payload.mime_type || 'application/octet-stream';

    return {
      id: this.createUniqueId(),
      fileName: payload.file_name,
      mimeType,
      size: payload.size,
      downloadUrl,
      isImage: mimeType.startsWith('image/'),
      isVideo: mimeType.startsWith('video/'),
    };
  }

  private createMessageAttachmentFromFile(file: File): MessageAttachment {
    return {
      id: this.createUniqueId(),
      fileName: file.name,
      mimeType: file.type || 'application/octet-stream',
      size: file.size,
      downloadUrl: URL.createObjectURL(file),
      isImage: file.type.startsWith('image/'),
      isVideo: file.type.startsWith('video/'),
    };
  }

  private releaseMessageResources(messages: Message[]): void {
    for (const message of messages) {
      for (const attachment of message.attachments) {
        URL.revokeObjectURL(attachment.downloadUrl);
      }
    }
  }

  private createUniqueId(): string {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
      return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  private addConversationToList(id: string, name: string): void {
    if (this.conversations.find(c => c.id === id)) return;
    this.conversations.unshift({ id, name, lastMessage: '', lastMessageTime: '', messageLifetime: 0 });
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

  private formatFileSize(bytes: number): string {
    if (bytes < 1024) {
      return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`;
    }
    if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    }
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  }

  private getAttachmentEmoji(mimeType: string): string {
    if (mimeType.startsWith('image/')) {
      return '🖼️';
    }
    if (mimeType.startsWith('video/')) {
      return '🎬';
    }
    if (mimeType.includes('pdf')) {
      return '📄';
    }
    if (mimeType.includes('zip') || mimeType.includes('compressed')) {
      return '🗜️';
    }
    if (mimeType.startsWith('audio/')) {
      return '🎵';
    }
    return '📎';
  }
}
