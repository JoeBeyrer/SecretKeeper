# Sprint 4 Report

## Demo Video
[Sprint 4 Demo Video]()

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
We plan to implement functionality to support user stories 18, 21, and 22. Additional features were also added to this sprint as initial application goals were met.
### Adding and Removing Friends from Group Conversations
- Add support for removing members from group conversations.
- Add support for adding friends to a group conversation.
- Add system messages when members are added to or removed from group conversations.
### Managing Group Conversations
- Create an all-in-one settings popup for group conversation management such as leaving group conversations and managing disappearing messages.
- Add support for sending friend requests to members of a group conversation.
- Add support for changing group conversations names.
- Add support for leaving conversations.
- Add system messages for modifications to group conversations such as name changes and members leaving.
### Creating Group Conversations
- Implement frontend group conversation creation.
- Allow users to add one or more friends during chat creation, optionally create conversation names, and set group conversation room keys.



## Successfully Completed
- Create group conversations.
- Set group conversation room keys.
- Create group conversation names.
- Send friend requests to other members of a group conversation.
- Add members to a group conversation.
- Remove members from group conversations.
- Change group conversation names.
- Leave conversations.
- Group conversation management settings all in one place.
- View all members of group conversation.


## Incomplete / Carried Over
None.

---

# Testing
## Running Frontend Tests

### Cypress End-to-End Tests

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
- messaging_conversation_list.cy.ts — Conversation list renders with conversation items.
- messaging_edit_message.cy.ts — User edits a sent message inline from the message actions menu.
- messaging_send_attachment.cy.ts — User queues an attachment before sending and sees it render in the chat after sending.
- messaging_nav_to_profile.cy.ts — User navigates to profile page from messaging sidebar.
- messaging_nav_to_friends.cy.ts — User navigates to friends page from messaging sidebar.
- profile_update_display_name.cy.ts — User updates their display name and change is saved.
- profile_logout.cy.ts — User logs out and is redirected to login page.
- friends_load_page.cy.ts — Friends page loads with Friends, Requests, and Add Friend tabs.
- friends_send_request.cy.ts — User sends a friend request to another user.
- friends_view_requests.cy.ts — User views incoming friend requests on the Requests tab.

## Frontend Unit Tests

Unit tests are written with Jasmine and run with Karma inside the Angular project.

To run all frontend unit tests:

```
cd secret-keeper-app/frontend/secret-keeper-frontend
npx ng test --watch=false
```

**Component Tests** (`src/app/components/*/`)

friends/friends.spec.ts
- should create
- should load friends and pending requests on init
- should filter incomingRequests correctly
- should filter outgoingRequests correctly
- should return display_name if set, otherwise username
- should set activeTab when setTab is called
- should clear messages when setTab is called
- should call sendFriendRequest and set successMessage on success
- should set errorMessage when sendFriendRequest fails
- should not call sendFriendRequest when username is empty
- should call acceptRequest with correct username
- should call declineRequest with correct username
- should call removeFriend with correct username
- should navigate to messaging with chatWith param when startChat is called
- should redirect to login if user is not authenticated
- should track isActing correctly during actions

login/login.spec.ts
- should create
- should initialize form with empty username and password
- should set errorMessage when submitting invalid form
- should mark form invalid when username is too short
- should mark form invalid when password is too short
- should mark form valid with correct username and password
- should navigate to /signup when goToSignup is called
- should navigate to /reset-password when goToForgotPassword is called
- should set successMessage when verified query param is true

messaging/messaging.spec.ts
- should create
- should redirect to /login if user is not authenticated
- should set currentUsername and currentDisplayName from the authenticated user
- should load and map conversations on init
- should call messagingService.connect() on init
- startNewConversation() should set errorMessage when username is empty
- sendMessage() should do nothing when newMessage is empty
- sendMessage() should set errorMessage when no conversationId is set
- sendMessage() should set errorMessage when socket is not connected
- sendMessage() should encrypt and send the message
- closeModal() should reset modal state
- goTo() should navigate to the given page
- getActiveConversationName() should return name of current conversation
- getActiveConversationName() should return truncated id when conversation is unknown
- selectConversation() should prompt for room key when claimRoomKey returns NOT_AVAILABLE
- submitRoomKey() should set roomKeyError when input is empty
- sendMessage() should set errorMessage when no room key is cached for the conversation

