# Sprint 2 Report

## User Stories
1. As a user, I want to be able to login, so that my messages and data are associated with my personal account
2. As a user, I want my conversations to be private, so that only intended recipients can read them
3. As a user, I want to be able to search for other users, so that I may message certain people
4. As a user, I want to be able to add friends, so that I can maintain and start conversations with my known associates
5. As a user, I want to be able to start a conversation, so that I can message a specific friend
6. As a user, I want to be able to send images, so that I am not limited to text data
7. As a user, I want to be able to send videos, so that I am not limited to text data
8. As a user, I want to be able to modify my account information, so that I can ensure updated and accurate information
9. As a user, I want my account credentials to be stored securely, so that my account cannot be compromised
10. As a user, I want to have a profile picture, so that I can distinguish my account
11. As a user, I want my messages to disappear after a certain amount of time, so that my chat history remains secure and private
12. As a user, I want to delete a message I sent, so that I can remove incorrect or sensitive information
13. As a user, I want to log out of my account, so that others cannot access my messages on my device
14. As a user, I want to see a list of my conversations, so that I can quickly navigate between chats
15. As a user, I want to block another user, so that they cannot contact me
16. As a user, I want to be able to edit my messages, so that I can rectify incorrect or sensitive information
17. As a user, I want to implement password reset option
18. As a user, I want to implement group message chats
19. As a user, I want to be able to create an account, so that I can login and use SecretKeeper
20. As a user, I want to be able to send PDF files, so that I can transmit documents with important data
21. As a user, I want to be able to “mute” a chat so that I can more easily ignore messages from it
22. As a user, I want to access the app through the web, so that I can conveniently interface from any device

## Planned Issues
We plan to implement functionality to support user stories 2, 4, 8, and 14. In addition, we plan to complete frontend-backend integration for user stories 1, 5, 9, 17, 18, 19, and 22, and create both backend and frontend testing frameworks.
### Private Conversations and Encryption
- Complete frontend-backend integration for creating conversations, loading conversation lists, and loading message history
- Add room key based conversation access flow for creating and opening chats
- Verify room keys for existing conversations and support one-time room key retrieval for recipients
- Encrypt and decrypt chat messages on the client before sending and after receiving them
### Friends
- Implement backend and frontend support for loading friends and pending requests
- Support sending, accepting, declining, and removing friend requests
- Use friend relationships to support easier conversation creation
### Profile and Account Information
- Integrate frontend profile views with backend profile endpoints
- Support account information such as display name, bio, email, and profile picture
- Add support for modifying profile and account information
### Conversation List and Navigation
- Render a live conversation list from backend data
- Show conversation names, previews, and timestamps in the messaging sidebar
- Improve the overall messaging page layout and navigation flow for easier chat access
### Authentication and Web Integration
- Complete frontend-backend integration for login, signup, password reset, logout, and authenticated sessions
- Ensure authenticated requests consistently use session cookies across the web app
- Continue supporting browser-based access to SecretKeeper through the Angular frontend
### Group Conversations
- Extend conversation creation flow to support multiple members
- Ensure grouped conversations can be stored, loaded, and displayed through the same messaging infrastructure
### Testing
- Add backend unit tests for database and handler logic
- Add frontend tests for authentication, profile, friend, and messaging flows
- Expand frontend unit testing beyond the existing service and component scaffolds

## Successfully Completed
- Authentication backend routes are implemented for register, login, logout, email verification, email-change verification, and password reset flow
- Conversation backend routes are implemented for creating conversations, loading conversation lists, loading message history, verifying room keys, and one-time room key claiming
- Friends backend routes are implemented for loading friends, loading pending requests, sending requests, accepting requests, declining requests, and removing friends
- Profile backend routes are implemented for loading profile data, updating profile data, uploading profile pictures, and updating account information
- WebSocket messaging backend is implemented for real-time chat delivery
- Database support is implemented for users, sessions, conversations, friendship data, encrypted message storage, and room-key verification state
- Frontend-backend integration now exists for login, signup, password reset, profile loading, friends management, conversation loading, and messaging
- Real-time messaging is working through the WebSocket connection between the Angular frontend and Go backend
- Room and chat encryption is implemented on the client using a conversation key derived from the room key, and messages are encrypted before send and decrypted after receipt
- Conversation access flow now supports room key verification for existing chats and one-time room key retrieval for recipients
- Conversation list UI is implemented with names, encrypted-message previews, and timestamps in the messaging sidebar
- Key-management is implemented in the backend and frontend for storing user keys, retrieving public keys, saving encrypted conversation keys, and loading stored conversation keys
- Backend unit tests exist for database and handler coverage
- Frontend test scaffolds exist for the app, services, and several components

## Incomplete / Carried Over
All issues that were incomplete are due to time constraints. The team decided to continue prioritizing core messaging, privacy, authentication, and account functionality first for the MVP, while carrying lower-priority or unfinished features into the next sprint.
- User search functionality has not been completed yet
- Sending images has not been implemented yet
- Sending videos has not been implemented yet
- Disappearing messages have not been implemented yet
- Message deletion has not been implemented yet
- Blocking users has not been implemented yet
- Message editing has not been implemented yet
- Sending PDF files has not been implemented yet
- Chat muting has not been implemented yet
- Additional frontend and backend test coverage




