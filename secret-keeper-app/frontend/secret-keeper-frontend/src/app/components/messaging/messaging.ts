import { Component, NgZone, OnInit, OnDestroy, ViewChild, ElementRef, AfterViewChecked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
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
  isConnected: boolean = false;

  currentUsername: string = '';
  conversations: Conversation[] = [];

  private messageSub: Subscription | null = null;
  private shouldScrollToBottom = false;

  @ViewChild('messagesArea') private messagesArea?: ElementRef;

  constructor(
    private ngZone: NgZone,
    private router: Router,
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
    this.currentUsername = user.display_name || user.username;

    this.messageSub = this.messagingService.messages$.subscribe((incoming) => {
      if (incoming.conversation_id !== this.conversationId) {
        // Update sidebar preview for other conversations
        this.updateConversationPreview(incoming.conversation_id, incoming.ciphertext);
        return;
      }

      if (incoming.sender_id === this.currentUsername) return;

      const msg: Message = {
        username: incoming.sender_id,
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
  }

  ngAfterViewChecked(): void {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  selectConversation(convId: string): void {
    if (this.conversationId === convId) return;
    this.conversationId = convId;
    this.messages = [];
    this.errorMessage = '';

    if (!this.messagingService.isConnected()) {
      this.messagingService.connect();
    }
    this.isConnected = true;
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

    this.addConversationToList(this.conversationId.trim(), this.conversationId.trim().substring(0, 8));
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
      username: this.currentUsername,
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