password-reset/password-reset.spec.ts
- should create
- should start in request state
- should set requestError when email is invalid
- should POST to password-reset/request and set requestMessage on success
- should set requestError on POST failure
- goToLogin() should navigate to /login
- requestNewLink() should navigate to /reset-password
- should enter validating state and set reset state when token is valid
- should set expired state when token is invalid
- should set expired state when validation request errors
- should show reset page after valid token
- should set resetError when passwords do not match
- should set resetError when password is too short
- should POST confirm and set success state on success
- should set resetError with 422 expired link message
- should set generic resetError on other confirm errors

profile/profile.spec.ts
- should create
- should load profile data on init
- should populate profileForm with loaded profile data
- should navigate to /login when loadProfile returns 401
- should set errorMessage when loadProfile returns non-401 error
- onSave() should set errorMessage when profileForm is invalid
- onSave() should PUT profile and set successMessage on success
- onSave() should set errorMessage on save failure
- onSaveAccount() should set accountErrorMessage when all fields are empty
- onSaveAccount() should set accountErrorMessage when passwords do not match
- onSaveAccount() should PUT account and reset form on success
- onSaveAccount() should set accountErrorMessage with 409 conflict
- onLogout() should POST to /api/logout and navigate to /login
- onLogout() should navigate to /login even on logout error
- goToMessaging() should navigate to /messaging
- should set accountSuccessMessage when email_updated query param is true
- onPictureSelected() should set errorMessage for disallowed file type

signup/signup.spec.ts
- should create
- should initialize the form with empty fields
- should set errorMessage when username is missing
- should set errorMessage when username is too short
- should set errorMessage when email is missing
- should set errorMessage when email is invalid
- should set errorMessage when password is missing
- should set errorMessage when password is too short
- should set errorMessage when passwords do not match
- should navigate to /login when goToLogin is called

**Service Tests** (`src/app/services/`)

api.spec.ts
- should be created
- get() should send a GET request to the correct URL
- post() should send a POST request with the given body
- put() should send a PUT request with the given body
- delete() should send a DELETE request to the correct URL

auth.service.spec.ts
- should be created
- getCurrentUser() should return null before any load
- clearCurrentUser() should set currentUser back to null
- loadCurrentUser() should return cached user on second call without re-fetching
- loadCurrentUser() should return null when response is not ok
- loadCurrentUser() should return null when fetch throws
- loadCurrentUser() should return and cache the user profile on success

conversation.service.spec.ts
- should be created
- createConversation() should POST /conversations/create with member_ids and room_key
- createConversation() should throw on non-ok response
- getConversations() should GET /conversations/get and return list
- getConversations() should throw on error
- getMessages() should GET /conversations/:id/messages
- verifyRoomKey() should POST /conversations/:id/verify-room-key and resolve on ok
- verifyRoomKey() should throw ROOM_KEY_VERIFIER_NOT_SET on 404
- verifyRoomKey() should throw generic error on other failures
- claimRoomKey() should POST /conversations/:id/claim-room-key and return the room_key
- claimRoomKey() should throw ROOM_KEY_NOT_AVAILABLE on 404
- claimRoomKey() should throw on other errors
- editMessage() should PATCH /messages/:id with ciphertext

crypto.service.spec.ts
- should be created
- generateRoomKey() should return a non-empty base64 string
- generateRoomKey() should return a different key on each call
- generateRoomKey() should produce a valid base64-decodable string of 18 raw bytes
- deriveConversationKey() should return a CryptoKey
- deriveConversationKey() should produce the same key for the same passphrase and convId
- deriveConversationKey() should produce different keys for different passphrases
- deriveConversationKey() should produce different keys for different conversation IDs
- encryptMessage() should encrypt a message to a non-empty base64 string
- decryptMessage() should decrypt back to the original plaintext
- encryptMessage() should produce different ciphertexts for the same message (random IV)
- encryptMessage()/decryptMessage() should encrypt and decrypt an empty string
- encryptMessage()/decryptMessage() should encrypt and decrypt a long unicode message
- decryptMessage() should reject decryption when the ciphertext is tampered with
- decryptMessage() should reject decryption when the wrong key is used

