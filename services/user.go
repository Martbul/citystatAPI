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

// UserUpdateRequest represents the structure for user update requests
type UserUpdateRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
	ImageURL  *string `json:"imageUrl,omitempty"`
}

func NewUserService(client *db.PrismaClient) *UserService {
	return &UserService{client: client}
}

// UpdateUser updates user data in the database
func (s *UserService) UpdateUser(ctx context.Context, clerkUserID string, updates UserUpdateRequest) (*db.UserModel, error) {
	// Ensure user exists first
	existingUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Exec(ctx)

	if err != nil {
		if err == db.ErrNotFound {
			// User doesn't exist, sync from Clerk first
			user, syncErr := s.SyncUserFromClerk(ctx, clerkUserID)
			if syncErr != nil {
				return nil, fmt.Errorf("failed to sync user from Clerk: %w", syncErr)
			}
			existingUser = user
		} else {
			return nil, fmt.Errorf("error checking existing user: %w", err)
		}
	}

	// Build update operations based on provided fields
	updateOps := []db.UserSetParam{}

	if updates.Email != nil {
		updateOps = append(updateOps, db.User.Email.Set(*updates.Email))
	}
	if updates.FirstName != nil {
		updateOps = append(updateOps, db.User.FirstName.Set(*updates.FirstName))
	}
	if updates.LastName != nil {
		updateOps = append(updateOps, db.User.LastName.Set(*updates.LastName))
	}
	if updates.ImageURL != nil {
		updateOps = append(updateOps, db.User.ImageURL.Set(*updates.ImageURL))
	}

	// If no updates provided, return existing user
	if len(updateOps) == 0 {
		return existingUser, nil
	}

	// Perform the update
	updatedUser, err := s.client.User.FindUnique(
		db.User.ID.Equals(clerkUserID),
	).Update(updateOps...).Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return updatedUser, nil
}

// SyncUserFromClerk creates or updates user from Clerk data
func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
	// Get user from Clerk
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
	if clerkUser.ImageURL != nil && *clerkUser.ImageURL != "" {
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
// package services

// import (
// 	"context"
// 	"fmt"

// 	"github.com/clerk/clerk-sdk-go/v2/user"
// 	"citystatAPI/prisma/db"
// )

// type UserService struct {
// 	client *db.PrismaClient
// }

// func NewUserService(client *db.PrismaClient) *UserService {
// 	return &UserService{client: client}
// }

// // SyncUserFromClerk creates or updates user from Clerk data
// func (s *UserService) SyncUserFromClerk(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
// 	// Get user from Clerk
// 	//clerkUser, err := clerk.Users().Read(ctx, clerkUserID)
// 		clerkUser, err := user.Get(ctx, clerkUserID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
// 	}

// 	// Prepare user data
// 	var email string
// 	if len(clerkUser.EmailAddresses) > 0 {
// 		email = clerkUser.EmailAddresses[0].EmailAddress
// 	}

// 	var imageUrl *string
// 	if *clerkUser.ImageURL != "" {
// 		imageUrl = clerkUser.ImageURL
// 	}

// 	// Try to find existing user
// 	existingUser, err := s.client.User.FindUnique(
// 		db.User.ID.Equals(clerkUserID),
// 	).Exec(ctx)

// 	if err != nil && err != db.ErrNotFound {
// 		return nil, fmt.Errorf("error checking existing user: %w", err)
// 	}

// 	// User exists, update it
// 	if existingUser != nil {
// 		updatedUser, err := s.client.User.FindUnique(
// 			db.User.ID.Equals(clerkUserID),
// 		).Update(
// 			db.User.Email.Set(email),
// 			db.User.FirstName.SetIfPresent(clerkUser.FirstName),
// 			db.User.LastName.SetIfPresent(clerkUser.LastName),
// 			db.User.ImageURL.SetIfPresent(imageUrl),
// 		).Exec(ctx)

// 		if err != nil {
// 			return nil, fmt.Errorf("failed to update user: %w", err)
// 		}

// 		return updatedUser, nil
// 	}

// 	// User doesn't exist, create new one
// 	newUser, err := s.client.User.CreateOne(
// 		db.User.ID.Set(clerkUserID),
// 		db.User.Email.Set(email),
// 		db.User.FirstName.SetIfPresent(clerkUser.FirstName),
// 		db.User.LastName.SetIfPresent(clerkUser.LastName),
// 		db.User.ImageURL.SetIfPresent(imageUrl),
// 	).Exec(ctx)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create user: %w", err)
// 	}

// 	return newUser, nil
// }

// // GetOrCreateUser ensures user exists in database
// func (s *UserService) GetOrCreateUser(ctx context.Context, clerkUserID string) (*db.UserModel, error) {
// 	// Try to get user from database first
// 	user, err := s.client.User.FindUnique(
// 		db.User.ID.Equals(clerkUserID),
// 	).Exec(ctx)

// 	if err == nil {
// 		return user, nil
// 	}

// 	if err == db.ErrNotFound {
// 		// User not in database, sync from Clerk
// 		return s.SyncUserFromClerk(ctx, clerkUserID)
// 	}

// 	return nil, fmt.Errorf("database error: %w", err)
// }
