package types

// CityData represents the selected city information from the frontend
type CityData struct {
    Name        string  `json:"name"`
    Country     string  `json:"country"`
    State       string  `json:"state"`
    Lat         float64 `json:"lat"`
    Lng         float64 `json:"lng"`
    DisplayName string  `json:"display_name"`
}

// UserUpdateRequest represents the request body for updating user details
// type UserUpdateRequest struct {
// 	FirstName                 *string `json:"firstName,omitempty"`
// 	LastName                  *string `json:"lastName,omitempty"`
// 	UserName                  *string `json:"userName,omitempty"`
// 	ImageURL                  *string `json:"imageUrl,omitempty"`
// 	CompletedTutorial         *bool   `json:"completedTutorial,omitempty"`
// 	IsLocationTrackingEnabled bool    `json:"isLocationTrackingEnabled,omitempty"`

// 	// City selection data
// 	SelectedCity *CityData `json:"selectedCity,omitempty"`

// 	// Alternative: if you prefer individual fields
// 	CityName    *string `json:"cityName,omitempty"`
// 	CityCountry *string `json:"cityCountry,omitempty"`
// 	CityState   *string `json:"cityState,omitempty"`
// 	CityCoords  *struct {
// 		Lat float64 `json:"lat"`
// 		Lng float64 `json:"lng"`
// 	} `json:"cityCoords,omitempty"` 
    // }

type UserUpdateRequest struct {
    FirstName                 string    `json:"firstName,omitempty"`
    LastName                  string    `json:"lastName,omitempty"`
    UserName                  string    `json:"userName,omitempty"`
    ImageURL                  string    `json:"imageUrl,omitempty"`
    CompletedTutorial         bool      `json:"completedTutorial,omitempty"`
    IsLocationTrackingEnabled bool      `json:"isLocationTrackingEnabled,omitempty"`
    SelectedCity              *CityData `json:"selectedCity,omitempty"`
}
//TODO: Finish the profile req type and other logi for it
type UserEditProfileRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	AboutMe   *string `json:"aboutMe,omitempty"`
	ImageURL  *string `json:"imageUrl,omitempty"`
}

type SearchUsersResponse struct {
	Users []UserSearchResult `json:"users"`
}
