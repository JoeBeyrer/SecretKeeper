package models
 
type Friendship struct {
	ID string `json:"id"`
	RequesterID string `json:"requester_id"`
	AddresseeID string `json:"addressee_id"`
	Accepted bool `json:"accepted"` 
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
 

type FriendEntry struct {
	UserID string `json:"user_id"`
	Username string `json:"username"`
	DisplayName string `json:"display_name"`
	Accepted bool `json:"accepted"` 
	Direction string `json:"direction"`
}
 