friend.service.spec.ts
- should be created
- getFriends() should GET /friends and return the list
- getFriends() should throw when response is not ok
- getPendingRequests() should GET /friends/requests and return list
- sendFriendRequest() should POST /friends/request with correct username
- sendFriendRequest() should throw when response is not ok
- acceptRequest() should POST /friends/accept with correct username
- declineRequest() should POST /friends/decline with correct username
- removeFriend() should DELETE /friends/remove with correct username
- removeFriend() should throw when response is not ok

key.service.spec.ts
- should be created
- saveKeys() should POST /keys/save with the correct payload
- saveKeys() should throw when response is not ok
- getKeys() should GET /keys/get and return public and private keys
- getKeys() should throw when response is not ok
- getPublicKey() should GET /users/:username/public-key and return public_key and user_id
- getPublicKey() should throw when the user is not found
- saveConversationKeys() should POST /conversations/:id/keys with the keys array
- saveConversationKeys() should throw when response is not ok
- getConversationKey() should GET /conversations/:id/key and return the encrypted_key string
- getConversationKey() should throw when response is not ok

messaging.service.spec.ts
- should be created
- connect() should create a WebSocket pointing to the correct URL
- connect() should not create a second socket if already OPEN
- connect() should set socket to null on close
- isConnected() should return false before connect() is called
- isConnected() should return true when socket is OPEN
- isConnected() should return false after disconnect()
- sendMessage() should send a properly structured JSON payload
- sendMessage() should include client_message_id when provided
- sendMessage() should not throw when socket is not open
- sendMessage() should not send when socket has been disconnected
- disconnect() should close the socket
- disconnect() should not throw when called before connect()
- messages$ should emit an IncomingMessage when a new_message event is received
- messages$ should not emit for messages with a type other than new_message
- messages$ should not throw when the incoming message is malformed JSON
- messages$ should emit multiple messages in order
- ngOnDestroy() should disconnect and complete the subject without errors

## Backend Tests

To run backend tests:

```
cd secret-keeper-app/backend
go test ./tests/... -v
```

**database_test.go**
- Test_init_db_func — Verifies database schema initializes correctly.
- Test_create_session_func — Verifies session creation and retrieval.
- Test_delete_session_func — Verifies session deletion.
- Test_send_friend_request_func — Verifies friend request creation in DB.
- Test_accept_friend_request_func — Verifies friend request acceptance in DB.
- Test_decline_friend_request_func — Verifies friend request decline in DB.
- Test_remove_friend_func — Verifies friend removal in DB.
- Test_get_friends_func — Verifies accepted friends are returned.
- Test_get_pending_requests_func — Verifies pending requests are returned.
- Test_friendship_exists_func — Verifies friendship existence checks.
- Test_get_user_id_by_username_func — Verifies user lookup by username.

**handlers_test.go**
- Test_register_handler_func — Invalid password length, empty username, duplicate username, valid registration.
- Test_verify_email_handler_func — Missing token, expired token, valid token verification.
- Test_login_handler_func — Invalid credentials, unverified email, valid login.

**auth_friends_test.go**
- Test_send_friend_request_handler — Unauthorized, user not found, self-add, duplicate, valid request.
- Test_accept_friend_request_handler — Unauthorized, user not found, valid acceptance.
- Test_decline_friend_request_handler — Unauthorized, user not found, valid decline.
- Test_remove_friend_handler — Unauthorized, user not found, valid removal.
- Test_get_friends_handler — Unauthorized, empty list, populated list.
- Test_get_pending_requests_handler — Unauthorized, incoming and outgoing requests.

