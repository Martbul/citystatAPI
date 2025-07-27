package types

type InviteUserInfo struct {
	ID        string  `json:"id"`
	UserName  *string `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	ImageURL  *string `json:"imageUrl"`
}

type InviteInfoResponse struct {
	InvitedBy InviteUserInfo `json:"invitedBy"`
	Message   string         `json:"message"`
}

type AcceptInviteRequest struct {
	InvitedBy string `json:"invitedBy"`
}

type AcceptInviteResponse struct {
	Message string           `json:"message"`
	Friend  UserSearchResult `json:"friend"`
}

type InviteLinkResponse struct {
	InviteLink string `json:"inviteLink"`
	Message    string `json:"message"`
}