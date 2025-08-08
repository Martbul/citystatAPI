package types

type SaveLocationPermitionRequest struct {
	HasLocationPermission bool `json:"hasLocationPermission"`
}

type SaveLocationPermitionResponse struct {
	Success bool `json:"success"`
}


type AddVisitedStreetsRequest struct {
	FriendID string `json:"friendId"`
}

type AddVisitedStreetsResponse struct {
	Message string             `json:"message"`
	Friend  UserSearchResult   `json:"friend"`
}