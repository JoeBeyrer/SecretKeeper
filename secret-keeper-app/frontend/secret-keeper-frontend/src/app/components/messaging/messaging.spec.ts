import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, ActivatedRoute } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { Subject } from 'rxjs';
import { vi } from 'vitest';

import { Messaging } from './messaging';
import { MessagingService, IncomingMessage } from '../../services/messaging.service';
import { ConversationService } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';
import { CryptoService } from '../../services/crypto.service';

describe('Messaging', () => {
  let component: Messaging;
  let fixture: ComponentFixture<Messaging>;

  let messagingServiceSpy: any;
  let conversationServiceSpy: any;
  let authServiceSpy: any;
  let cryptoServiceSpy: any;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };
  let messageSubject: Subject<IncomingMessage>;

  const mockUser = {
    username: 'alice',
    email: 'alice@example.com',
    display_name: 'Alice',
    bio: '',
    profile_picture_url: '',
  };

  const mockConversations = [
    { id: 'conv-1', name: 'Bob', last_message: 'encrypted', last_message_time: 1700000000 },
  ];

  beforeEach(async () => {
    messageSubject = new Subject<IncomingMessage>();

    messagingServiceSpy = {
      connect: vi.fn(),
      disconnect: vi.fn(),
      sendMessage: vi.fn(),
      isConnected: vi.fn().mockReturnValue(true),
      messages$: messageSubject.asObservable(),
    };

    conversationServiceSpy = {
      getConversations: vi.fn().mockResolvedValue(mockConversations),
      createConversation: vi.fn(),
      getMessages: vi.fn(),
      verifyRoomKey: vi.fn(),
      claimRoomKey: vi.fn(),
    };

    authServiceSpy = {
      loadCurrentUser: vi.fn().mockResolvedValue(mockUser),
    };

    cryptoServiceSpy = {
      generateRoomKey: vi.fn(),
      deriveConversationKey: vi.fn(),
      encryptMessage: vi.fn(),
      decryptMessage: vi.fn(),
    };

    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Messaging, FormsModule],
      providers: [
        { provide: MessagingService, useValue: messagingServiceSpy },
        { provide: ConversationService, useValue: conversationServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: CryptoService, useValue: cryptoServiceSpy },
        { provide: Router, useValue: routerSpy },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { queryParamMap: { get: () => null } } },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(Messaging);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should redirect to /login if user is not authenticated', async () => {
    authServiceSpy.loadCurrentUser.mockResolvedValue(null);
    const f = TestBed.createComponent(Messaging);
    await f.whenStable();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('should set currentUsername and currentDisplayName from the authenticated user', () => {
    expect(component.currentUsername).toBe('alice');
    expect(component.currentDisplayName).toBe('Alice');
  });

  it('should load and map conversations on init', () => {
    expect(component.conversations.length).toBe(1);
    expect(component.conversations[0].name).toBe('Bob');
    expect(component.conversations[0].id).toBe('conv-1');
  });

  it('should call messagingService.connect() on init', () => {
    expect(messagingServiceSpy.connect).toHaveBeenCalled();
  });

  it('startNewConversation() should set errorMessage when username is empty', async () => {
    component.newConversationMemberId = '   ';
    await component.startNewConversation();
    expect(component.errorMessage).toBeTruthy();
    expect(conversationServiceSpy.createConversation).not.toHaveBeenCalled();
  });

  it('sendMessage() should do nothing when newMessage is empty', async () => {
    component.newMessage = '   ';
    await component.sendMessage();
    expect(messagingServiceSpy.sendMessage).not.toHaveBeenCalled();
  });

  it('sendMessage() should set errorMessage when no conversationId is set', async () => {
    component.newMessage = 'hello';
    component.conversationId = '';
    await component.sendMessage();
    expect(component.errorMessage).toContain('Join or create');
  });

  it('sendMessage() should set errorMessage when socket is not connected', async () => {
    component.newMessage = 'hello';
    component.conversationId = 'conv-1';
    messagingServiceSpy.isConnected.mockReturnValue(false);
    await component.sendMessage();
    expect(component.errorMessage).toContain('Not connected');
  });

  it('sendMessage() should encrypt and send the message', async () => {
    const mockKey = {} as CryptoKey;
    component.conversationId = 'conv-1';
    component.conversationKeys.set('conv-1', mockKey);
    component.newMessage = 'hello world';
    cryptoServiceSpy.encryptMessage.mockResolvedValue('encrypted-blob');
    messagingServiceSpy.isConnected.mockReturnValue(true);

    await component.sendMessage();

    expect(cryptoServiceSpy.encryptMessage).toHaveBeenCalledWith('hello world', mockKey);
    expect(messagingServiceSpy.sendMessage).toHaveBeenCalledWith('conv-1', 'encrypted-blob');
    expect(component.newMessage).toBe('');
    expect(component.messages[component.messages.length - 1].content).toBe('hello world');
    expect(component.messages[component.messages.length - 1].isMine).toBe(true);
  });

  it('closeModal() should reset modal state', () => {
    component.modal = { type: 'enter-room-key', convId: 'conv-1' };
    component.roomKeyInput = 'somekey';
    component.closeModal();
    expect(component.modal.type).toBe('none');
    expect(component.roomKeyInput).toBe('');
  });

  it('goTo() should navigate to the given page', () => {
    component.goTo('friends');
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/friends']);
  });

  it('getActiveConversationName() should return name of current conversation', () => {
    component.conversations = [{ id: 'conv-1', name: 'Bob', lastMessage: '', lastMessageTime: '' }];
    component.conversationId = 'conv-1';
    expect(component.getActiveConversationName()).toBe('Bob');
  });

  it('getActiveConversationName() should return truncated id when conversation is unknown', () => {
    component.conversationId = 'unknown-conv-xyz';
    expect(component.getActiveConversationName()).toBe('unknown-');
  });

  it('selectConversation() should prompt for room key when claimRoomKey returns NOT_AVAILABLE', async () => {
    conversationServiceSpy.claimRoomKey.mockRejectedValue(new Error('ROOM_KEY_NOT_AVAILABLE'));
    await component.selectConversation('conv-1');
    expect(component.modal.type).toBe('enter-room-key');
  });

  it('submitRoomKey() should set roomKeyError when input is empty', async () => {
    component.modal = { type: 'enter-room-key', convId: 'conv-1' };
    component.roomKeyInput = '   ';
    await component.submitRoomKey();
    expect(component.roomKeyError).toBeTruthy();
  });

  it('sendMessage() should set errorMessage when no room key is cached for the conversation', async () => {
    component.newMessage = 'hello';
    component.conversationId = 'conv-1';
    await component.sendMessage();
    expect(component.errorMessage).toContain('room key');
  });
});