**profile_passwordreset_test.go**
- Test_forgot_password_handler — Empty email, unknown email, unverified email, valid request with rate limit.
- Test_validate_reset_token_handler — Missing token, expired token, valid token.
- Test_reset_password_handler — Missing token, expired token, password too short, valid reset.
- Test_get_profile_handler — Unauthorized, valid profile fetch, auto-creates blank profile row.
- Test_update_profile_handler — Unauthorized, display name too long, bio too long, valid update.
- Test_update_account_handler — Unauthorized, duplicate username/email conflict, valid username and email update.
- Test_verify_email_change_handler — Missing token, expired token, valid verification and redirect.
- Test_logout_handler — Valid logout clears session cookie.
- Test_upload_profile_picture_handler — Unauthorized, oversized file, invalid file type, valid upload.

**conversations_test.go**
- Test_create_conversation_handler — Missing room key, unknown member, valid creation, duplicate returns existing, unauthorized.
- Test_get_conversations_handler — Empty list, list after creation, unauthorized.
- Test_get_conversation_messages_handler — Empty message list, non-member forbidden access.
- Test_verify_conversation_room_key_handler — Correct key returns 204, wrong key returns 401, empty key returns 400.
- Test_claim_conversation_room_key_handler — Recipient claims key, double-claim returns 404, non-member returns 403.
- Test_edit_message_handler — Missing ciphertext returns 400, editing another user's message returns 404, valid edit updates ciphertext.
- Test_get_conversation_messages_handler_attachment_ciphertext — Attachment-message ciphertext is returned unchanged by the messages endpoint.

**keys_test.go**
- Test_save_keys_handler — Empty keys return 400, valid save returns 204, unauthorized returns 401.
- Test_get_keys_handler — No keys returns 404, returns key pair after save, unauthorized returns 401.
- Test_get_public_key_handler — No keys returns 404, returns key after save, unknown username returns 404, unauthorized returns 401.

**hub_test.go**
- Test_hub_register_and_send — Registered client receives message.
- Test_hub_send_to_unregistered_user — Sending to unknown user does not panic.
- Test_hub_unregister — Unregistered client no longer receives messages.
- Test_hub_register_multiple_tabs_same_user — Re-registering the same user adds a second client; both tabs receive messages; unregistering one tab leaves the other active.
- Test_hub_multiple_clients — Multiple clients each receive only their own messages.

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

### `editMessage(messageId: string, ciphertext: string): Promise<void>`
- **Params:** `messageId`, `ciphertext`
- **Input:** message ID in URL and `{ ciphertext }` body
- **Output:** `void`
- **Description:** Updates an existing encrypted message in place

### `deleteMessage(messageId: string): Promise<void>`
- **Params:** `messageId`
- **Input:** message ID in URL
- **Output:** `void`
- **Description:** Deletes a sent message. Only the sender can delete their own messages

### `setMessageLifetime(conversationId: string, lifetime: number): Promise<void>`
- **Params:** `conversationId`, `lifetime`
- **Input:** `{ message_lifetime }` — value in minutes (0 = never)
- **Output:** `void`
- **Description:** Sets the message expiry lifetime for a conversation. Allowed values are 60, 1440, 10080, 43200, 525600, or 0

### `toggleReaction(messageId: string, emoji: string): Promise<void>`
- **Params:** `messageId`, `emoji`
- **Input:** `{ emoji }`
- **Output:** `void`
- **Description:** Toggles an emoji reaction on a message. Adds the reaction if not present, removes it if already set

### `CreateConversationResponse`
- `conversation_id: string`
- `created: boolean`

### `ConversationSummary`
- `id: string`
- `name: string`
- `last_message: string`
- `last_message_time: number`
- `message_lifetime: number`


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

### `bytesToBase64(bytes: Uint8Array): string`
- **Params:** `bytes`
- **Input:** raw byte array
- **Output:** base64 string
- **Description:** Encodes attachment or message bytes into base64 for transport

### `base64ToArrayBuffer(base64: string): ArrayBuffer`
- **Params:** `base64`
- **Input:** base64 string
- **Output:** `ArrayBuffer`
- **Description:** Decodes base64 attachment data back into binary data



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

### `rescindRequest(username: string): Promise<void>`
- **Params:** `username`
- **Input:** `{ username }`
- **Output:** `void`
- **Description:** Cancels an outgoing friend request that has not yet been accepted

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

