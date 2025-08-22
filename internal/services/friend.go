
// internal/services/friend.go
package services

import (
	"context"
	"fmt"

	"citystatAPI/internal/repository"
	"citystatAPI/types"
)

type FriendService struct {
	friendRepo repository.FriendRepository
	userRepo   repository.UserRepository
}

func NewFriendService(friendRepo repository.FriendRepository, userRepo repository.UserRepository) *FriendService {
	return &FriendService{
		friendRepo: friendRepo,
		userRepo:   userRepo,
	}
}

func (s *FriendService) AddFriend(ctx context.Context, userID, friendID string) (*types.UserSearchResult, error) {
	// Check if friend user exists
	friendUser, err := s.userRepo.GetUser(ctx, friendID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if friendship already exists
	exists, err := s.friendRepo.CheckFriendshipExists(ctx, userID, friendID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing friendship: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("already friends with this user")
	}

	// Create friendship
	_, err = s.friendRepo.CreateFriend(ctx, repository.CreateFriendParams{
		UserID:    userID,
		FriendID:  friendID,
		UserName:  stringFromPgtype(friendUser.UserName),
		FirstName: stringFromPgtype(friendUser.FirstName),
		LastName:  stringFromPgtype(friendUser.LastName),
		ImageURL:  stringFromPgtype(friendUser.ImageUrl),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create friendship: %w", err)
	}

	// Create reciprocal friendship
	currentUser, err := s.userRepo.GetUser(ctx, userID)
	if err == nil {
		_, err = s.friendRepo.CreateFriend(ctx, repository.CreateFriendParams{
			UserID:    friendID,
			FriendID:  userID,
			UserName:  stringFromPgtype(currentUser.UserName),
			FirstName: stringFromPgtype(currentUser.FirstName),
			LastName:  stringFromPgtype(currentUser.LastName),
			ImageURL:  stringFromPgtype(currentUser.ImageUrl),
		})
		// If reciprocal creation fails, log but don't fail the operation
		if err != nil {
			fmt.Printf("Warning: failed to create reciprocal friendship: %v\n", err)
		}
	}

	// Return friend info
	return &types.UserSearchResult{
		ID:        friendUser.ID,
		UserName:  stringFromPgtype(friendUser.UserName),
		FirstName: stringFromPgtype(friendUser.FirstName),
		LastName:  stringFromPgtype(friendUser.LastName),
		ImageURL:  stringFromPgtype(friendUser.ImageUrl),
		IsFriend:  true,
	}, nil
}

func (s *FriendService) GetUserFriends(ctx context.Context, userID string) ([]types.FriendResult, error) {
	friends, err := s.friendRepo.GetUserFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	results := make([]types.FriendResult, len(friends))
	for i, friend := range friends {
		results[i] = types.FriendResult{
			ID:        friend.ID,
			FriendID:  friend.FriendID,
			UserName:  friend.UserName,
			FirstName: stringFromPgtype(friend.FirstName),
			LastName:  stringFromPgtype(friend.LastName),
			ImageURL:  stringFromPgtype(friend.ImageUrl),
			CreatedAt: friend.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return results, nil
}

func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendID string) error {
	// Remove friendship
	err := s.friendRepo.DeleteFriend(ctx, userID, friendID)
	if err != nil {
		return fmt.Errorf("failed to remove friendship: %w", err)
	}

	// Remove reciprocal friendship
	err = s.friendRepo.DeleteFriend(ctx, friendID, userID)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to remove reciprocal friendship: %v\n", err)
	}

	return nil
}