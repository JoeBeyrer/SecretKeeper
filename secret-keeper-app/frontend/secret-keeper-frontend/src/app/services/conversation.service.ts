import { Injectable } from '@angular/core';

export interface CreateConversationResponse {
  conversation_id: string;
  created: boolean;
}

export interface ConversationSummary {
  id: string;
  name: string;
  last_message: string;
  last_message_time: number;
  message_lifetime?: number;
  member_count?: number;
  profile_picture_url?: string;
  other_username?: string;
}

export interface ConversationMemberSummary {
  user_id: string;
  username: string;
  display_name: string;
  profile_picture_url: string;
  friendship_status: 'self' | 'friend' | 'pending_outgoing' | 'pending_incoming' | 'none';
}

@Injectable({
  providedIn: 'root',
})
export class ConversationService {
  async createConversation(memberIds: string[], roomKey: string, groupName: string = ''): Promise<CreateConversationResponse> {
    const response = await fetch('http://localhost:8080/api/conversations/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ member_ids: memberIds, room_key: roomKey, group_name: groupName }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to create conversation: ${text}`);
    }

    return response.json();
  }

  async getConversations(): Promise<ConversationSummary[]> {
    const response = await fetch('http://localhost:8080/api/conversations/get', {
      credentials: 'include',
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to load conversations: ${text}`);
    }

    return response.json();
  }

  async getConversationMembers(conversationId: string): Promise<ConversationMemberSummary[]> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/members`, {
      credentials: 'include',
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to load conversation members: ${text}`);
    }

    return response.json();
  }

  async getMessages(conversationId: string): Promise<any[]> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/messages`, {
      credentials: 'include',
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to load messages: ${text}`);
    }

    return response.json();
  }

  async verifyRoomKey(conversationId: string, roomKey: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/verify-room-key`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ room_key: roomKey }),
    });

    if (response.ok) {
      return;
    }

    const text = await response.text();
    if (response.status === 404) {
      throw new Error('ROOM_KEY_VERIFIER_NOT_SET');
    }

    throw new Error(text || 'Failed to verify room key.');
  }

  async claimRoomKey(conversationId: string): Promise<string> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/claim-room-key`, {
      method: 'POST',
      credentials: 'include',
    });

    if (response.ok) {
      const data = await response.json();
      return data.room_key;
    }

    const text = await response.text();
    if (response.status === 404) {
      throw new Error('ROOM_KEY_NOT_AVAILABLE');
    }

    throw new Error(text || 'Failed to retrieve room key.');
  }

  async updateGroupName(conversationId: string, groupName: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/group-name`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ group_name: groupName }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Failed to update group name.');
    }
  }

  async removeConversationMembers(conversationId: string, memberIds: string[]): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/members/remove`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ member_ids: memberIds }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Failed to remove conversation members.');
    }
  }

  async addConversationMembers(conversationId: string, memberIds: string[], roomKey: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/members/add`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ member_ids: memberIds, room_key: roomKey }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Failed to add conversation members.');
    }
  }

  async leaveConversation(conversationId: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/leave`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || 'Failed to leave conversation.');
    }
  }

  async setMessageLifetime(conversationId: string, lifetime: number): Promise<void> {
    console.log('[Lifetime] calling API for', conversationId, 'with', lifetime);
    const response = await fetch(`http://localhost:8080/api/conversations/${conversationId}/lifetime`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ message_lifetime: lifetime }),
    });
    console.log('[Lifetime] response status:', response.status);
    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to set message lifetime: ${text}`);
    }
  }
  async toggleReaction(messageId: string, emoji: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/messages/${messageId}/react`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ emoji }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to toggle reaction: ${text}`);
    }
  }

  async editMessage(messageId: string, ciphertext: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/messages/${messageId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ ciphertext }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to edit message: ${text}`);
    }
  }

  async deleteMessage(messageId: string): Promise<void> {
    const response = await fetch(`http://localhost:8080/api/messages/${messageId}`, {
      method: 'DELETE',
      credentials: 'include',
    });
    console.log('[message deletion] response status:', response.status);
    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to delete message: ${text}`);
    }
  }
}
