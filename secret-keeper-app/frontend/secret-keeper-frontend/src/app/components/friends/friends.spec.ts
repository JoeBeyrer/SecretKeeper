import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { ChangeDetectorRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { vi } from 'vitest';

import { Friends } from './friends';
import { FriendService, FriendEntry } from '../../services/friend.service';
import { AuthService } from '../../services/auth.service';

describe('Friends', () => {
  let component: Friends;
  let fixture: ComponentFixture<Friends>;
  let friendServiceSpy: { [K in keyof FriendService]: ReturnType<typeof vi.fn> };
  let authServiceSpy: { loadCurrentUser: ReturnType<typeof vi.fn> };
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  const mockFriend: FriendEntry = {
    user_id: 'u1',
    username: 'bob',
    display_name: 'Bob Smith',
    accepted: true,
  };

  const mockIncoming: FriendEntry = {
    user_id: 'u2',
    username: 'carol',
    display_name: '',
    accepted: false,
    direction: 'incoming',
  };

  const mockOutgoing: FriendEntry = {
    user_id: 'u3',
    username: 'dave',
    display_name: '',
    accepted: false,
    direction: 'outgoing',
  };

  beforeEach(async () => {
    friendServiceSpy = {
      getFriends: vi.fn().mockResolvedValue([mockFriend]),
      getPendingRequests: vi.fn().mockResolvedValue([mockIncoming, mockOutgoing]),
      sendFriendRequest: vi.fn().mockResolvedValue(undefined),
      acceptRequest: vi.fn().mockResolvedValue(undefined),
      declineRequest: vi.fn().mockResolvedValue(undefined),
      removeFriend: vi.fn().mockResolvedValue(undefined),
    };

    authServiceSpy = {
      loadCurrentUser: vi.fn().mockResolvedValue({
        username: 'alice', email: 'a@b.com', display_name: '', bio: '', profile_picture_url: '',
      }),
    };

    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Friends, FormsModule],
      providers: [
        { provide: FriendService, useValue: friendServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        { provide: Router, useValue: routerSpy },
        ChangeDetectorRef,
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(Friends);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load friends and pending requests on init', () => {
    expect(component.friends.length).toBe(1);
    expect(component.friends[0].username).toBe('bob');
    expect(component.pendingRequests.length).toBe(2);
  });

  it('should filter incomingRequests correctly', () => {
    expect(component.incomingRequests.length).toBe(1);
    expect(component.incomingRequests[0].username).toBe('carol');
  });

  it('should filter outgoingRequests correctly', () => {
    expect(component.outgoingRequests.length).toBe(1);
    expect(component.outgoingRequests[0].username).toBe('dave');
  });

  it('should return display_name if set, otherwise username', () => {
    expect(component.displayName(mockFriend)).toBe('Bob Smith');
    expect(component.displayName(mockIncoming)).toBe('carol');
  });

  it('should set activeTab when setTab is called', async () => {
    component.setTab('requests');
    expect(component.activeTab).toBe('requests');
  });

  it('should clear messages when setTab is called', async () => {
    component.errorMessage = 'some error';
    component.successMessage = 'some success';
    component.setTab('add');
    expect(component.errorMessage).toBe('');
    expect(component.successMessage).toBe('');
  });

  it('should call sendFriendRequest and set successMessage on success', async () => {
    component.addUsername = 'newuser';
    await component.sendRequest();
    expect(friendServiceSpy.sendFriendRequest).toHaveBeenCalledWith('newuser');
    expect(component.successMessage).toContain('@newuser');
    expect(component.addUsername).toBe('');
  });

  it('should set errorMessage when sendFriendRequest fails', async () => {
    friendServiceSpy.sendFriendRequest.mockRejectedValue(new Error('User not found'));
    component.addUsername = 'ghost';
    await component.sendRequest();
    expect(component.errorMessage).toBe('User not found');
  });

  it('should not call sendFriendRequest when username is empty', async () => {
    component.addUsername = '   ';
    await component.sendRequest();
    expect(friendServiceSpy.sendFriendRequest).not.toHaveBeenCalled();
  });

  it('should call acceptRequest with correct username', async () => {
    await component.accept(mockIncoming);
    expect(friendServiceSpy.acceptRequest).toHaveBeenCalledWith('carol');
  });

  it('should call declineRequest with correct username', async () => {
    await component.decline(mockIncoming);
    expect(friendServiceSpy.declineRequest).toHaveBeenCalledWith('carol');
  });

  it('should call removeFriend with correct username', async () => {
    await component.remove(mockFriend);
    expect(friendServiceSpy.removeFriend).toHaveBeenCalledWith('bob');
  });

  it('should navigate to messaging with chatWith param when startChat is called', () => {
    component.startChat(mockFriend);
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/messaging'], { queryParams: { chatWith: 'bob' } });
  });

  it('should redirect to login if user is not authenticated', async () => {
    authServiceSpy.loadCurrentUser.mockResolvedValue(null);
    const newFixture = TestBed.createComponent(Friends);
    await newFixture.whenStable();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });

  it('should track isActing correctly during actions', async () => {
    let resolveAccept!: () => void;
    friendServiceSpy.acceptRequest.mockReturnValue(new Promise<void>(r => { resolveAccept = r; }));
    const promise = component.accept(mockIncoming);
    expect(component.isActing('carol')).toBe(true);
    resolveAccept();
    await promise;
    expect(component.isActing('carol')).toBe(false);
  });
});
