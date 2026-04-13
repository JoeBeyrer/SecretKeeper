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

// UserSearchResult is returned by the user-search endpoint.
// Status is one of: "none", "friend", "pending_outgoing", "pending_incoming".
type UserSearchResult struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}
 