### `sendMessage(conversationId: string, ciphertext: string, clientMessageId?: string): void`
- **Params:** `conversationId`, `ciphertext`, `clientMessageId?`
- **Input:** `{ type: 'send_message', conversation_id, ciphertext, client_message_id? }`
- **Output:** `void`
- **Description:** Sends an encrypted text or attachment message over WebSocket

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
- `profile_picture_url: string`
- `message_id: string`
- `client_message_id?: string`

### Outgoing WebSocket Message
- `type: string`
- `conversation_id: string`
- `ciphertext: string`
- `client_message_id?: string`

### `RichMessagePayload`
- `version: 1`
- `type: 'rich_message'`
- `text: string`
- `attachments: RichMessageAttachmentPayload[]`

### `RichMessageAttachmentPayload`
- `file_name: string`
- `mime_type: string`
- `size: number`
- `data_b64: string`


---

# Backend API Routes

All protected routes require a valid `sk_session` cookie set at login.

## Auth Routes

### `POST /api/register`
Registers a new user. Creates the account in an unverified state and sends a verification email.
- **Body:** `{ username, email, password }`
- **Response:** `200 OK`

### `POST /api/login`
Logs in with username and password. Requires a verified email address.
- **Body:** `{ username, password }`
- **Response:** `200 OK` with `sk_session` cookie

### `POST /api/logout`
Clears the session cookie.
- **Response:** `200 OK`

### `GET /api/verify-email?token=`
Marks an account as verified using the token from the verification email. Redirects to `/login?verified=true`.

### `GET /api/verify-email-change?token=`
Confirms an email address change using the token from the change-verification email. Redirects to `/profile?email_updated=true`.

### `POST /api/password-reset/request`
Sends a password reset email if the address exists and is verified.
- **Body:** `{ email }`
- **Response:** `200 OK`

### `GET /api/password-reset/validate?token=`
Validates a password reset token without consuming it.
- **Response:** `200 OK` or `401 Unauthorized`

### `POST /api/password-reset/confirm`
Resets the password using a valid reset token.
- **Body:** `{ token, new_password }`
- **Response:** `200 OK`

## Profile Routes

### `GET /api/profile`
Returns the logged-in user's profile.
- **Response:** `{ username, email, display_name, bio, profile_picture_url }`

### `PUT /api/profile/update`
Updates display name and/or bio. Optionally clears the profile picture.
- **Body:** `{ display_name, bio, clear_picture? }`
- **Response:** `200 OK`

### `POST /api/profile/picture`
Uploads a profile picture. Accepts JPEG, PNG, GIF, or WebP up to 2 MB. Stores as a base64 data URL and broadcasts a `profile_updated` WebSocket event to all conversation members.
- **Body:** `multipart/form-data` with `picture` field
- **Response:** `{ message, profile_picture_url }`

### `GET /api/profile/by-username/{username}`
Returns the profile picture URL for a given username. Used to resolve pictures in the conversation view.
- **Response:** `{ profile_picture_url }`

### `PUT /api/account`
Updates the logged-in user's username, email, and/or password.
- **Body:** `{ new_username?, new_email?, new_password? }`
- **Response:** `200 OK`

## Conversation Routes

### `POST /api/conversations/create`
Creates a new conversation or returns an existing one between the same members. Notifies all members via WebSocket on creation.
- **Body:** `{ member_ids, room_key }`
- **Response:** `{ conversation_id, created }` — `201 Created` for new, `200 OK` for existing

### `GET /api/conversations/get`
Returns all conversations the logged-in user is a member of, with name, last message preview, timestamp, and message lifetime.
- **Response:** array of conversation summaries

### `GET /api/conversations/{id}/messages`
Returns the message history for a conversation. Requires membership.
- **Response:** array of messages with `ID`, `Ciphertext`, `Username`, `DisplayName`, `ProfilePictureURL`, `CreatedAt`, `ExpiresAt`, `Reactions`

### `POST /api/conversations/{id}/verify-room-key`
Verifies a room key against the stored hash without consuming it.
- **Body:** `{ room_key }`
- **Response:** `204 No Content` or `401 Unauthorized`

