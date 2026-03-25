import { Component, NgZone, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewChecked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { MessagingService } from '../../services/messaging.service';
import { ConversationService } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';

interface Message {
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
}

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
  newConversationName: string = '';
  isConnected: boolean = false;

  currentUsername: string = '';
  currentDisplayName: string = '';
  conversations: Conversation[] = [];

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
  ) {}

  async ngOnInit(): Promise<void> {
    const user = await this.authService.loadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    this.currentUsername = user.username;
    this.currentDisplayName = user.display_name || user.username;

    try {
      const convs = await this.conversationService.getConversations();
      this.ngZone.run(() => {
        this.conversations = convs.map(c => ({
          id: c.id,
          name: c.name,
          lastMessage: c.last_message,
          lastMessageTime: c.last_message_time
            ? this.formatTimeShort(new Date(c.last_message_time * 1000))
            : '',
        }));
        if (this.conversations.length > 0) {
          this.isConnected = true;
        }
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to load conversations:', e);
    }

    this.messagingService.connect();

    this.messageSub = this.messagingService.messages$.subscribe((incoming) => {
      if (incoming.sender_id !== this.currentUsername) {
        this.updateConversationName(incoming.conversation_id, incoming.display_name || incoming.sender_id);
      }

      if (incoming.conversation_id !== this.conversationId) {
        this.updateConversationPreview(incoming.conversation_id, incoming.ciphertext);
        return;
      }

      if (incoming.sender_id === this.currentUsername) return;

      const msg: Message = {
        username: incoming.display_name || incoming.sender_id,
        time: this.formatTime(new Date()),
        content: incoming.ciphertext,
        isMine: false,
      };

      this.ngZone.run(() => {
        this.messages.push(msg);
        this.updateConversationPreview(incoming.conversation_id, incoming.ciphertext);
        this.shouldScrollToBottom = true;
      });
    });

    const chatWith = this.route.snapshot.queryParamMap.get('chatWith');
    if (chatWith) {
      try {
        const convId = await this.conversationService.createConversation([chatWith]);
        this.ngZone.run(() => {
          this.conversationId = convId;
          this.messages = [];
          this.errorMessage = '';
          this.addConversationToList(convId, chatWith);
          this.messagingService.connect();
          this.isConnected = true;
        });
      } catch (e: any) {
        this.ngZone.run(() => {
          this.errorMessage = e.message || 'Failed to start conversation.';
        });
      }
    }
  }

  ngAfterViewChecked(): void {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  async selectConversation(convId: string): Promise<void> {
    if (this.conversationId === convId) return;
    
    this.conversationId = convId;
    this.messages = [];
    this.errorMessage = '';
    this.isConnected = true;

    if (!this.messagingService.isConnected()) {
      this.messagingService.connect();
    }

    try {
      const history = await this.conversationService.getMessages(convId);
      this.ngZone.run(() => {
        this.messages = history.map((m: any) => ({
          username: m.DisplayName || m.Username,
          time: this.formatTime(new Date(m.CreatedAt * 1000)),
          content: m.Ciphertext,
          isMine: m.Username === this.currentUsername,
        }));
        this.shouldScrollToBottom = true;
      });
    } catch (e: any) {
      console.error('[Messaging] Failed to load message history:', e);
    }
  }

  connectToConversation(): void {
    if (!this.conversationId.trim()) {
      this.errorMessage = 'Please enter a conversation ID.';
      return;
    }
    this.errorMessage = '';
    this.messages = [];

    this.messagingService.connect();
    this.isConnected = true;

    const convName = this.newConversationName.trim() || this.conversationId.trim().substring(0, 8);
    this.addConversationToList(this.conversationId.trim(), convName);
    this.newConversationName = '';
  }

  async startNewConversation(): Promise<void> {
    if (!this.newConversationMemberId.trim()) {
      this.errorMessage = 'Please enter a username to start a conversation with.';
      return;
    }

    try {
      const username = this.newConversationMemberId.trim();
      const convId = await this.conversationService.createConversation([username]);
      this.conversationId = convId;
      this.newConversationMemberId = '';
      this.errorMessage = '';
      this.messages = [];

      this.addConversationToList(convId, username);

      this.messagingService.connect();
      this.isConnected = true;
    } catch (e: any) {
      this.ngZone.run(() => {
        this.errorMessage = e.message || 'Failed to create conversation.';
      });
    }
  }

  sendMessage(): void {
    if (!this.newMessage.trim()) return;

    if (!this.conversationId) {
      this.errorMessage = 'Join or create a conversation first.';
      return;
    }

    if (!this.messagingService.isConnected()) {
      this.errorMessage = 'Not connected. Try rejoining the conversation.';
      return;
    }

    this.messagingService.sendMessage(this.conversationId, this.newMessage.trim());

    const msg: Message = {
      username: this.currentDisplayName,
      time: this.formatTime(new Date()),
      content: this.newMessage.trim(),
      isMine: true,
    };
    this.messages.push(msg);
    this.updateConversationPreview(this.conversationId, this.newMessage.trim());
    this.newMessage = '';
    this.shouldScrollToBottom = true;
  }

  goToProfile(): void {
    this.router.navigate(['/profile']);
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

  private addConversationToList(id: string, name: string): void {
    if (this.conversations.find(c => c.id === id)) return;
    this.conversations.unshift({
      id,
      name,
      lastMessage: '',
      lastMessageTime: '',
    });
  }

  private updateConversationName(convId: string, displayName: string): void {
    const conv = this.conversations.find(c => c.id === convId);
    if (conv && conv.name === convId.substring(0, 8)) {
      conv.name = displayName;
    }
  }

  private updateConversationPreview(convId: string, text: string): void {
    const conv = this.conversations.find(c => c.id === convId);
    if (conv) {
      conv.lastMessage = text.length > 40 ? text.substring(0, 40) + '...' : text;
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
