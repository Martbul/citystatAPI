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
	"citystatAPI/telemetry"
	"citystatAPI/types"
	"citystatAPI/utils"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)


var userTracer = otel.Tracer("citystat-api/handlers/user")

// var (
//     userTracer = otel.Tracer("citystat-api/handlers/user")
//     userMeter  = otel.Meter("citystat-api/handlers/user")
    
//     // User-specific metrics
//     userOperationsTotal metric.Int64Counter
//     userSearchDuration  metric.Float64Histogram
//     cityDataProcessingDuration metric.Float64Histogram
// )

// func init() {
//     var err error
    
//     userOperationsTotal, err = userMeter.Int64Counter(
//         "user_operations_total",
//         metric.WithDescription("Total number of user operations"),
//     )
//     if err != nil {
//         panic(err)
//     }
    
//     userSearchDuration, err = userMeter.Float64Histogram(
//         "user_search_duration_seconds",
//         metric.WithDescription("Duration of user search operations"),
//         metric.WithUnit("s"),
//     )
//     if err != nil {
//         panic(err)
//     }
    
//     cityDataProcessingDuration, err = userMeter.Float64Histogram(
//         "city_data_processing_duration_seconds",
//         metric.WithDescription("Duration of city data processing"),
//         metric.WithUnit("s"),
//     )
//     if err != nil {
//         panic(err)
//     }
// }


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
    // Use the context from the request (already has telemetry from middleware)
    ctx := r.Context()
    
    // Create a span for this specific operation
    ctx, span := userTracer.Start(ctx, "GetProfile")
    defer span.End()
    
    // Record user operation using the telemetry package
    telemetry.RecordUserOperation(ctx, "get_profile")
    
    userID, ok := middleware.GetUserID(r)
    if !ok {
        span.SetAttributes(
            attribute.Bool("error", true),
            attribute.String("error.type", "no_user_id"),
        )
        // Record error using telemetry package
        telemetry.RecordError(ctx, "no_user_id", "get_profile")
        middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
        return
    }

    span.SetAttributes(attribute.String("user.id", userID))

    user, err := h.userService.GetOrCreateUser(ctx, userID)
    if err != nil {
        span.SetAttributes(
            attribute.Bool("error", true),
            attribute.String("error.message", err.Error()),
        )
        // Use the proper error recording function
        telemetry.RecordErrorWithTrace(ctx, err, "get_profile")
        middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
        return
    }

    span.SetAttributes(attribute.Bool("success", true))
    middleware.JSONResponse(w, user, http.StatusOK)
}

// func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := middleware.GetUserID(r)
// 	if !ok {
// 		middleware.ErrorResponse(w, "User ID not found in context", http.StatusUnauthorized)
// 		return
// 	}

// 	user, err := h.userService.GetOrCreateUser(r.Context(), userID)
// 	if err != nil {
// 		middleware.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	middleware.JSONResponse(w, user, http.StatusOK)
// }



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

	

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	
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
