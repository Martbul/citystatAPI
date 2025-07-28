package types


type UserUpdateRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	UserName *string `json:"userName,omitempty"`
	ImageURL  *string `json:"imageUrl,omitempty"`
	CompletedTutorial *bool   `json:"completedTutorial,omitempty"`
}


//TODO: Finish the profile req type and other logi for it
type UserEditProfileRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	AboutMe *string `json:"aboutMe,omitempty"`
	ImageURL  *string `json:"imageUrl,omitempty"`
}




type SearchUsersResponse struct {
	Users []UserSearchResult `json:"users"`
}