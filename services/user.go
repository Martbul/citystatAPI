package services

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2/user"
	"citystatAPI/prisma/db"
)

type UserService struct {
	client *db.PrismaClient
}

func NewUserService(client *db.PrismaClient) *UserService {
	return &UserService{client: client}
}

// SyncUserFromClerk creates or updates user from Clerk data
func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	// Get user from Clerk
	//clerkUser, err := clerk.Users().Read(ctx, clerkUserID)
		clerkUser, err := user.Get(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}

	// Prepare user data
	var email string
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	var imageUrl *string
	if *clerkUser.ImageURL != "" {
		imageUrl = clerkUser.ImageURL
	}

	// Try to find existing user
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)

	if err != nil && err != db.ErrNotFound {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	// User exists, update it
	if existingUser != nil {
		updatedUser, err := s.client.User.FindUnique(
			db.User.ID.Equals(clerkUserID),
		).Update(
			db.User.Email.Set(email),
			db.User.FirstName.SetIfPresent(clerkUser.FirstName),
			db.User.LastName.SetIfPresent(clerkUser.LastName),
			db.User.ImageURL.SetIfPresent(imageUrl),
		).Exec(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}

		return updatedUser, nil
	}

	// User doesn't exist, create new one
	newUser, err := s.client.User.CreateOne(
		db.User.ID.Set(clerkUserID),
		db.User.Email.Set(email),
		db.User.FirstName.SetIfPresent(clerkUser.FirstName),
		db.User.LastName.SetIfPresent(clerkUser.LastName),
		db.User.ImageURL.SetIfPresent(imageUrl),
	).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// GetOrCreateUser ensures user exists in database
func (s *UserService) GetOrCreateUser(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	// Try to get user from database first
	user, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)

	if err == nil {
		return user, nil
	}

	if err == db.ErrNotFound {
		// User not in database, sync from Clerk
		return s.SyncUserFromClerk(ctx, clerkUserID)
	}

	return nil, fmt.Errorf("database error: %w", err)
}
