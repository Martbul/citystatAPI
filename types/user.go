package types


type UserUpdateRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	UserName *string `json:"userName,omitempty"`
	ImageURL  *string `json:"imageUrl,omitempty"`
	CompletedTutorial *bool   `json:"completedTutorial,omitempty"`
}


type SearchUsersResponse struct {
	Users []UserSearchResult `json:"users"`
}