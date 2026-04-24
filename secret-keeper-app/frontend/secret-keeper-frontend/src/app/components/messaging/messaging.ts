import { Component, NgZone, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewChecked, ChangeDetectorRef, HostListener } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { MessagingService } from '../../services/messaging.service';
import { ConversationService, ConversationMemberSummary } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';
import { CryptoService } from '../../services/crypto.service';
import { FriendService, FriendEntry } from '../../services/friend.service';

interface ReactionUser {
  username: string;
  displayName: string;
}

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
  isSystem: boolean;
  attachments: MessageAttachment[];
  profilePictureUrl: string;
  expiresAt?: number;
}

interface Conversation {
  id: string;
  name: string;
  fullName: string;
  lastMessage: string;
  lastMessageTime: string;
  messageLifetime?: number;
  memberCount: number;
}

interface ConversationMember {
  userId: string;
  username: string;
  displayName: string;
  profilePictureUrl: string;
  friendshipStatus: ConversationMemberSummary['friendship_status'];
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
  | { type: 'select-conversation-members' }
  | { type: 'create-room-key'; memberUsernames: string[] }
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
  readonly QUICK_REACTIONS = ['❤️', '👍', '😂', '😮', '😢', '🙏'];

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
  isConnected: boolean = false;

  availableFriends: FriendEntry[] = [];
  selectedConversationMembers: string[] = [];
  isLoadingFriends: boolean = false;
  createConversationError: string = '';
  groupConversationNameInput: string = '';
  memberPickerMode: 'create' | 'add' = 'create';

  currentUsername: string = '';
  currentDisplayName: string = '';
  currentUserPictureUrl: string = '';
  activeConversationPictureUrl: string = '';
  conversations: Conversation[] = [];
  conversationMembers: ConversationMember[] = [];
  isLoadingConversationMembers: boolean = false;
  memberActionInProgress: Record<string, boolean> = {};
  pendingRemovedMemberIds: string[] = [];
  manageError: string = '';
  messageLifetime: number = 0;
  selectedMessageLifetime: number = 0;
  settingsError: string = '';
  editedGroupName: string = '';
  openMessageMenuId: string | null = null;
  editingMessageId: string | null = null;
  editDraft: string = '';
  editError: string = '';
  isSavingEdit: boolean = false;

  modal: ModalState = { type: 'none' };
  roomKeyInput: string = '';
  roomKeyError: string = '';
  roomKeyCopied: boolean = false;

  reactionPickerMessageId: string | null = null;
  // messageId → emoji → reaction users
  messageReactions = new Map<string, Map<string, ReactionUser[]>>();

