# SecretKeeper API Documentation


## Notes
- Most requests use `credentials: 'include'` as the app expects a session cookie for authenticated endpoints
- `CryptoService` is client-side only and does not call the backend
- `MessagingService` uses WebSocket

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

