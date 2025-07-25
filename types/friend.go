package types


type UserSearchResult struct {
	ID        string  `json:"id"`
	UserName  *string `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	ImageURL  *string `json:"imageUrl"`
	IsFriend  bool    `json:"isFriend"`
}

type AddFriendRequest struct {
	FriendID string `json:"friendId"`
}

type AddFriendResponse struct {
	Message string             `json:"message"`
	Friend  UserSearchResult   `json:"friend"`
}


type FriendResult struct {
	ID        string  `json:"id"`
	FriendID  string  `json:"friendId"`
	UserName  string  `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	ImageURL  *string `json:"imageUrl"`
	CreatedAt string  `json:"createdAt"`
}

type FriendsListResponse struct {
	Friends []FriendResult `json:"friends"`
}