### `POST /api/conversations/{id}/claim-room-key`
Claims the one-time pending room key stored for the recipient. Can only be claimed once.
- **Response:** `{ room_key }` or `404 Not Found`

### `PATCH /api/conversations/{id}/lifetime`
Sets the message expiry lifetime for a conversation. Applies to all existing and future messages. Notifies members via WebSocket.
- **Body:** `{ message_lifetime }` — minutes: `60`, `1440`, `10080`, `43200`, `525600`, or `0` (never)
- **Response:** `204 No Content`

## Message Routes

### `PATCH /api/messages/{id}`
Edits a sent message. Only the original sender can edit. Broadcasts a `messages_updated` event to all conversation members.
- **Body:** `{ ciphertext }`
- **Response:** `204 No Content`

### `DELETE /api/messages/{id}`
Deletes a sent message. Only the original sender can delete. Broadcasts a `messages_updated` event to all conversation members.
- **Response:** `204 No Content`

### `POST /api/messages/{id}/react`
Toggles an emoji reaction on a message. Adds the reaction if not present, removes it if already set. Broadcasts a `messages_updated` event.
- **Body:** `{ emoji }`
- **Response:** `200 OK`

## Friends Routes

### `GET /api/friends`
Returns the logged-in user's accepted friends and pending requests.

### `GET /api/friends/requests`
Returns all pending incoming and outgoing friend requests.

### `POST /api/friends/request`
Sends a friend request to another user.
- **Body:** `{ username }`
- **Response:** `200 OK`

### `POST /api/friends/accept`
Accepts an incoming friend request.
- **Body:** `{ username }`
- **Response:** `200 OK`

### `POST /api/friends/decline`
Declines an incoming friend request.
- **Body:** `{ username }`
- **Response:** `200 OK`

### `DELETE /api/friends/rescind`
Cancels an outgoing friend request that has not yet been accepted.
- **Body:** `{ username }`
- **Response:** `200 OK`

### `DELETE /api/friends/remove`
Removes an existing friend.
- **Body:** `{ username }`
- **Response:** `200 OK`

## Block Routes

### `POST /api/blocks/block/{blockee_id}`
Blocks a user by their user ID. Blocked users cannot send messages through SecretKeeper. Also removes any existing friendship between the two users.
- **Body:** `{ blockee_id }`
- **Response:** `201 Created`

### `DELETE /api/blocks/unblock/{blockee_id}`
Unblocks a previously blocked user.
- **Response:** `204 No Content`

## User Routes

### `GET /api/users/search?q=`
Searches for users by username prefix. Excludes the calling user from results.
- **Query:** `q` — search string
- **Response:** array of `{ user_id, username, display_name }`

## Key Routes

### `POST /api/keys/save`
Saves the user's public key and encrypted private key.
- **Body:** `{ public_key, encrypted_private_key }`
- **Response:** `204 No Content`

### `GET /api/keys/get`
Returns the logged-in user's stored key pair.
- **Response:** `{ public_key, encrypted_private_key }`

### `GET /api/users/{username}/public-key`
Returns another user's public key.
- **Response:** `{ public_key, user_id }`

### `POST /api/conversations/{id}/keys`
Saves encrypted conversation keys for all conversation members.
- **Body:** `{ keys: [{ user_id, encrypted_key }] }`
- **Response:** `204 No Content`

### `GET /api/conversations/{id}/key`
Returns the logged-in user's encrypted conversation key for a given conversation.
- **Response:** `{ encrypted_key }`

## WebSocket

### `GET /ws`
Upgrades the connection to a WebSocket. Requires a valid session cookie.

**Outgoing (client → server):**
- `{ type: "send_message", conversation_id, ciphertext, client_message_id? }`

**Incoming (server → client):**
- `{ type: "new_message", conversation_id, ciphertext, sender_id, display_name, profile_picture_url, message_id, expires_at? }`
- `{ type: "message_ack", conversation_id, message_id, client_message_id }`
- `{ type: "messages_updated", conversation_id }` — triggers client to reload message history
- `{ type: "profile_updated", username, display_name, profile_picture_url }` — triggers client to patch profile pictures in place
