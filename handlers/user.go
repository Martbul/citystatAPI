package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"citystatAPI/middleware"
	"citystatAPI/services"
	"citystatAPI/types"
	"citystatAPI/utils"
)

type UserHandler struct {
	userService *services.UserService
}


func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}



// GetProfile godoc
// @Summary      Get user profile
// @Description  Returns the current user's profile
// @Tags         user
// @Produce      json
// @Success      200  {object}  db.UserModel
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /user [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	user, err := h.userService.GetOrCreateUser(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}



//! after user selects its city calculate its street count, its street area, 
// UpdateUserDetails godoc
// @Summary      Update user details
// @Description  Updates profile fields like name, username, and selected city
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        request  body      types.UserUpdateRequest  true  "User update payload"
// @Success      200      {object}  db.UserModel
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /user/details [put]

func (h *UserHandler) UpdateUserDetails(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// log.Printf("🔍 Current struct definition hash: %x", 
   //  fmt.Sprintf("%+v", reflect.TypeOf(types.UserUpdateRequest{})))

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	log.Printf("Content-Type: %s", contentType)

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Raw request body: %s", string(body))
	log.Printf("Body length: %d", len(body))
	
	// Validate that we have a non-empty body
	if len(body) == 0 {
		log.Printf("❌ Empty request body")
		http.Error(w, `{"error":"Empty request body"}`, http.StatusBadRequest)
		return
	}

	// Test if it's valid JSON first
	var testJSON interface{}
	if err := json.Unmarshal(body, &testJSON); err != nil {
		log.Printf("❌ Invalid JSON: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	log.Printf("✅ Valid JSON structure: %+v", testJSON)

	// Parse into our struct
	var req services.UserUpdateRequest2
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("❌ Failed to unmarshal into UserUpdateRequest: %v", err)
		log.Printf("❌ Error type: %T", err)
		
		// Let's see what specific field is causing issues
		if typeErr, ok := err.(*json.UnmarshalTypeError); ok {
			log.Printf("❌ Type error details - Field: %s, Expected: %s, Got: %s, Offset: %d", 
				typeErr.Field, typeErr.Type, typeErr.Value, typeErr.Offset)
		}
		
		http.Error(w, fmt.Sprintf(`{"error":"Failed to parse request: %v"}`, err), http.StatusBadRequest)
		return
	}

	log.Printf("✅ Successfully decoded request: %+v", req)
	log.Printf("FirstName: %s", req.FirstName)
	log.Printf("LastName: %s", req.LastName)
	log.Printf("UserName: %s", req.UserName)
	log.Printf("ImageURL: %s", req.ImageURL)
	log.Printf("CompletedTutorial: %v", req.CompletedTutorial)
	log.Printf("IsLocationTrackingEnabled: %v", req.IsLocationTrackingEnabled)

	if req.SelectedCity != nil {
		log.Printf("SelectedCity: %+v", *req.SelectedCity)
		log.Printf("  Name: %s", req.SelectedCity.Name)
		log.Printf("  Country: %s", req.SelectedCity.Country)
		log.Printf("  State: %s", req.SelectedCity.State)
		log.Printf("  Lat: %f", req.SelectedCity.Lat)
		log.Printf("  Lng: %f", req.SelectedCity.Lng)
		log.Printf("  DisplayName: %s", req.SelectedCity.DisplayName)
	} else {
		log.Printf("SelectedCity: nil")
	}

	// Update user with the provided data
	user, err := h.userService.UpdateUserDetails(r.Context(), userID, req)
	if err != nil {
		log.Printf("❌ Service error: %v", err)
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ User updated successfully")
	middleware.JSONResponse(w, user, http.StatusOK)
}

// func (h *UserHandler) UpdateUserDetails(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := middleware.GetUserID(r)
// 	if !ok {
// 		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
// 		return
// 	}
// 	    body, err := io.ReadAll(r.Body)
// 		  if err != nil {
//         log.Printf("Failed to read request body: %v", err)
//         http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
//         return
//     }
    
// 	    log.Printf("Raw request body: %s", string(body))
// 	fmt.Println("+++++++++")

// 	    r.Body = io.NopCloser(strings.NewReader(string(body)))

// 	// Parse request body
//     var req types.UserUpdateRequest

// 	 decoder := json.NewDecoder(r.Body)
//     decoder.DisallowUnknownFields() // This might be causing the issue!

// 	 if err := decoder.Decode(&req); err != nil {
//         log.Printf("❌ JSON decode error: %v", err)
//         log.Printf("❌ Error type: %T", err)
        
//         // Try decoding without DisallowUnknownFields
//         r.Body = io.NopCloser(strings.NewReader(string(body)))
//         decoder2 := json.NewDecoder(r.Body)
//         // Don't call DisallowUnknownFields() this time
        
//         if err2 := decoder2.Decode(&req); err2 != nil {
//             log.Printf("❌ JSON decode error even without DisallowUnknownFields: %v", err2)
//             http.Error(w, fmt.Sprintf(`{"error":"Invalid request body - JSON decode failed: %v"}`, err2), http.StatusBadRequest)
//             return
//         } else {
//             log.Printf("⚠️ JSON decoded successfully WITHOUT DisallowUnknownFields - there might be extra fields")
//         }
//     }
    
//     log.Printf("✅ Successfully decoded request: %+v", req)
//   log.Printf("FirstName: %s", req.FirstName)
//     log.Printf("LastName: %s", req.LastName)
//     log.Printf("UserName: %s", req.UserName)
//     log.Printf("ImageURL: %s", req.ImageURL)
//     log.Printf("CompletedTutorial: %v", req.CompletedTutorial)
//     log.Printf("IsLocationTrackingEnabled: %v", req.IsLocationTrackingEnabled)
    
//     if req.SelectedCity != nil {
//         log.Printf("SelectedCity: %+v", *req.SelectedCity)
//         log.Printf("  Name: %s", req.SelectedCity.Name)
//         log.Printf("  Country: %s", req.SelectedCity.Country)
//         log.Printf("  State: %s", req.SelectedCity.State)
//         log.Printf("  Lat: %f", req.SelectedCity.Lat)
//         log.Printf("  Lng: %f", req.SelectedCity.Lng)
//         log.Printf("  DisplayName: %s", req.SelectedCity.DisplayName)
//     } else {
//         log.Printf("SelectedCity: nil")
//     }

	
// 	fmt.Println("------------updating user req body parsed")

// 	// Update user with the provided data
// 	user, err := h.userService.UpdateUserDetails(r.Context(), userID, req)
// 	if err != nil {
// 		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	fmt.Println("user updated successfuly")

// 	middleware.JSONResponse(w, user, http.StatusOK)
// }


func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// userID, ok := middleware.GetUserID(r)
	// if !ok {
	// 	middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
	// 	return
	// }

	// var updateReq types.UserEditProfileRequest
	// if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
	// 	middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
	// 	return
	// }
	// fmt.Println("req body parsed")

	// user, err := h.userService.UpdateUser(r.Context(), userID, updateReq)
	// if err != nil {
	// 	middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// fmt.Println("user updated successfuly")

	// middleware.JSONResponse(w, user, http.StatusOK)
}

// func (h *UserHandler) EditProfile(w http.ResponseWriter, r *http.Request) {
//     // Parse into generic map
//     var updateReq map[string]interface{}
//     if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
//         middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
//         return
//     }
    
//     // Pass directly to service layer
//     user, err := h.userService.UpdateUser(r.Context(), userID, updateReq)
//     if err != nil {
//         middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
//         return
//     }
    
//     middleware.JSONResponse(w, user, http.StatusOK)
// }


func (h *UserHandler) SyncProfileFromClerk(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Ensure user exists
	_, err := h.userService.GetOrCreateUser(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sync latest data from Clerk
	user, err := h.userService.SyncUserFromClerk(r.Context(), userID)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	middleware.JSONResponse(w, user, http.StatusOK)
}





func (h *UserHandler) UpdateNote(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r)
    if !ok {
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    // Use ONLY your ParseJSON helper - remove the json.NewDecoder line
    updateReq, err := utils.ParseJSON[map[string]interface{}](r)
    if err != nil {
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    fmt.Println("req body parsed")
    
    user, err := h.userService.UpdateNote(r.Context(), userID, updateReq)
    if err != nil {
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Println("user updated successfully")
    middleware.JSONResponse(w, user, http.StatusOK)
}




func (h *UserHandler) UpdateActiveHours(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r)
    if !ok {
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    fmt.Printf("UpdateActiveHours called for user: %s\n", userID)

    updateReq, err := utils.ParseJSON[map[string]interface{}](r)
    if err != nil {
        fmt.Printf("Error parsing JSON: %v\n", err)
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    fmt.Printf("Request body parsed: %+v\n", updateReq)
    
    // Validate activeHours exists and is the right type
    activeHours, exists := updateReq["activeHours"]
    if !exists {
        fmt.Println("activeHours field missing from request")
        middleware.ErrorResponse(w, "activeHours field is required", http.StatusBadRequest)
        return
    }
    
    fmt.Printf("activeHours value: %v (type: %T)\n", activeHours, activeHours)
    
    user, err := h.userService.UpdateActiveHours(r.Context(), userID, updateReq)
    if err != nil {
        fmt.Printf("Service error: %v\n", err)
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Printf("User updated successfully: %+v\n", user)
    middleware.JSONResponse(w, user, http.StatusOK)
}


func (h *UserHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
    userID, ok := middleware.GetUserID(r)
    if !ok {
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    var settingsReq map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&settingsReq); err != nil {
        middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    fmt.Println("Settings update request:", settingsReq)

    user, err := h.userService.UpdateUserSettings(r.Context(), userID, settingsReq)
    if err != nil {
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Println("Settings updated successfully")
    middleware.JSONResponse(w, user, http.StatusOK)
}

// func (h *UserHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
//     userID, ok := middleware.GetUserID(r)
//     if !ok {
//         middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
//         return
//     }

//     var updateReq map[string]interface{}
//     if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
//         middleware.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
//         return
//     }
    
//     fmt.Println("Profile update request:", updateReq)
    
//     user, err := h.userService.UpdateUserProfile(r.Context(), userID, updateReq)
//     if err != nil {
//         middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
//         return
//     }
    
//     fmt.Println("Profile updated successfully")
//     middleware.JSONResponse(w, user, http.StatusOK)
// }


func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	fmt.Println("Searching for users with username:", username)
	if username == "" {
		middleware.ErrorResponse(w, "Username parameter is required", http.StatusBadRequest)
		return
	}

	username = strings.TrimSpace(username)
	if len(username) < 1 {
		middleware.ErrorResponse(w, "Username must be at least 1 character", http.StatusBadRequest)
		return
	}

	users, err := h.userService.SearchUsers(r.Context(), userID, username)
	if err != nil {
		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.SearchUsersResponse{Users: users}
	middleware.JSONResponse(w, response, http.StatusOK)
}


func (h *UserHandler) GetUsersSameCity(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		fmt.Printf("Error: User ID not found in context\n")
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	fmt.Printf("Fetching users in same city for user: %s\n", userID)

	users, err := h.userService.GetUsersSameCity(r.Context(), userID)
	if err != nil {
		fmt.Printf("Error in GetUsersSameCity: %v\n", err)
		// Don't return 500 for business logic errors - return appropriate status codes
		if err.Error() == "current user not found" {
			middleware.ErrorResponse(w, "User not found", http.StatusNotFound)
			return
		}
		if err.Error() == "current user has no city set" {
			middleware.ErrorResponse(w, "User location not set", http.StatusBadRequest)
			return
		}
		middleware.ErrorResponse(w, "Failed to fetch suggested users", http.StatusInternalServerError)
		return
	}


	fmt.Println("USERS IN SAME CITY")
	fmt.Println(users)

	response := types.SearchUsersResponse{Users: users}
	middleware.JSONResponse(w, response, http.StatusOK)
}


// Additional debugging middleware you can add
func (h *UserHandler) DebugGetUsersSameCity(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("=== Debug GetUsersSameCity Request ===\n")
	fmt.Printf("Method: %s\n", r.Method)
	fmt.Printf("URL: %s\n", r.URL.String())
	fmt.Printf("Headers: %v\n", r.Header)
	
	userID, ok := middleware.GetUserID(r)
	if !ok {
		fmt.Printf("ERROR: No user ID in context\n")
		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	
	fmt.Printf("User ID from context: %s\n", userID)
	
	// Check database connection
	if err := h.userService.HealthCheck(r.Context()); err != nil {
		fmt.Printf("ERROR: Database health check failed: %v\n", err)
		middleware.ErrorResponse(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}
	
	// Proceed with normal flow
	h.GetUsersSameCity(w, r)
}
