import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, ActivatedRoute, convertToParamMap } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { BehaviorSubject, Subject } from 'rxjs';
import { vi } from 'vitest';

import { Messaging } from './messaging';
import { MessagingService, IncomingMessage } from '../../services/messaging.service';
import { ConversationService } from '../../services/conversation.service';
import { AuthService } from '../../services/auth.service';
import { CryptoService } from '../../services/crypto.service';
import { FriendService } from '../../services/friend.service';

describe('Messaging', () => {
  let component: Messaging;
  let fixture: ComponentFixture<Messaging>;

  let messagingServiceSpy: any;
  let conversationServiceSpy: any;
  let authServiceSpy: any;
  let cryptoServiceSpy: any;
  let friendServiceSpy: any;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };
  let messageSubject: Subject<IncomingMessage>;
  let queryParamMap$: BehaviorSubject<any>;

  const mockUser = {
    username: 'alice',
    email: 'alice@example.com',
    display_name: 'Alice',
    bio: '',
    profile_picture_url: '',
  };

  const mockConversations = [
    { id: 'conv-1', name: 'bob', last_message: 'encrypted', last_message_time: 1700000000, message_lifetime: 1440 },
  ];

  const mockFriends = [
    { user_id: 'friend-1', username: 'bob', display_name: 'Bob', accepted: true },
    { user_id: 'friend-2', username: 'carol', display_name: 'Carol', accepted: true },
  ];

  beforeEach(async () => {
    messageSubject = new Subject<IncomingMessage>();
    queryParamMap$ = new BehaviorSubject(convertToParamMap({}));

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
      getMessages: vi.fn().mockResolvedValue([]),
      verifyRoomKey: vi.fn(),
      claimRoomKey: vi.fn(),
      setMessageLifetime: vi.fn().mockResolvedValue(undefined),
      leaveConversation: vi.fn().mockResolvedValue(undefined),
      editMessage: vi.fn().mockResolvedValue(undefined),
      DeleteMessage: vi.fn().mockResolvedValue(undefined),
    };

    authServiceSpy = {
      reloadCurrentUser: vi.fn().mockResolvedValue(mockUser),
    };

    cryptoServiceSpy = {
      generateRoomKey: vi.fn().mockReturnValue('generated-room-key'),
      deriveConversationKey: vi.fn(),
      encryptMessage: vi.fn(),
      decryptMessage: vi.fn(),
      bytesToBase64: vi.fn((bytes: Uint8Array) => {
        let binary = '';
        for (const byte of bytes) {
          binary += String.fromCharCode(byte);
        }
        return btoa(binary);
      }),
      base64ToBytes: vi.fn((base64: string) => {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let index = 0; index < binary.length; index += 1) {
          bytes[index] = binary.charCodeAt(index);
        }
        return bytes;
      }),
      base64ToArrayBuffer: vi.fn((base64: string) => {
        const binary = atob(base64);
        const buffer = new ArrayBuffer(binary.length);
        const bytes = new Uint8Array(buffer);
        for (let index = 0; index < binary.length; index += 1) {
          bytes[index] = binary.charCodeAt(index);
        }
        return buffer;
      }),
    };

    friendServiceSpy = {
      getFriends: vi.fn().mockResolvedValue(mockFriends),
    };

    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Messaging, FormsModule],
      providers: [
        { provide: MessagingService, useValue: messagingServiceSpy },
        { provide: ConversationService, useValue: conversationServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: CryptoService, useValue: cryptoServiceSpy },
        { provide: FriendService, useValue: friendServiceSpy },
        { provide: Router, useValue: routerSpy },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { queryParamMap: convertToParamMap({}) },
            queryParamMap: queryParamMap$.asObservable(),
          },
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
    authServiceSpy.reloadCurrentUser.mockResolvedValue(null);
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
    expect(component.conversations[0].name).toBe('bob');
    expect(component.conversations[0].id).toBe('conv-1');
    expect(component.conversations[0].messageLifetime).toBe(1440);
  });

  it('should call messagingService.connect() on init', () => {
    expect(messagingServiceSpy.connect).toHaveBeenCalled();
  });

  it('should open the room-key modal immediately from the chatWith route param', async () => {
    queryParamMap$.next(convertToParamMap({ chatWith: 'bob' }));
    await fixture.whenStable();

    expect(component.modal).toEqual({ type: 'create-room-key', memberUsernames: ['bob'] });
    expect(component.roomKeyInput).toBe('generated-room-key');
    expect(routerSpy.navigate).toHaveBeenCalled();
  });

  it('startNewConversation() should load friends and open the member picker', async () => {
    await component.startNewConversation();

    expect(friendServiceSpy.getFriends).toHaveBeenCalled();
    expect(component.modal).toEqual({ type: 'select-conversation-members' });
    expect(component.availableFriends.length).toBe(2);
  });

  it('continueCreateConversation() should require at least one selected friend', () => {
    component.modal = { type: 'select-conversation-members' };

    component.continueCreateConversation();

    expect(component.createConversationError).toContain('Please select at least one friend');
    expect(conversationServiceSpy.createConversation).not.toHaveBeenCalled();
  });

  it('continueCreateConversation() should open a room-key modal with the selected friends', () => {
    component.selectedConversationMembers = ['bob', 'carol'];

    component.continueCreateConversation();

    expect(component.modal).toEqual({ type: 'create-room-key', memberUsernames: ['bob', 'carol'] });
    expect(component.roomKeyInput).toBe('generated-room-key');
  });

  it('submitCreateConversation() should pass an optional group name for group chats', async () => {
    component.modal = { type: 'create-room-key', memberUsernames: ['bob', 'carol'] };
    component.groupConversationNameInput = 'Weekend Plans';
    component.roomKeyInput = 'long-enough-room-key';
    conversationServiceSpy.createConversation.mockResolvedValue({ conversation_id: 'conv-group', created: true });
    cryptoServiceSpy.deriveConversationKey.mockResolvedValue({} as CryptoKey);

    await component.submitCreateConversation();

    expect(conversationServiceSpy.createConversation).toHaveBeenCalledWith(['bob', 'carol'], 'long-enough-room-key', 'Weekend Plans');
  });

  it('toggleConversationMember() should clear the group name when only one friend remains selected', () => {
    component.selectedConversationMembers = ['bob', 'carol'];
    component.groupConversationNameInput = 'Weekend Plans';

    component.toggleConversationMember('carol');

    expect(component.selectedConversationMembers).toEqual(['bob']);
    expect(component.groupConversationNameInput).toBe('');
  });

  it('leaveConversation() should remove the active conversation and clear the chat state', async () => {
    component.conversationId = 'conv-1';
    component.isConnected = true;
    component.messages = [{ id: 'msg-1', username: 'alice', time: 'Now', content: 'Hello', isMine: true, isSystem: false, attachments: [], profilePictureUrl: '' }];
    component.modal = { type: 'conversation-settings', convId: 'conv-1' };

    await component.leaveConversation();

    expect(conversationServiceSpy.leaveConversation).toHaveBeenCalledWith('conv-1');
    expect(component.conversations).toEqual([]);
    expect(component.conversationId).toBe('');
    expect(component.isConnected).toBe(false);
    expect(component.messages).toEqual([]);
    expect(component.modal).toEqual({ type: 'none' });
  });

  it('submitCreateConversation() should set roomKeyError when the key is too short', async () => {
    component.modal = { type: 'create-room-key', memberUsernames: ['bob'] };
    component.roomKeyInput = 'short';

    await component.submitCreateConversation();

    expect(component.roomKeyError).toContain('longer than 6');
    expect(conversationServiceSpy.createConversation).not.toHaveBeenCalled();
  });

  it('sendMessage() should do nothing when message and attachments are empty', async () => {
    component.newMessage = '   ';
    await component.sendMessage();
    expect(messagingServiceSpy.sendMessage).not.toHaveBeenCalled();
  });

  it('sendMessage() should set composerError when no conversationId is set', async () => {
    component.newMessage = 'hello';
    component.conversationId = '';
    await component.sendMessage();
    expect(component.composerError).toContain('Join or create');
  });

  it('sendMessage() should set composerError when socket is not connected', async () => {
    component.newMessage = 'hello';
    component.conversationId = 'conv-1';
    messagingServiceSpy.isConnected.mockReturnValue(false);
    await component.sendMessage();
    expect(component.composerError).toContain('Not connected');
  });

  it('sendMessage() should encrypt and send a text-only message', async () => {
    const mockKey = {} as CryptoKey;
    component.conversationId = 'conv-1';
    component.conversationKeys.set('conv-1', mockKey);
    component.newMessage = 'hello world';
    cryptoServiceSpy.encryptMessage.mockResolvedValue('encrypted-blob');
    messagingServiceSpy.isConnected.mockReturnValue(true);

    await component.sendMessage();

    expect(cryptoServiceSpy.encryptMessage).toHaveBeenCalledWith('hello world', mockKey);
    expect(messagingServiceSpy.sendMessage).toHaveBeenCalledWith('conv-1', 'encrypted-blob', expect.any(String));
    expect(component.newMessage).toBe('');
    expect(component.messages[component.messages.length - 1].id).toBe('');
    expect(component.messages[component.messages.length - 1].content).toBe('hello world');
    expect(component.messages[component.messages.length - 1].isMine).toBe(true);
  });

  it('onAttachmentsSelected() should queue files without sending them', () => {
    const file = new File(['hello'], 'hello.txt', { type: 'text/plain' });
    const input = document.createElement('input');
    Object.defineProperty(input, 'files', { value: [file] });

    component.onAttachmentsSelected({ target: input } as unknown as Event);

    expect(component.pendingAttachments.length).toBe(1);
    expect(component.pendingAttachments[0].fileName).toBe('hello.txt');
    expect(messagingServiceSpy.sendMessage).not.toHaveBeenCalled();
  });

  it('sendMessage() should encrypt and send queued files together with text', async () => {
    const mockKey = {} as CryptoKey;
    const file = new File(['hello'], 'hello.txt', { type: 'text/plain' });
    component.conversationId = 'conv-1';
    component.conversationKeys.set('conv-1', mockKey);
    component.newMessage = 'see attachment';
    component.pendingAttachments = [
      {
        id: 'att-1',
        file,
        fileName: 'hello.txt',
        mimeType: 'text/plain',
        size: file.size,
        isImage: false,
        isVideo: false,
      },
    ];
    cryptoServiceSpy.encryptMessage.mockResolvedValue('encrypted-rich-message');

    await component.sendMessage();

    const serializedPayload = JSON.parse(cryptoServiceSpy.encryptMessage.mock.calls[0][0]);
    expect(serializedPayload.type).toBe('rich_message');
    expect(serializedPayload.text).toBe('see attachment');
    expect(serializedPayload.attachments[0].file_name).toBe('hello.txt');
    expect(messagingServiceSpy.sendMessage).toHaveBeenCalledWith('conv-1', 'encrypted-rich-message', expect.any(String));
    expect(component.pendingAttachments.length).toBe(0);
    expect(component.messages[component.messages.length - 1].id).toBe('');
    expect(component.messages[component.messages.length - 1].attachments[0].fileName).toBe('hello.txt');
  });

  // it('deleteMessage() should call the API and remove the message from the list', async () => {
  //   component.messages = [
  //     {
  //       id: 'msg-1',
  //       username: 'Alice',
  //       time: 'Today, 1.00pm',
  //       content: 'hello',
  //       isMine: true,
  //       attachments: [],
  //     },
  //   ];

  //   await component.deleteMessage('msg-1', 0);

  //   expect(conversationServiceSpy.DeleteMessage).toHaveBeenCalledWith('msg-1');
  //   expect(component.messages).toHaveLength(0);
  // });

  // it('deleteMessage() should ignore unsaved optimistic messages with no id', async () => {
  //   component.messages = [
  //     {
  //       id: '',
  //       username: 'Alice',
  //       time: 'Today, 1.00pm',
  //       content: 'hello',
  //       isMine: true,
  //       attachments: [],
  //     },
  //   ];

  //   await component.deleteMessage('', 0);

  //   expect(conversationServiceSpy.DeleteMessage).not.toHaveBeenCalled();
  //   expect(component.messages).toHaveLength(1);
  // });

  it('closeModal() should reset modal state', () => {
    component.modal = { type: 'enter-room-key', convId: 'conv-1' };
    component.roomKeyInput = 'somekey';
    component.settingsError = 'something failed';
    component.closeModal();
    expect(component.modal.type).toBe('none');
    expect(component.roomKeyInput).toBe('');
    expect(component.settingsError).toBe('');
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
    expect(component.messageLifetime).toBe(1440);
  });

  it('submitRoomKey() should set roomKeyError when input is empty', async () => {
    component.modal = { type: 'enter-room-key', convId: 'conv-1' };
    component.roomKeyInput = '   ';
    await component.submitRoomKey();
    expect(component.roomKeyError).toBeTruthy();
  });

  it('sendMessage() should set composerError when no room key is cached for the conversation', async () => {
    component.newMessage = 'hello';
    component.conversationId = 'conv-1';
    await component.sendMessage();
    expect(component.composerError).toContain('room key');
  });

  it('openConversationSettings() should open the settings modal with the current lifetime', () => {
    component.conversationId = 'conv-1';
    component.messageLifetime = 10080;

    component.openConversationSettings();

    expect(component.modal).toEqual({ type: 'conversation-settings', convId: 'conv-1' });
    expect(component.selectedMessageLifetime).toBe(10080);
  });

  it('saveConversationSettings() should persist the selected lifetime and refresh messages', async () => {
    const loadMessagesSpy = vi.spyOn(component as any, 'loadMessages').mockResolvedValue(undefined);
    const refreshConversationListSpy = vi.spyOn(component as any, 'refreshConversationList').mockResolvedValue(undefined);

    component.conversationId = 'conv-1';
    component.conversations = [{ id: 'conv-1', name: 'Bob', lastMessage: '', lastMessageTime: '', messageLifetime: 1440 }];
    component.modal = { type: 'conversation-settings', convId: 'conv-1' };
    component.selectedMessageLifetime = 60;

    await component.saveConversationSettings();

    expect(conversationServiceSpy.setMessageLifetime).toHaveBeenCalledWith('conv-1', 60);
    expect(component.messageLifetime).toBe(60);
    expect(component.conversations[0].messageLifetime).toBe(60);
    expect(component.modal.type).toBe('none');
    expect(loadMessagesSpy).toHaveBeenCalledWith('conv-1');
    expect(refreshConversationListSpy).toHaveBeenCalled();
  });

  it('getMessageLifetimeLabel() should map preset values to labels', () => {
    expect(component.getMessageLifetimeLabel(60)).toBe('1 hour');
    expect(component.getMessageLifetimeLabel(0)).toBe('Never');
  });
  it('startEditingMessage() should open inline editing with the current message content', () => {
    component.startEditingMessage({
      id: 'msg-1',
      username: 'Alice',
      time: 'Today, 1.00pm',
      content: 'original text',
      isMine: true,
      isSystem: false,
      attachments: [],
      profilePictureUrl: '',
    });

    expect(component.editingMessageId).toBe('msg-1');
    expect(component.editDraft).toBe('original text');
  });

  it('saveEditedMessage() should encrypt and persist a text edit', async () => {
    const mockKey = {} as CryptoKey;
    component.conversationId = 'conv-1';
    component.conversationKeys.set('conv-1', mockKey);
    messagingServiceSpy.isConnected.mockReturnValue(true);
    cryptoServiceSpy.encryptMessage.mockResolvedValue('edited-ciphertext');

    const message = {
      id: 'msg-1',
      username: 'Alice',
      time: 'Today, 1.00pm',
      content: 'original text',
      isMine: true,
      isSystem: false,
      attachments: [],
      profilePictureUrl: '',
    };

    component.startEditingMessage(message);
    component.editDraft = 'edited text';

    await component.saveEditedMessage(message);

    expect(cryptoServiceSpy.encryptMessage).toHaveBeenCalledWith('edited text', mockKey);
    expect(conversationServiceSpy.editMessage).toHaveBeenCalledWith('msg-1', 'edited-ciphertext');
    expect(message.content).toBe('edited text');
    expect(component.editingMessageId).toBeNull();
  });

});
