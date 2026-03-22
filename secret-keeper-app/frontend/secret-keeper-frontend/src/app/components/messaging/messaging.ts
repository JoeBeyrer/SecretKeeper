import { Component, NgZone, OnInit, OnDestroy } from '@angular/core';
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

@Component({
  selector: 'app-messaging',
  imports: [FormsModule],
  templateUrl: './messaging.html',
  styleUrl: './messaging.css',
})
export class Messaging implements OnInit, OnDestroy {
  messages: Message[] = [];
  newMessage: string = '';
  errorMessage: string = '';

  // Conversation state
  conversationId: string = '';
  newConversationMemberId: string = ''; 
  isConnected: boolean = false;

  currentUsername: string = '';

  private messageSub: Subscription | null = null;

  constructor(
    private ngZone: NgZone,
    private router: Router,
    private messagingService: MessagingService,
    private conversationService: ConversationService,
    private authService: AuthService,
  ) {}

  async ngOnInit(): Promise<void> {
    // Load the current user to label messages
    const user = await this.authService.loadCurrentUser();
    if (!user) {
      this.router.navigate(['/login']);
      return;
    }
    this.currentUsername = user.display_name || user.username;

    // Subscribe to incoming messages from the service
    this.messageSub = this.messagingService.messages$.subscribe((incoming) => {
      if (incoming.conversation_id !== this.conversationId) return;

      // Skip our own messages
      if (incoming.sender_id === this.currentUsername) return;

      const msg: Message = {
        username: incoming.sender_id,
        time: this.formatTime(new Date()),
        content: incoming.ciphertext,
        isMine: false,
      };

      this.ngZone.run(() => {
        this.messages.push(msg);
      });
    });
  }

  // Connect to a specific conversation - currently using conv ID, will change to panel view
  connectToConversation(): void {
    if (!this.conversationId.trim()) {
      this.errorMessage = 'Please enter a conversation ID.';
      return;
    }
    this.errorMessage = '';
    this.messages = []; // clear old messages when switching conversations

    this.messagingService.connect();
    this.isConnected = this.messagingService.isConnected();
  }

  // Create a new conversation with specified user
  async startNewConversation(): Promise<void> {
    if (!this.newConversationMemberId.trim()) {
      this.errorMessage = 'Please enter a user ID to start a conversation with.';
      return;
    }

    try {
      const convId = await this.conversationService.createConversation([
        this.newConversationMemberId.trim(),
      ]);
      this.conversationId = convId;
      this.newConversationMemberId = '';
      this.connectToConversation();
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

    // MVP: sends plaintext in the ciphertext field.
    this.messagingService.sendMessage(this.conversationId, this.newMessage.trim());

    this.messages.push({
      username: this.currentUsername,
      time: this.formatTime(new Date()),
      content: this.newMessage.trim(),
      isMine: true,
    });

    this.newMessage = '';
  }

  goToProfile(): void {
    this.router.navigate(['/profile']);
  }

  ngOnDestroy(): void {
    this.messageSub?.unsubscribe();
  }

  private formatTime(d: Date): string {
    const hh = String(d.getUTCHours()).padStart(2, '0');
    const mm = String(d.getUTCMinutes()).padStart(2, '0');
    const ss = String(d.getUTCSeconds()).padStart(2, '0');
    const month = d.getUTCMonth() + 1;
    const day = d.getUTCDate();
    const year = d.getUTCFullYear();
    return `${hh}:${mm}:${ss} ${month}-${day}-${year}`;
  }
}