---

# SecretKeeper API Documentation

## ApiService

### `get<T>(endpoint: string): Observable<T>`
- **Params:** `endpoint`
- **Input:** endpoint path string
- **Output:** `Observable<T>`
- **Description:** Generic HTTP GET helper for `/api/{endpoint}`

### `post<T>(endpoint: string, data: any): Observable<T>`
- **Params:** `endpoint`, `data`
- **Input:** endpoint path and request body
- **Output:** `Observable<T>`
- **Description:** Generic HTTP POST helper

### `put<T>(endpoint: string, data: any): Observable<T>`
- **Params:** `endpoint`, `data`
- **Input:** endpoint path and request body
- **Output:** `Observable<T>`
- **Description:** Generic HTTP PUT helper

### `delete<T>(endpoint: string): Observable<T>`
- **Params:** `endpoint`
- **Input:** endpoint path
- **Output:** `Observable<T>`
- **Description:** Generic HTTP DELETE helper


## AuthService

### `loadCurrentUser(): Promise<UserProfile | null>`
- **Params:** none
- **Input:** none
- **Output:** `UserProfile | null`
- **Description:** Loads the current logged-in user's profile from `/api/profile` and caches it

### `getCurrentUser(): UserProfile | null`
- **Params:** none
- **Input:** none
- **Output:** cached `UserProfile | null`
- **Description:** Returns the cached current user

### `clearCurrentUser(): void`
- **Params:** none
- **Input:** none
- **Output:** `void`
- **Description:** Clears the cached current user

### `UserProfile`
- `username: string`
- `email: string`
- `display_name: string`
- `bio: string`
- `profile_picture_url: string`


## ConversationService

### `createConversation(memberIds: string[], roomKey: string): Promise<CreateConversationResponse>`
- **Params:** `memberIds`, `roomKey`
- **Input:** `{ member_ids, room_key }`
- **Output:** `{ conversation_id, created }`
- **Description:** Creates a conversation or returns an existing one

### `getConversations(): Promise<ConversationSummary[]>`
- **Params:** none
- **Input:** none
- **Output:** list of conversation summaries
- **Description:** Loads the user's conversation list

### `getMessages(conversationId: string): Promise<any[]>`
- **Params:** `conversationId`
- **Input:** conversation ID in URL
- **Output:** message array
- **Description:** Loads messages for a conversation

### `verifyRoomKey(conversationId: string, roomKey: string): Promise<void>`
- **Params:** `conversationId`, `roomKey`
- **Input:** `{ room_key }`
- **Output:** `void`
- **Description:** Verifies a room key for a conversation. Throws `ROOM_KEY_VERIFIER_NOT_SET` on 404

### `claimRoomKey(conversationId: string): Promise<string>`
- **Params:** `conversationId`
- **Input:** conversation ID in URL
- **Output:** room key string
- **Description:** Claims a one-time room key for a conversation. Throws `ROOM_KEY_NOT_AVAILABLE` on 404

### `CreateConversationResponse`
- `conversation_id: string`
- `created: boolean`

### `ConversationSummary`
- `id: string`
- `name: string`
- `last_message: string`
- `last_message_time: number`


## CryptoService

### `generateRoomKey(): string`
- **Params:** none
- **Input:** none
- **Output:** random room key string
- **Description:** Generates a random room key in the browser

### `deriveConversationKey(passphrase: string, convId: string): Promise<CryptoKey>`
- **Params:** `passphrase`, `convId`
- **Input:** room key / passphrase and conversation ID
- **Output:** `CryptoKey`
- **Description:** Derives a stable AES key using PBKDF2 and the conversation ID as salt

### `encryptMessage(plaintext: string, convKey: CryptoKey): Promise<string>`
- **Params:** `plaintext`, `convKey`
- **Input:** plaintext message and derived key
- **Output:** base64 encrypted string
- **Description:** Encrypts a message with AES-GCM

### `decryptMessage(encryptedB64: string, convKey: CryptoKey): Promise<string>`
- **Params:** `encryptedB64`, `convKey`
- **Input:** base64 ciphertext and derived key
- **Output:** plaintext string
- **Description:** Decrypts a message with AES-GCM



## FriendService

### `getFriends(): Promise<FriendEntry[]>`
- **Params:** none
- **Input:** none
- **Output:** friend list
- **Description:** Loads accepted friends

### `getPendingRequests(): Promise<FriendEntry[]>`
- **Params:** none
- **Input:** none
- **Output:** pending friend requests
- **Description:** Loads pending friend requests

### `sendFriendRequest(username: string): Promise<void>`
- **Params:** `username`
- **Input:** `{ username }`
- **Output:** `void`
- **Description:** Sends a friend request

