# Sprint 1 Report

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
### Password Reset
- Move SMTP credentials out of mailer.go and into environment variables to prevent secret exposure in version control
- Add rate limiting to prevent repeated submissions of the same email address from flooding a user's inbox with reset emails
- Add email verification at signup so that the email stored in the database is confirmed to belong to the user before a reset can be triggered
- Perhaps move used/expired reset tokens out of the password_resets table into a separate audit log table to keep the active table small and queryable

## Successfully Completed
- Account creation — users can register with a username, email, and password; passwords are hashed with bcrypt before storage
- Login — users can authenticate with their username and password; a secure session cookie is issued on success with a 24-hour TTL
- Password reset — users can request a reset link via email, receive a secure one-time token valid for 1 hour, and set a new password through a dedicated page; all active sessions are invalidated on reset
- Secure credential storage — passwords are hashed using bcrypt, never stored in plaintext; session tokens are stored as UUIDs with expiration enforcement
- Web access — the app runs in the browser via Angular at localhost:4200, communicating with the Go backend at localhost:8080
- Backend session management — sessions are created, validated, and deleted from the database; expired sessions are rejected automatically
- Database schema — SQLite database initialized with tables for users, sessions, password resets, conversations, conversation members, and messages with foreign key enforcement
- Messaging infrastructure — WebSocket handler implemented with a hub that manages connected clients, routes messages to conversation members, and saves ciphertext to the database
- Conversation creation — backend endpoint to create conversations and add members, with deduplication logic
- Basic messaging UI — frontend messaging page renders messages with username, timestamp, and content; supports sending new messages with Enter key or button click
- CORS configuration — backend configured to accept requests from the Angular frontend origin with credentials

## Incomplete / Carried Over
- Messaging UI is not yet connected to the backend WebSocket — messages are currently loaded from a static messaging.txt file
- Signup form validation logic needs to be cleaned up
- No logout endpoint implemented
- User profiles table exists in the database but has no corresponding handlers or frontend: display name, bio, and profile picture are not usable yet
- Password reset SMTP credentials are hardcoded in source code — will expose secrets if not revoked from public repository and hidden
- No end-to-end encryption implemented yet - messages are stored as ciphertext but no key exchange or encryption logic exists on the frontend
- No user search functionality
- No friends/contacts system 

## Demo Video
