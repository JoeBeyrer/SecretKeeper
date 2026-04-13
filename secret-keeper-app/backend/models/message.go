package models

type WSMessage struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	Ciphertext     string `json:"ciphertext"`
	SenderID       string `json:"sender_id"`
	DisplayName    string `json:"display_name"`
	ProfilePictureURL string `json:"profile_picture_url"`
}
