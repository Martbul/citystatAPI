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

// SearchUsers searches for users by username and returns their friend status
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
		imageURL, _ := user.ImageURL()
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

// AddFriend adds a friend relationship
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

	// Create friendship record
	_, err = s.client.Friend.CreateOne(
		db.Friend.UserID.Set(userID),
		db.Friend.FriendID.Set(friendID),
		db.Friend.UserName.Set(getStringValue(friendUser.UserName)),
		db.Friend.FirstName.SetIfPresent(friendUser.FirstName),
		db.Friend.LastName.SetIfPresent(friendUser.LastName),
		db.Friend.ImageURL.SetIfPresent(friendUser.ImageURL),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create friendship: %w", err)
	}

	// Optionally create reciprocal friendship (bidirectional friendship)
	// Get current user info for reciprocal friendship
	currentUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(userID),
	).Exec(ctx)
	if err == nil {
		// Create reciprocal friendship
		_, err = s.client.Friend.CreateOne(
			db.Friend.UserID.Set(friendID),
			db.Friend.FriendID.Set(userID),
			db.Friend.UserName.Set(getStringValue(currentUser.UserName)),
			db.Friend.FirstName.SetIfPresent(currentUser.FirstName),
			db.Friend.LastName.SetIfPresent(currentUser.LastName),
			db.Friend.ImageURL.SetIfPresent(currentUser.ImageURL),
		).Exec(ctx)
		// If reciprocal creation fails, we could log it but still return success
		// since the main friendship was created
	}


	firstName, _ := friendUser.FirstName()
		lastName, _ := friendUser.LastName()
		userName, _ := friendUser.UserName()
		imageURL, _ := friendUser.ImageURL()
	// Return friend info
	result := &types.UserSearchResult{
		ID:        friendUser.ID,
		UserName:  &userName,
		FirstName: &firstName,
		LastName:  &lastName,
		ImageURL:  &imageURL,
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
		results[i] = types.FriendResult{
			ID:        friend.ID,
			FriendID:  friend.FriendID,
			UserName:  friend.UserName,
			FirstName: friend.FirstName,
			LastName:  friend.LastName,
			ImageURL:  friend.ImageURL,
			CreatedAt: friend.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return results, nil
}

// RemoveFriend removes a friendship
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	// Remove the friendship
	_, err := s.client.Friend.FindFirst(
		db.Friend.And(
			db.Friend.UserID.Equals(userID),
			db.Friend.FriendID.Equals(friendID),
		),
	).Delete().Exec(ctx)
	if err != nil {
		if err == db.ErrNotFound {
			return fmt.Errorf("friendship not found")
		}
		return fmt.Errorf("failed to remove friendship: %w", err)
	}

	// Remove reciprocal friendship if it exists
	_, err = s.client.Friend.FindFirst(
		db.Friend.And(
			db.Friend.UserID.Equals(friendID),
			db.Friend.FriendID.Equals(userID),
		),
	).Delete().Exec(ctx)
	// Don't return error if reciprocal doesn't exist, just log it
	if err != nil && err != db.ErrNotFound {
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
