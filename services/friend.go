package services

import (
	"context"
	"fmt"

	"citystatAPI/prisma/db"
	"citystatAPI/types"
)

type FriendService struct {
	client *db.PrismaClient
}

func NewFriendService(client *db.PrismaClient) *FriendService {
	return &FriendService{client: client}
}
func (s *FriendService) SearchUsers(ctx context.Context, currentUserID, username string) ([]types.UserSearchResult, error) {
	// Get current user's friends to check friend status
	currentUserFriends, err := s.client.Friend.FindMany(
		db.Friend.UserID.Equals(currentUserID),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user friends: %w", err)
	}

	// Create a map for quick friend lookup
	friendMap := make(map[string]bool)
	for _, friend := range currentUserFriends {
		friendMap[friend.FriendID] = true
	}

	// Search for users by username (case-insensitive partial matching)
	users, err := s.client.User.FindMany(
		db.User.And(
			db.User.UserName.Contains(username),
			db.User.ID.Not(currentUserID), // Exclude current user
		),
	).Take(10).Exec(ctx) // Limit to 10 results

	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Convert to response format with friend status
	results := make([]types.UserSearchResult, len(users))
	for i, user := range users {
		firstName, _ := user.FirstName()
		lastName, _ := user.LastName()
		userName, _ := user.UserName()
		imageURL:= user.ImageURL
		results[i] = types.UserSearchResult{
			ID:        user.ID,
			UserName:  &userName,
			FirstName: &firstName,
			LastName:  &lastName,
			ImageURL:  &imageURL,
			IsFriend:  friendMap[user.ID],
		}
	}

	return results, nil
}

func (s *FriendService) AddFriend(ctx context.Context, userID, friendID string) (*types.UserSearchResult, error) {
	// Check if friend user exists
	friendUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(friendID),
	).Exec(ctx)
	if err != nil {
		if err == db.ErrNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to find friend user: %w", err)
	}

	// Check if friendship already exists
	existingFriendship, err := s.client.Friend.FindFirst(
		db.Friend.And(
			db.Friend.UserID.Equals(userID),
			db.Friend.FriendID.Equals(friendID),
		),
	).Exec(ctx)
	if err != nil && err != db.ErrNotFound {
		return nil, fmt.Errorf("failed to check existing friendship: %w", err)
	}

	if existingFriendship != nil {
		return nil, fmt.Errorf("already friends with this user")
	}

	// Get friend user details
	friendUserName, _ := friendUser.UserName()
	friendFirstName, _ := friendUser.FirstName()
	friendLastName, _ := friendUser.LastName()
	friendImageURL := friendUser.ImageURL
	// Create friendship record with required parameters first, then optional ones
	var optionalParams []db.FriendSetParam
	
	if friendFirstName != "" {
		optionalParams = append(optionalParams, db.Friend.FirstName.Set(friendFirstName))
	}
	if friendLastName != "" {
		optionalParams = append(optionalParams, db.Friend.LastName.Set(friendLastName))
	}
	if friendImageURL != "" {
		optionalParams = append(optionalParams, db.Friend.ImageURL.Set(friendImageURL))
	}
	
	_, err = s.client.Friend.CreateOne(
		db.Friend.UserName.Set(friendUserName),           // Required: userName
		db.Friend.User.Link(db.User.ID.Equals(userID)),   // Required: user relation
		db.Friend.Friend.Link(db.User.ID.Equals(friendID)), // Required: friend relation
		optionalParams...,                                // Optional parameters
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create friendship: %w", err)
	}

	// Optionally create reciprocal friendship (bidirectional friendship)
	currentUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(userID),
	).Exec(ctx)
	if err == nil {
		// Get current user details
		currentUserName, _ := currentUser.UserName()
		currentFirstName, _ := currentUser.FirstName()
		currentLastName, _ := currentUser.LastName()
		currentImageURL := currentUser.ImageURL

		// Create reciprocal friendship
		var reciprocalOptionalParams []db.FriendSetParam
		
		if currentFirstName != "" {
			reciprocalOptionalParams = append(reciprocalOptionalParams, db.Friend.FirstName.Set(currentFirstName))
		}
		if currentLastName != "" {
			reciprocalOptionalParams = append(reciprocalOptionalParams, db.Friend.LastName.Set(currentLastName))
		}
		if currentImageURL != "" {
			reciprocalOptionalParams = append(reciprocalOptionalParams, db.Friend.ImageURL.Set(currentImageURL))
		}
		
		_, err = s.client.Friend.CreateOne(
			db.Friend.UserName.Set(currentUserName),           // Required: userName
			db.Friend.User.Link(db.User.ID.Equals(friendID)),   // Required: user relation
			db.Friend.Friend.Link(db.User.ID.Equals(userID)),   // Required: friend relation
			reciprocalOptionalParams...,                        // Optional parameters
		).Exec(ctx)
		// If reciprocal creation fails, we could log it but still return success
		// since the main friendship was created
	}

	// Return friend info
	result := &types.UserSearchResult{
		ID:        friendUser.ID,
		UserName:  &friendUserName,
		FirstName: &friendFirstName,
		LastName:  &friendLastName,
		ImageURL:  &friendImageURL,
		IsFriend:  true,
	}

	return result, nil
}

// GetUserFriends returns all friends for a user
func (s *FriendService) GetUserFriends(ctx context.Context, userID string) ([]types.FriendResult, error) {
	friends, err := s.client.Friend.FindMany(
		db.Friend.UserID.Equals(userID),
	).OrderBy(
		db.Friend.CreatedAt.Order(db.DESC),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user friends: %w", err)
	}

	results := make([]types.FriendResult, len(friends))
	for i, friend := range friends {
		fn, _ := friend.FirstName()
		ln, _ := friend.LastName()
		imageURL, _ := friend.ImageURL()
		results[i] = types.FriendResult{
			ID:        friend.ID,
			FriendID:  friend.FriendID,
			UserName:  friend.UserName,
			FirstName: &fn,
			LastName:  &ln,
			ImageURL:  &imageURL,
			CreatedAt: friend.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return results, nil
}
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	// Remove the friendship using DeleteMany
	result, err := s.client.Friend.FindMany(
		db.Friend.And(
			db.Friend.UserID.Equals(userID),
			db.Friend.FriendID.Equals(friendID),
		),
	).Delete().Exec(ctx)
	
	if err != nil {
		return fmt.Errorf("failed to remove friendship: %w", err)
	}

	// Check if any friendship was actually deleted
	if result.Count == 0 {
		return fmt.Errorf("friendship not found")
	}

	// Remove reciprocal friendship if it exists
	_, err = s.client.Friend.FindMany(
		db.Friend.And(
			db.Friend.UserID.Equals(friendID),
			db.Friend.FriendID.Equals(userID),
		),
	).Delete().Exec(ctx)
	
	// Don't return error if reciprocal doesn't exist, just log it
	if err != nil {
		// Log the error but don't fail the operation
		fmt.Printf("Warning: failed to remove reciprocal friendship: %v\n", err)
	}

	return nil
}

// Helper function to safely get string value from pointer
func getStringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}