### `acceptRequest(username: string): Promise<void>`
- **Params:** `username`
- **Input:** `{ username }`
- **Output:** `void`
- **Description:** Accepts a friend request

### `declineRequest(username: string): Promise<void>`
- **Params:** `username`
- **Input:** `{ username }`
- **Output:** `void`
- **Description:** Declines a friend request

### `removeFriend(username: string): Promise<void>`
- **Params:** `username`
- **Input:** `{ username }`
- **Output:** `void`
- **Description:** Removes a friend

### `FriendEntry`
- `user_id: string`
- `username: string`
- `display_name: string`
- `accepted: boolean`
- `direction?: string`



## KeyService

### `saveKeys(publicKey: string, encryptedPrivateKey: string): Promise<void>`
- **Params:** `publicKey`, `encryptedPrivateKey`
- **Input:** `{ public_key, encrypted_private_key }`
- **Output:** `void`
- **Description:** Saves the user's key pair data to the backend

### `getKeys(): Promise<{ public_key: string; encrypted_private_key: string }>`
- **Params:** none
- **Input:** none
- **Output:** public key and encrypted private key
- **Description:** Loads the current user's stored keys

### `getPublicKey(username: string): Promise<{ public_key: string; user_id: string }>`
- **Params:** `username`
- **Input:** username in URL
- **Output:** target user's public key and user ID
- **Description:** Fetches another user's public key

### `saveConversationKeys(convId: string, keys: { user_id: string; encrypted_key: string }[]): Promise<void>`
- **Params:** `convId`, `keys`
- **Input:** `{ keys }`
- **Output:** `void`
- **Description:** Saves encrypted conversation keys for conversation members

### `getConversationKey(convId: string): Promise<string>`
- **Params:** `convId`
- **Input:** conversation ID in URL
- **Output:** encrypted key string
- **Description:** Loads the current user's encrypted conversation key



## MessagingService

### `connect(): void`
- **Params:** none
- **Input:** none
- **Output:** `void`
- **Description:** Opens the WebSocket connection to `/ws`

### `sendMessage(conversationId: string, ciphertext: string): void`
- **Params:** `conversationId`, `ciphertext`
- **Input:** `{ type: 'send_message', conversation_id, ciphertext }`
- **Output:** `void`
- **Description:** Sends an encrypted message over WebSocket

### `disconnect(): void`
- **Params:** none
- **Input:** none
- **Output:** `void`
- **Description:** Closes the WebSocket connection

### `isConnected(): boolean`
- **Params:** none
- **Input:** none
- **Output:** `boolean`
- **Description:** Returns whether the socket is currently open

### `ngOnDestroy(): void`
- **Params:** none
- **Input:** none
- **Output:** `void`
- **Description:** Cleanup hook that disconnects the socket and completes the stream

### Incoming WebSocket Message
- `type: string`
- `conversation_id: string`
- `ciphertext: string`
- `sender_id: string`
- `display_name: string`

### Outgoing WebSocket Message
- `type: string`
- `conversation_id: string`
- `ciphertext: string`

---

## Running Frontend Tests

Frontend end-to-end tests are written with Cypress and live in `secret-keeper-app/frontend/secret-keeper-frontend/cypress/e2e/`.

Before running the tests, make sure the Angular dev server and Go backend are both running.

To open the Cypress test runner:

```
cd secret-keeper-app/frontend/secret-keeper-frontend
npx cypress open --e2e --config baseUrl=http://localhost:4200
```

Select **E2E Testing**, choose a browser, then click any spec file to run it. Here is the list of tests:

- go_to_signup.cy.ts — User navigates from login page to signup page.
- signup_username_too_short.cy.ts — Signup rejects username shorter than required minimum.
- signup_password_too_short.cy.ts — Signup rejects password shorter than required minimum.
- signup_accepts_valid_credentials.cy.ts — Signup accepts valid username and password combination.
- go_to_password_reset.cy.ts — User navigates from login page to password reset page.
- use_password_reset.cy.ts — User submits email for password reset and invalid email is rejected.
- messaging_load_page.cy.ts — Messaging page loads with sidebar and empty conversation state.
- messaging_create_conversation.cy.ts — User creates a new conversation and room key modal appears.
- messaging_send_message.cy.ts — User sends a message in a conversation and it appears in the chat.
- messaging_conversation_list.cy.ts — Conversation list renders with conversation items.
- messaging_nav_to_profile.cy.ts — User navigates to profile page from messaging sidebar.
- messaging_nav_to_friends.cy.ts — User navigates to friends page from messaging sidebar.
- profile_load_page.cy.ts — Profile page loads and displays user's username and email.
- profile_update_display_name.cy.ts — User updates their display name and change is saved.
- profile_logout.cy.ts — User logs out and is redirected to login page.
- friends_load_page.cy.ts — Friends page loads with Friends, Requests, and Add Friend tabs.
- friends_send_request.cy.ts — User sends a friend request to another user.
- friends_view_requests.cy.ts — User views incoming friend requests on the Requests tab.
- friends_message_button.cy.ts — User adds a friend, friend accepts, then user opens chat from friends list.
