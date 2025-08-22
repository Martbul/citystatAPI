
// internal/repository/friend.go
package repository

import (
	"context"
	"fmt"

	"citystatAPI/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type FriendRepository interface {
	GetUserFriends(ctx context.Context, userID string) ([]db.UserFriend, error)
	CreateFriend(ctx context.Context, params CreateFriendParams) (*db.UserFriend, error)
	CheckFriendshipExists(ctx context.Context, userID, friendID string) (bool, error)
	DeleteFriend(ctx context.Context, userID, friendID string) error
}

type CreateFriendParams struct {
	UserID    string
	FriendID  string
	UserName  string
	FirstName *string
	LastName  *string
	ImageURL  *string
}

type friendRepository struct {
	queries *db.Queries
}

func NewFriendRepository(queries *db.Queries) FriendRepository {
	return &friendRepository{queries: queries}
}

func (r *friendRepository) GetUserFriends(ctx context.Context, userID string) ([]db.UserFriend, error) {
	friends, err := r.queries.GetUserFriends(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user friends: %w", err)
	}
	return friends, nil
}

func (r *friendRepository) CreateFriend(ctx context.Context, params CreateFriendParams) (*db.UserFriend, error) {
	sqlcParams := db.CreateFriendParams{
		UserID:    params.UserID,
		FriendID:  params.FriendID,
		UserName:  params.UserName,
		FirstName: pgtype.Text{String: stringValue(params.FirstName), Valid: params.FirstName != nil},
		LastName:  pgtype.Text{String: stringValue(params.LastName), Valid: params.LastName != nil},
		ImageUrl:  pgtype.Text{String: stringValue(params.ImageURL), Valid: params.ImageURL != nil},
	}

	friend, err := r.queries.CreateFriend(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create friend: %w", err)
	}
	return &friend, nil
}

func (r *friendRepository) CheckFriendshipExists(ctx context.Context, userID, friendID string) (bool, error) {
	count, err := r.queries.CheckFriendshipExists(ctx, db.CheckFriendshipExistsParams{
		UserID:   userID,
		FriendID: friendID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check friendship: %w", err)
	}
	return count > 0, nil
}

func (r *friendRepository) DeleteFriend(ctx context.Context, userID, friendID string) error {
	if err := r.queries.DeleteFriend(ctx, db.DeleteFriendParams{
		UserID:   userID,
		FriendID: friendID,
	}); err != nil {
		return fmt.Errorf("failed to delete friend: %w", err)
	}
	return nil
}
