

// internal/services/user.go
package services

import (
	"context"
	"fmt"
	"strings"

	"citystatAPI/internal/db"
	"citystatAPI/internal/repository"
	"citystatAPI/types"

	"github.com/clerk/clerk-sdk-go/v2/user"
)

type UserService struct {
	userRepo     repository.UserRepository
	settingsRepo repository.SettingsRepository
}

func NewUserService(userRepo repository.UserRepository, settingsRepo repository.SettingsRepository) *UserService {
	return &UserService{
		userRepo:     userRepo,
		settingsRepo: settingsRepo,
	}
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*db.User, error) {
	return s.userRepo.GetUser(ctx, userID)
}

func (s *UserService) GetOrCreateUser(ctx context.Context, clerkUserID string) (*db.User, error) {
	// Try to get user from database first
	user, err := s.userRepo.GetUser(ctx, clerkUserID)
	if err == nil {
		return user, nil
	}

	if strings.Contains(err.Error(), "not found") {
		// User not in database, sync from Clerk
		return s.SyncUserFromClerk(ctx, clerkUserID)
	}

	return nil, fmt.Errorf("database error: %w", err)
}

func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.User, error) {
	clerkUser, err := user.Get(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}

	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	// Check if user exists
	existingUser, err := s.userRepo.GetUser(ctx, clerkUserID)
	if err == nil {
		fmt.Println("user exist", existingUser)
		// Update existing user
		return s.userRepo.UpdateUser(ctx, repository.UpdateUserParams{
			ID:        clerkUserID,
			FirstName: clerkUser.FirstName,
			LastName:  clerkUser.LastName,
			UserName:  clerkUser.Username,
			ImageURL:  clerkUser.ImageURL,
		})
	}

	// Create new user
	newUser, err := s.userRepo.CreateUser(ctx, repository.CreateUserParams{
		ID:        clerkUserID,
		Email:     email,
		FirstName: clerkUser.FirstName,
		LastName:  clerkUser.LastName,
		UserName:  clerkUser.Username,
		ImageURL:  clerkUser.ImageURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default settings for new user
	_, err = s.settingsRepo.CreateUserSettings(ctx, repository.CreateSettingsParams{
		UserID: clerkUserID,
	})
	if err != nil {
		// Log error but don't fail user creation
		fmt.Printf("Warning: failed to create default settings for user %s: %v\n", clerkUserID, err)
	}

	return newUser, nil
}

func (s *UserService) UpdateUserDetails(ctx context.Context, clerkUserID string, updates types.UserUpdateRequest) (*db.User, error) {
	// Ensure user exists
	_, err := s.GetOrCreateUser(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	return s.userRepo.UpdateUser(ctx, repository.UpdateUserParams{
		ID:                clerkUserID,
		FirstName:         updates.FirstName,
		LastName:          updates.LastName,
		UserName:          updates.UserName,
		ImageURL:          updates.ImageURL,
		CompletedTutorial: updates.CompletedTutorial,
	})
}

func (s *UserService) UpdateNote(ctx context.Context, clerkUserID string, note string) (*db.User, error) {
	return s.userRepo.UpdateUser(ctx, repository.UpdateUserParams{
		ID:   clerkUserID,
		Note: &note,
	})
}

func (s *UserService) SearchUsers(ctx context.Context, currentUserID, username string) ([]types.UserSearchResult, error) {
	users, err := s.userRepo.SearchUsers(ctx, currentUserID, username)
	if err != nil {
		return nil, err
	}

	// TODO: Add friend status checking logic here
	results := make([]types.UserSearchResult, len(users))
	for i, user := range users {
		results[i] = types.UserSearchResult{
			ID:        user.ID,
			UserName:  stringFromPgtype(user.UserName),
			FirstName: stringFromPgtype(user.FirstName),
			LastName:  stringFromPgtype(user.LastName),
			ImageURL:  stringFromPgtype(user.ImageUrl),
			IsFriend:  false, // TODO: Implement friend checking
		}
	}

	return results, nil
}

// Helper function to convert possible *string or string to *string
func stringFromPgtype(pgText interface{}) *string {
	switch v := pgText.(type) {
	case *string:
		return v
	case string:
		return &v
	default:
		return nil
	}
}

// Returns string value, or empty string if nil or not a string
func stringValueFromPgtype(pgText interface{}) string {
	switch v := pgText.(type) {
	case *string:
		if v != nil {
			return *v
		}
		return ""
	case string:
		return v
	default:
		return ""
	}
}