  conversationKeys = new Map<string, CryptoKey>();
  conversationPassphrases = new Map<string, string>();

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
    public friendService: FriendService,
  ) {}

  async ngOnInit(): Promise<void> {
    const user = await this.authService.reloadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    this.currentUsername = user.username;
    this.currentDisplayName = user.display_name || user.username;
    this.currentUserPictureUrl = user.profile_picture_url || '';
    this.friendService.refreshPendingCount();

    this.routeQuerySub = this.route.queryParamMap.subscribe(params => {
      const chatWith = params.get('chatWith')?.trim();
      if (!chatWith) return;

      this.openCreateConversationModalImmediately([chatWith]);

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
      if (incoming.type === 'profile_updated') {
        this.ngZone.run(() => {
          if (incoming.username === this.currentUsername) {
            // Update own picture on this tab (sent from account page on another tab).
            this.currentUserPictureUrl = incoming.profile_picture_url ?? '';
            for (const msg of this.messages) {
              if (msg.isMine) {
                msg.profilePictureUrl = incoming.profile_picture_url ?? '';
              }
            }
          } else {
            for (const msg of this.messages) {
              if (!msg.isMine && msg.username === incoming.username) {
                msg.profilePictureUrl = incoming.profile_picture_url ?? '';
              }
            }
            // Fix: update header picture even when there are zero messages from
            // the other participant (previously stayed as the SVG placeholder).
            const conv = this.conversations.find(c => c.id === this.conversationId);
            if (conv && conv.name === incoming.username) {
              this.activeConversationPictureUrl = incoming.profile_picture_url ?? '';
            }
          }
        });
        return;
      }

      if (incoming.type === 'message_ack') {
        this.ngZone.run(() => {
          this.applyMessageAck(incoming.message_id, incoming.client_message_id);
        });
        return;
      }

      if (incoming.type === 'messages_updated') {
        console.log('[Frontend] messages_updated received for', incoming.conversation_id);
      await this.ngZone.run(async () => {
        await this.refreshConversationList();
        if (incoming.conversation_id === this.conversationId) {
          if (!this.conversations.some(conversation => conversation.id === this.conversationId)) {
            this.resetActiveConversationState(this.conversationId);
            return;
          }
          this.cancelEditingMessage(false);
          await this.loadMessages(this.conversationId, false);
        }
      });
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
      // When the server echoes our own sent message back, use it to update
      // the optimistic message with the real expiresAt from the server.
      if (incoming.sender_id === this.currentUsername) {
        if (incoming.expires_at !== undefined && incoming.message_id) {
          this.ngZone.run(() => {
            const optimistic = this.messages.find(m => m.id === incoming.message_id);
            if (optimistic) {
              optimistic.expiresAt = incoming.expires_at;
            }
          });
        }
        return;
      }

      const convKey = this.conversationKeys.get(incoming.conversation_id);
      if (!convKey) return;

      this.cryptoService.decryptMessage(incoming.ciphertext, convKey).then(plaintext => {
        const msg = this.buildMessageFromDecryptedContent(
          incoming.message_id ?? '',
          incoming.display_name || incoming.sender_id,
          this.formatTime(new Date()),
          false,
          plaintext,
          incoming.profile_picture_url ?? '',
          // Pass the expiry from the WS broadcast so the label shows immediately
          // for real-time messages, matching the label shown in loaded history.
          incoming.expires_at ?? undefined
        );
        this.ngZone.run(() => {
          this.messages.push(msg);
          this.shouldScrollToBottom = true;
	  if (!this.activeConversationPictureUrl && msg.profilePictureUrl) {
            this.activeConversationPictureUrl = msg.profilePictureUrl;
	  }
        });
      }).catch(() => {
        this.ngZone.run(() => {
          this.messages.push({
            id: '',
            username: incoming.display_name || incoming.sender_id,
            time: this.formatTime(new Date()),
            content: '🔒 Could not decrypt message',
            isMine: false,
            isSystem: false,
            attachments: [],
            profilePictureUrl: incoming.profile_picture_url ?? '',
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
    this.manageError = '';
    this.conversationMembers = [];
    this.memberActionInProgress = {};
    this.cancelEditingMessage(false);
    this.openMessageMenuId = null;
    this.activeConversationPictureUrl = '';
    this.stopPictureRefresh();
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

    await this.startNewConversationWith(this.modal.memberUsernames, passphrase, this.groupConversationNameInput);
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
      this.conversationPassphrases.set(convId, passphrase);
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
    this.manageError = '';
    this.isLoadingConversationMembers = false;
    this.conversationMembers = [];
    this.memberActionInProgress = {};
    this.pendingRemovedMemberIds = [];
    this.createConversationError = '';
    this.selectedConversationMembers = [];
    this.groupConversationNameInput = '';
    this.memberPickerMode = 'create';
    this.editedGroupName = '';
  }

  copyRoomKey(): void {
    if (this.modal.type !== 'show-room-key') return;
    navigator.clipboard.writeText(this.modal.key);
    this.roomKeyCopied = true;
    setTimeout(() => this.roomKeyCopied = false, 2000);
  }

  async openConversationSettings(): Promise<void> {
    if (!this.conversationId) return;

    this.selectedMessageLifetime = this.messageLifetime;
    this.editedGroupName = this.conversations.find(c => c.id === this.conversationId)?.fullName ?? '';
    this.settingsError = '';
    this.manageError = '';
    this.memberActionInProgress = {};
    this.pendingRemovedMemberIds = [];
    this.modal = { type: 'conversation-settings', convId: this.conversationId };

    if (!this.isActiveConversationGroup()) {
      this.conversationMembers = [];
      this.isLoadingConversationMembers = false;
      return;
    }

    this.isLoadingConversationMembers = true;
    this.conversationMembers = [];

    try {
      const members = await this.conversationService.getConversationMembers(this.conversationId);
      this.ngZone.run(() => {
        this.conversationMembers = members.map(member => ({
          userId: member.user_id,
          username: member.username,
          displayName: member.display_name || member.username,
          profilePictureUrl: member.profile_picture_url || '',
          friendshipStatus: member.friendship_status,
        }));
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to load conversation members:', e);
      this.ngZone.run(() => {
        this.manageError = e?.message || 'Failed to load group members.';
      });
    } finally {
      this.ngZone.run(() => {
        this.isLoadingConversationMembers = false;
      });
    }
  }

  async saveConversationSettings(): Promise<void> {
    if (this.modal.type !== 'conversation-settings' || !this.conversationId) {
      return;
    }

    const conv = this.conversations.find(c => c.id === this.conversationId);
    const nextGroupName = this.editedGroupName.trim();
    const currentGroupName = (conv?.fullName ?? '').trim();
    const currentMemberCount = conv?.memberCount ?? this.conversationMembers.length;
    const membersToRemove = [...this.pendingRemovedMemberIds];
    const remainingMembersAfterRemoval = Math.max(currentMemberCount - membersToRemove.length, 0);
    const shouldRemoveMembers = membersToRemove.length > 0;
    const shouldRenameGroup = this.isActiveConversationGroup() && remainingMembersAfterRemoval > 2 && nextGroupName !== currentGroupName;
    const shouldUpdateLifetime = this.selectedMessageLifetime !== this.messageLifetime;

    if (shouldRemoveMembers && remainingMembersAfterRemoval < 2) {
      this.settingsError = 'A conversation must keep at least two members.';
      return;
    }

    if (this.isActiveConversationGroup() && remainingMembersAfterRemoval > 2 && !nextGroupName) {
      this.settingsError = 'Group name is required.';
      return;
    }

    try {
      if (shouldUpdateLifetime) {
        await this.conversationService.setMessageLifetime(this.conversationId, this.selectedMessageLifetime);
        this.messageLifetime = this.selectedMessageLifetime;
        if (conv) {
          conv.messageLifetime = this.selectedMessageLifetime;
        }
      }

      if (shouldRemoveMembers) {
        await this.conversationService.removeConversationMembers(this.conversationId, membersToRemove);
        if (conv) {
          conv.memberCount = remainingMembersAfterRemoval;
        }
      }

      if (shouldRenameGroup) {
        await this.conversationService.updateGroupName(this.conversationId, nextGroupName);
        if (conv) {
          conv.fullName = nextGroupName;
          conv.name = this.formatConversationName(nextGroupName);
        }
      }

      this.modal = { type: 'none' };
      this.settingsError = '';
      this.manageError = '';
      this.editedGroupName = '';
      this.pendingRemovedMemberIds = [];

      if (shouldRenameGroup || shouldUpdateLifetime || shouldRemoveMembers) {
        await Promise.all([
          this.refreshConversationList(),
          this.loadMessages(this.conversationId),
        ]);
      }
    } catch (e: any) {
      console.error('[Messaging] Failed to save conversation settings:', e);
      this.settingsError = e?.message || 'Failed to update conversation settings.';
    }
  }

  async leaveConversation(): Promise<void> {
    const convId = this.conversationId;
    if (!convId) {
      return;
    }

    try {
      await this.conversationService.leaveConversation(convId);
      this.ngZone.run(() => {
        this.conversations = this.conversations.filter(conversation => conversation.id !== convId);
        this.resetActiveConversationState(convId);
        this.errorMessage = '';
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to leave conversation:', e);
      this.errorMessage = e?.message || 'Failed to leave conversation.';
    }
  }

  async startNewConversation(): Promise<void> {
    await this.openCreateConversationMemberModal();
  }

  async openAddConversationMembersModal(): Promise<void> {
    if (!this.conversationId || !this.isActiveConversationGroup()) {
      return;
    }

    await this.openCreateConversationMemberModal([], 'add');
  }

  toggleConversationMember(username: string): void {
    const normalizedUsername = username.trim();
    if (!normalizedUsername) {
      return;
    }

    this.createConversationError = '';

    if (this.selectedConversationMembers.includes(normalizedUsername)) {
      this.selectedConversationMembers = this.selectedConversationMembers.filter(member => member !== normalizedUsername);
      if (this.selectedConversationMembers.length <= 1) {
        this.groupConversationNameInput = '';
      }
      return;
    }

    this.selectedConversationMembers = [...this.selectedConversationMembers, normalizedUsername];
  }

  continueCreateConversation(): void {
    if (this.selectedConversationMembers.length === 0) {
      this.createConversationError = 'Please select at least one friend.';
      return;
    }

    if (this.memberPickerMode === 'add') {
      void this.addSelectedConversationMembers();
      return;
    }

    this.openCreateConversationModal(this.selectedConversationMembers);
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
      const tempMessageId = this.createTempMessageId();
      let ciphertext: string;
      let optimisticMessage: Message;

      if (hasAttachments) {
        const payload = await this.buildRichMessagePayload(text, this.pendingAttachments);
        ciphertext = await this.cryptoService.encryptMessage(JSON.stringify(payload), convKey);
        optimisticMessage = this.createRichMessageFromFiles(tempMessageId, this.currentDisplayName, this.formatTime(new Date()), true, text, this.pendingAttachments);
        // Same local expiresAt estimate for rich messages.
        if (this.messageLifetime > 0) {
          optimisticMessage.expiresAt = Math.floor(Date.now() / 1000) + this.messageLifetime * 60;
        }
      } else {
        ciphertext = await this.cryptoService.encryptMessage(text, convKey);
        optimisticMessage = {
          id: tempMessageId,
          username: this.currentDisplayName,
          time: this.formatTime(new Date()),
          content: text,
          isMine: true,
          isSystem: false,
          attachments: [],
          profilePictureUrl: this.currentUserPictureUrl,
          // Estimate expiresAt locally so the expiry label appears immediately.
          // The server echo will overwrite this with the authoritative value.
          expiresAt: this.messageLifetime > 0
            ? Math.floor(Date.now() / 1000) + this.messageLifetime * 60
            : undefined,
        };
      }

      this.messagingService.sendMessage(this.conversationId, ciphertext, tempMessageId);
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

  toggleMessageMenu(messageId: string): void {
    this.openMessageMenuId = this.openMessageMenuId === messageId ? null : messageId;
  }

  startEditingMessage(message: Message): void {
    if (!message.isMine || !message.id || message.id.startsWith('temp-')) {
      return;
    }

    this.editingMessageId = message.id;
    this.editDraft = message.content;
    this.editError = '';
    this.openMessageMenuId = null;
  }

  cancelEditingMessage(clearMenu: boolean = true): void {
    this.editingMessageId = null;
    this.editDraft = '';
    this.editError = '';
    this.isSavingEdit = false;
    if (clearMenu) {
      this.openMessageMenuId = null;
    }
  }

  async saveEditedMessage(message: Message): Promise<void> {
    if (!message.isMine || !message.id || message.id.startsWith('temp-')) {
      this.editError = 'Message is not ready to edit yet.';
      return;
    }

    const convKey = this.ensureReadyForOutgoingContent();
    if (!convKey) {
      this.editError = this.composerError;
      return;
    }

    const trimmedDraft = this.editDraft.trim();
    if (!trimmedDraft && message.attachments.length === 0) {
      this.editError = 'Message cannot be empty.';
      return;
    }

    this.isSavingEdit = true;
    this.editError = '';

    try {
      let ciphertext: string;
      if (message.attachments.length > 0) {
        const payload = await this.buildRichMessagePayloadFromExistingAttachments(trimmedDraft, message.attachments);
        ciphertext = await this.cryptoService.encryptMessage(JSON.stringify(payload), convKey);
      } else {
        ciphertext = await this.cryptoService.encryptMessage(trimmedDraft, convKey);
      }

      await this.conversationService.editMessage(message.id, ciphertext);
      message.content = trimmedDraft;
      this.cancelEditingMessage();
    } catch (e) {
      console.error('[Messaging] Failed to edit message:', e);
      this.editError = 'Failed to save message changes.';
    } finally {
      this.isSavingEdit = false;
    }
  }

  onEditDraftKeydown(event: KeyboardEvent, message: Message): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      void this.saveEditedMessage(message);
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      this.cancelEditingMessage();
    }
  }

  async deleteMessage(messageId: string, index: number): Promise<void> {
    if (!messageId) return;
    try {
      await this.conversationService.deleteMessage(messageId);
      this.ngZone.run(() => {
        if (this.editingMessageId === messageId) {
          this.cancelEditingMessage();
        }
        this.releaseMessageResources([this.messages[index]]);
        this.messages.splice(index, 1);
      });
      // Backend now broadcasts messages_updated so the other user reloads too
    } catch (e: any) {
      console.error('[Messaging] Failed to delete message:', e);
    }
  }

  @HostListener('document:click')
  closeReactionPicker(): void {
    this.reactionPickerMessageId = null;
  }

  openReactionPicker(messageId: string, event: MouseEvent): void {
    event.stopPropagation();
    this.reactionPickerMessageId = this.reactionPickerMessageId === messageId ? null : messageId;
  }

  async toggleReaction(messageId: string, emoji: string, event: MouseEvent): Promise<void> {
    event.stopPropagation();
    this.reactionPickerMessageId = null;
    if (!messageId) return;

    const wasMine = this.hasMyReaction(messageId, emoji);
    this.applyReactionLocally(messageId, emoji, !wasMine);

    try {
      await this.conversationService.toggleReaction(messageId, emoji);
      // The backend broadcasts messages_updated to ALL conversation members (including this user).
      // The WS handler below catches that and calls loadMessages(false) for everyone.
      // No direct loadMessages call here — that would race with the WS-triggered one.
    } catch (e) {
      console.error('[Messaging] Failed to toggle reaction:', e);
      this.applyReactionLocally(messageId, emoji, wasMine);
    }
  }

  private applyReactionLocally(messageId: string, emoji: string, add: boolean): void {
    // Clone outer map so Angular detects the reference change and re-evaluates template methods.
    const nextReactions = new Map(this.messageReactions);

    let emojiMap = nextReactions.get(messageId);
    if (!emojiMap) {
      emojiMap = new Map();
    } else {
      emojiMap = new Map(emojiMap);
    }
    nextReactions.set(messageId, emojiMap);

    const users = emojiMap.get(emoji) ?? [];
    const idx = users.findIndex(u => u.username === this.currentUsername);

    if (add) {
      if (idx < 0) {
        emojiMap.set(emoji, [...users, { username: this.currentUsername, displayName: this.currentDisplayName }]);
      }
    } else {
      if (idx >= 0) {
        const updated = [...users];
        updated.splice(idx, 1);
        if (updated.length === 0) {
          emojiMap.delete(emoji);
        } else {
          emojiMap.set(emoji, updated);
        }
      }
    }

    this.messageReactions = nextReactions;
  }

  getMessageReactions(messageId: string): { emoji: string; count: number; users: string[]; isMine: boolean }[] {
    const emojiMap = this.messageReactions.get(messageId);
    if (!emojiMap) return [];
    return Array.from(emojiMap.entries())
      .filter(([, users]) => users.length > 0)
      .map(([emoji, users]) => ({
        emoji,
        count: users.length,
        users: users.map(u => u.displayName),
        isMine: users.some(u => u.username === this.currentUsername),
      }));
  }

  hasMyReaction(messageId: string, emoji: string): boolean {
    return this.messageReactions.get(messageId)?.get(emoji)?.some(u => u.username === this.currentUsername) ?? false;
  }

  getMessageLifetimeLabel(value: number = this.messageLifetime): string {
    return this.lifetimeOptions.find(option => option.value === value)?.label ?? 'Never';
  }

  formatExpiryLabel(expiresAt: number | undefined): string {
    if (expiresAt === undefined || expiresAt === null) {
      return '';
    }
    const nowSec = Math.floor(Date.now() / 1000);
    const remaining = expiresAt - nowSec;
    if (remaining <= 0) {
      return 'Expiring…';
    }
    if (remaining < 60) {
      return 'Expires in < 1 min';
    }
    if (remaining < 3600) {
      const mins = Math.floor(remaining / 60);
      return `Expires in ${mins} min`;
    }
    if (remaining < 86400) {
      const hours = Math.floor(remaining / 3600);
      return `Expires in ${hours} hr`;
    }
    const days = Math.floor(remaining / 86400);
    return `Expires in ${days} day${days !== 1 ? 's' : ''}`;
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
    return conv ? (conv.fullName || conv.name) : this.conversationId.substring(0, 8);
  }

  isActiveConversationGroup(): boolean {
    return (this.conversations.find(c => c.id === this.conversationId)?.memberCount ?? 0) > 2;
  }

  isManagingMemberAction(username: string): boolean {
    return !!this.memberActionInProgress[username];
  }

  isConversationMemberMarkedForRemoval(userId: string): boolean {
    return this.pendingRemovedMemberIds.includes(userId);
  }

  togglePendingConversationMemberRemoval(member: ConversationMember): void {
    if (member.friendshipStatus === 'self') {
      return;
    }

    this.settingsError = '';
    this.manageError = '';

    if (this.isConversationMemberMarkedForRemoval(member.userId)) {
      this.pendingRemovedMemberIds = this.pendingRemovedMemberIds.filter(userId => userId !== member.userId);
      return;
    }

    const currentMemberCount = this.conversationMembers.length || (this.conversations.find(c => c.id === this.conversationId)?.memberCount ?? 0);
    if (currentMemberCount - this.pendingRemovedMemberIds.length - 1 < 2) {
      this.manageError = 'A conversation must keep at least two members.';
      return;
    }

    this.pendingRemovedMemberIds = [...this.pendingRemovedMemberIds, member.userId];
  }

  getConversationMemberLabel(member: ConversationMember): string {
    return member.displayName || member.username;
  }

  async sendFriendRequestToConversationMember(member: ConversationMember): Promise<void> {
    if (member.friendshipStatus !== 'none' || this.isConversationMemberMarkedForRemoval(member.userId)) {
      return;
    }

    this.memberActionInProgress = { ...this.memberActionInProgress, [member.username]: true };
    this.manageError = '';

    try {
      await this.friendService.sendFriendRequest(member.username);
      member.friendshipStatus = 'pending_outgoing';
    } catch (e: any) {
      console.error('[Messaging] Failed to send friend request from manage modal:', e);
      this.manageError = e?.message || 'Failed to send friend request.';
    } finally {
      const next = { ...this.memberActionInProgress };
      delete next[member.username];
      this.memberActionInProgress = next;
    }
  }

  private async addSelectedConversationMembers(): Promise<void> {
    if (!this.conversationId) {
      return;
    }

    const roomKey = this.conversationPassphrases.get(this.conversationId)?.trim();
    if (!roomKey) {
      this.createConversationError = 'Open the group chat with its room key before adding new members.';
      return;
    }

    const memberUsernames = Array.from(
      new Set(this.selectedConversationMembers.map(username => username.trim()).filter(Boolean)),
    );

    if (memberUsernames.length === 0) {
      this.createConversationError = 'Please select at least one friend.';
      return;
    }

    try {
      await this.conversationService.addConversationMembers(this.conversationId, memberUsernames, roomKey);

      this.modal = { type: 'none' };
      this.createConversationError = '';
      this.manageError = '';
      this.settingsError = '';
      this.selectedConversationMembers = [];

      await Promise.all([
        this.refreshConversationList(),
        this.openConversationSettings(),
        this.loadMessages(this.conversationId),
      ]);
    } catch (e: any) {
      console.error('[Messaging] Failed to add members to conversation:', e);
      this.createConversationError = e?.message || 'Failed to add conversation members.';
    }
  }

  private pictureRefreshInterval: ReturnType<typeof setInterval> | null = null;

  private startPictureRefresh(convId: string): void {
    this.stopPictureRefresh();
    this.pictureRefreshInterval = setInterval(async () => {
      const otherMsg = this.messages.find(m => !m.isMine && !m.isSystem);
      if (!otherMsg) return;
      try {
        const res = await fetch(
          `http://localhost:8080/api/profile/by-username/${otherMsg.username}`,
          { credentials: 'include' }
        );
        if (!res.ok) return;
        const data = await res.json();
        this.ngZone.run(() => {
          const newUrl = data.profile_picture_url || '';
          this.activeConversationPictureUrl = newUrl;
          for (const msg of this.messages) {
            if (!msg.isMine) {
              msg.profilePictureUrl = newUrl;
            }
          }
        });
      } catch {}
    }, 15000);
  }

  private stopPictureRefresh(): void {
    if (this.pictureRefreshInterval !== null) {
      clearInterval(this.pictureRefreshInterval);
      this.pictureRefreshInterval = null;
    }
  }

  ngOnDestroy(): void {
    this.messageSub?.unsubscribe();
    this.routeQuerySub?.unsubscribe();
    this.stopPictureRefresh();
    this.releaseMessageResources(this.messages);
  }

  private openCreateConversationModal(memberUsernames: string[]): void {
    this.memberPickerMode = 'create';
    this.modal = { type: 'create-room-key', memberUsernames: [...memberUsernames] };
    this.roomKeyInput = this.cryptoService.generateRoomKey();
    this.roomKeyError = '';
    this.roomKeyCopied = false;
    this.createConversationError = '';
    this.errorMessage = '';
  }

  private openCreateConversationModalImmediately(memberUsernames: string[]): void {
    this.ngZone.run(() => {
      this.openCreateConversationModal(memberUsernames);
      this.cdr.detectChanges();
    });
  }

  private async openCreateConversationMemberModal(preselectedUsernames: string[] = [], mode: 'create' | 'add' = 'create'): Promise<void> {
    this.memberPickerMode = mode;
    this.modal = { type: 'select-conversation-members' };
    this.isLoadingFriends = true;
    this.createConversationError = '';
    this.selectedConversationMembers = [...preselectedUsernames];
    if (this.selectedConversationMembers.length <= 1) {
      this.groupConversationNameInput = '';
    }

    try {
      const friends = await this.friendService.getFriends();
      this.ngZone.run(() => {
        const existingConversationMembers = mode === 'add'
          ? new Set(this.conversationMembers.map(member => member.username))
          : new Set<string>();

        this.availableFriends = friends
          .filter(friend => friend.accepted)
          .filter(friend => !existingConversationMembers.has(friend.username))
          .sort((a, b) => a.username.localeCompare(b.username));

        const validFriendUsernames = new Set(this.availableFriends.map(friend => friend.username));
        this.selectedConversationMembers = this.selectedConversationMembers.filter(username => validFriendUsernames.has(username));
      });
    } catch (e: any) {
      this.ngZone.run(() => {
        this.availableFriends = [];
        this.createConversationError = e?.message || 'Failed to load your friends.';
      });
    } finally {
      this.ngZone.run(() => {
        this.isLoadingFriends = false;
      });
    }
  }

  private async refreshConversationList(): Promise<void> {
    try {
      const convs = await this.conversationService.getConversations();
      this.ngZone.run(() => {
        this.conversations = convs.map(c => ({
          id: c.id,
          name: this.formatConversationName(c.name),
          fullName: c.name,
          lastMessage: c.last_message ? '🔒 Encrypted message' : '',
          lastMessageTime: c.last_message_time
            ? this.formatTimeShort(new Date(c.last_message_time * 1000))
            : '',
          messageLifetime: c.message_lifetime ?? 0,
          memberCount: c.member_count ?? 2,
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
      this.conversationPassphrases.set(convId, roomKey);

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

  private async startNewConversationWith(memberUsernames: string[], passphrase: string, groupName: string = ''): Promise<void> {
    const uniqueMemberUsernames = Array.from(
      new Set(memberUsernames.map(username => username.trim()).filter(Boolean)),
    );

    if (uniqueMemberUsernames.length === 0) {
      this.errorMessage = 'Please select at least one friend.';
      return;
    }

    try {
      const trimmedGroupName = uniqueMemberUsernames.length > 1 ? groupName.trim() : '';
      const result = await this.conversationService.createConversation(uniqueMemberUsernames, passphrase, trimmedGroupName);

      this.ngZone.run(() => {
        this.conversationId = result.conversation_id;
        this.releaseMessageResources(this.messages);
        this.messages = [];
        this.pendingAttachments = [];
        this.newMessage = '';
        this.composerError = '';
        this.errorMessage = '';
        this.createConversationError = '';
        this.isConnected = true;
        this.addConversationToList(
          result.conversation_id,
          this.buildConversationName(uniqueMemberUsernames, trimmedGroupName),
          uniqueMemberUsernames.length + 1,
        );
      });

      if (result.created) {
        const key = await this.cryptoService.deriveConversationKey(passphrase, result.conversation_id);
        this.conversationKeys.set(result.conversation_id, key);
        this.conversationPassphrases.set(result.conversation_id, passphrase);

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

  private async loadMessages(convId: string, scroll = true): Promise<void> {
    const convKey = this.conversationKeys.get(convId);
    if (!convKey) return;

    try {
      const history = await this.conversationService.getMessages(convId);
      const decrypted: Message[] = await Promise.all(
        history.map(async (m: any) => {
          const messageId = m.ID ?? m.id ?? '';
          const senderID = m.SenderID ?? m.sender_id ?? '';
          const username = m.DisplayName || m.Username || '';
          const createdAt = this.formatTime(new Date(m.CreatedAt * 1000));

          if (m.IsSystem || !senderID) {
            return this.buildSystemMessage(
              messageId,
              createdAt,
              typeof m.Ciphertext === 'string' ? m.Ciphertext : String(m.Ciphertext ?? ''),
            );
          }

          try {
            const content = await this.cryptoService.decryptMessage(m.Ciphertext, convKey);
            return this.buildMessageFromDecryptedContent(
              messageId,
              username,
              createdAt,
              (m.Username ?? '') === this.currentUsername,
              content,
              m.ProfilePictureURL ?? '',
              m.ExpiresAt ?? undefined,
            );
          } catch {
            return {
              id: messageId,
              username,
              time: createdAt,
              content: '🔒 Could not decrypt message',
              isMine: (m.Username ?? '') === this.currentUsername,
              isSystem: false,
              attachments: [],
              profilePictureUrl: m.ProfilePictureURL ?? '',
              expiresAt: m.ExpiresAt ?? undefined,
            };
          }
        })
      );
      const nextReactions = new Map<string, Map<string, ReactionUser[]>>();
      for (const m of history) {
        const messageId = m.ID ?? m.id ?? '';
        const rawReactions: any[] = m.Reactions ?? [];
        if (!messageId || rawReactions.length === 0) continue;
        const emojiMap = new Map<string, ReactionUser[]>();
        for (const r of rawReactions) {
          const username = r.Username ?? r.username ?? '';
          const displayName = r.DisplayName ?? r.display_name ?? username;
          const emoji = r.Emoji ?? r.emoji ?? '';
          if (!emoji || !username) continue;
          const users = emojiMap.get(emoji) ?? [];
          users.push({ username, displayName });
          emojiMap.set(emoji, users);
        }
        if (emojiMap.size > 0) {
          nextReactions.set(messageId, emojiMap);
        }
      }

      this.ngZone.run(() => {
        this.releaseMessageResources(this.messages);
        this.messages = decrypted;
        this.messageReactions = nextReactions;
        if (scroll) {
          this.shouldScrollToBottom = true;
        }
        const otherMsg = decrypted.find(m => !m.isMine && !m.isSystem);
        this.activeConversationPictureUrl = otherMsg?.profilePictureUrl ?? '';
        this.startPictureRefresh(convId);
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

  private async buildRichMessagePayloadFromExistingAttachments(text: string, attachments: MessageAttachment[]): Promise<RichMessagePayload> {
    const serializedAttachments = await Promise.all(attachments.map(attachment => this.serializeAttachmentForEdit(attachment)));

    return {
      version: 1,
      type: 'rich_message',
      text,
      attachments: serializedAttachments,
    };
  }

  private async serializeAttachmentForEdit(attachment: MessageAttachment): Promise<RichMessageAttachmentPayload> {
    const response = await fetch(attachment.downloadUrl);
    const buffer = await response.arrayBuffer();

    return {
      file_name: attachment.fileName,
      mime_type: attachment.mimeType,
      size: attachment.size,
      data_b64: this.cryptoService.bytesToBase64(new Uint8Array(buffer)),
    };
  }

  private buildSystemMessage(id: string, time: string, content: string): Message {
    return {
      id,
      username: '',
      time,
      content,
      isMine: false,
      isSystem: true,
      attachments: [],
      profilePictureUrl: '',
    };
  }

  private buildMessageFromDecryptedContent(id: string, username: string, time: string, isMine: boolean, plaintext: string, profilePictureUrl: string = '', expiresAt?: number): Message {
  const payload = this.tryParseRichMessagePayload(plaintext);
    if (!payload) {
      return {
        id,
        username,
        time,
        content: plaintext,
        isMine,
        isSystem: false,
        attachments: [],
        profilePictureUrl,
        expiresAt,
      };
    }

    return {
      id,
      username,
      time,
      content: payload.text,
      isMine,
      isSystem: false,
      attachments: payload.attachments.map(attachment => this.createMessageAttachmentFromPayload(attachment)),
      profilePictureUrl,
      expiresAt,
    };
  }

  private createRichMessageFromFiles(id: string, username: string, time: string, isMine: boolean, text: string, attachments: PendingAttachment[]): Message {
    return {
      id,
      username,
      time,
      content: text,
      isMine,
      isSystem: false,
      attachments: attachments.map(attachment => this.createMessageAttachmentFromFile(attachment.file)),
      profilePictureUrl: this.currentUserPictureUrl,
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

  private resetActiveConversationState(convId: string): void {
    if (!convId) {
      return;
    }

		this.conversationKeys.delete(convId);
		this.conversationPassphrases.delete(convId);
    if (this.conversationId !== convId) {
      return;
    }

    this.releaseMessageResources(this.messages);
    this.messages = [];
    this.messageReactions = new Map();
    this.pendingAttachments = [];
    this.newMessage = '';
    this.composerError = '';
    this.settingsError = '';
    this.manageError = '';
    this.isLoadingConversationMembers = false;
    this.conversationMembers = [];
    this.memberActionInProgress = {};
    this.roomKeyInput = '';
    this.roomKeyError = '';
    this.roomKeyCopied = false;
    this.modal = { type: 'none' };
    this.cancelEditingMessage(false);
    this.openMessageMenuId = null;
    this.reactionPickerMessageId = null;
    this.activeConversationPictureUrl = '';
    this.stopPictureRefresh();
    this.conversationId = '';
    this.isConnected = false;
    this.messageLifetime = 0;
    this.selectedMessageLifetime = 0;
    this.editedGroupName = '';
  }

  private releaseMessageResources(messages: Message[]): void {
    for (const message of messages) {
      for (const attachment of message.attachments) {
        URL.revokeObjectURL(attachment.downloadUrl);
      }
    }
  }

  private applyMessageAck(messageId: string, clientMessageId?: string): void {
    if (!messageId || !clientMessageId) {
      return;
    }

    const optimisticMessage = this.messages.find(message => message.id === clientMessageId);
    if (optimisticMessage) {
      optimisticMessage.id = messageId;
    }
  }

  private createTempMessageId(): string {
    return `temp-${this.createUniqueId()}`;
  }

  private createUniqueId(): string {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
      return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }

  private addConversationToList(id: string, name: string, memberCount: number = 2): void {
    if (this.conversations.find(c => c.id === id)) return;
    this.conversations.unshift({
      id,
      name: this.formatConversationName(name),
      lastMessage: '',
      lastMessageTime: '',
      fullName: name,
      messageLifetime: 0,
      memberCount,
    });
  }

  private updateConversationName(convId: string, displayName: string): void {
    const conv = this.conversations.find(c => c.id === convId);
    if (conv && conv.name === convId.substring(0, 8)) {
      conv.fullName = displayName;
      conv.name = this.formatConversationName(displayName);
    }
  }

  private buildConversationName(memberUsernames: string[], groupName: string = ''): string {
    const normalizedGroupName = groupName.trim();
    if (normalizedGroupName) {
      return normalizedGroupName;
    }
    return memberUsernames.join(', ');
  }

  private formatConversationName(name: string): string {
    const normalizedName = name.trim().replace(/\s+/g, ' ');
    if (normalizedName.length <= 30) {
      return normalizedName;
    }
    return `${normalizedName.slice(0, 27).trimEnd()}...`;
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
