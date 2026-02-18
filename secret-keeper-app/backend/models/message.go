package models

type WSMessage struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	Ciphertext     string `json:"ciphertext"`
}