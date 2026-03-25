import { Injectable } from '@angular/core';

export interface CreateConversationResponse {
  conversation_id: string;
}

export interface ConversationSummary {
  id: string;
  name: string;
  last_message: string;
  last_message_time: number;
}

@Injectable({
  providedIn: 'root',
})
export class ConversationService {

  // Creates a new conversation
  async createConversation(memberIds: string[]): Promise<string> {
    const response = await fetch('http://localhost:8080/api/conversations/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ member_ids: memberIds }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to create conversation: ${text}`);
    }

    const data: CreateConversationResponse = await response.json();
    return data.conversation_id;
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
}