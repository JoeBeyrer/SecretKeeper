import { Injectable } from '@angular/core';

export interface CreateConversationResponse {
  conversation_id: string;
}

@Injectable({
  providedIn: 'root',
})
export class ConversationService {

  // Creates a new conversation
  async createConversation(memberIds: string[]): Promise<string> {
    const response = await fetch('http://localhost:8080/api/conversations', {
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
}