
// internal/repository/user.go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"citystatAPI/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository interface {
	GetUser(ctx context.Context, id string) (*db.User, error)
	CreateUser(ctx context.Context, params CreateUserParams) (*db.User, error)
	UpdateUser(ctx context.Context, params UpdateUserParams) (*db.User, error)
	SearchUsers(ctx context.Context, currentUserID, username string) ([]db.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type CreateUserParams struct {
	ID          string
	Email       string
	FirstName   *string
	LastName    *string
	UserName    *string
	ImageURL    *string
	PhoneNumber *string
	Role        *db.UserRole
	Status      *db.UserStatus
}

type UpdateUserParams struct {
	ID                string
	FirstName         *string
	LastName          *string
	UserName          *string
	ImageURL          *string
	PhoneNumber       *string
	CompletedTutorial *bool
	AboutMe           *string
	Note              *string
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(queries *db.Queries) UserRepository {
	return &userRepository{queries: queries}
}

func (r *userRepository) GetUser(ctx context.Context, id string) (*db.User, error) {
	user, err := r.queries.GetUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, params CreateUserParams) (*db.User, error) {
	// Convert to SQLC params
	sqlcParams := db.CreateUserParams{
		ID:          params.ID,
		Email:       params.Email,
		FirstName:   pgtype.Text{String: stringValue(params.FirstName), Valid: params.FirstName != nil},
		LastName:    pgtype.Text{String: stringValue(params.LastName), Valid: params.LastName != nil},
		UserName:    pgtype.Text{String: stringValue(params.UserName), Valid: params.UserName != nil},
		ImageUrl:    pgtype.Text{String: stringValue(params.ImageURL), Valid: params.ImageURL != nil},
		PhoneNumber: pgtype.Text{String: stringValue(params.PhoneNumber), Valid: params.PhoneNumber != nil},
		Role:        db.NullUserRole{UserRole: db.UserRole("USER"), Valid: true},
		Status:      db.NullUserStatus{UserStatus: db.UserStatus("ACTIVE"), Valid: true},
	}

	if params.Role != nil {
	sqlcParams.Role = db.NullUserRole{UserRole: *params.Role, Valid: true}
}
if params.Status != nil {
	sqlcParams.Status = db.NullUserStatus{UserStatus: *params.Status, Valid: true}
}

	user, err := r.queries.CreateUser(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, params UpdateUserParams) (*db.User, error) {
	sqlcParams := db.UpdateUserParams{
		ID:                params.ID,
		FirstName:         pgtype.Text{String: stringValue(params.FirstName), Valid: params.FirstName != nil},
		LastName:          pgtype.Text{String: stringValue(params.LastName), Valid: params.LastName != nil},
		UserName:          pgtype.Text{String: stringValue(params.UserName), Valid: params.UserName != nil},
		ImageUrl:          pgtype.Text{String: stringValue(params.ImageURL), Valid: params.ImageURL != nil},
		PhoneNumber:       pgtype.Text{String: stringValue(params.PhoneNumber), Valid: params.PhoneNumber != nil},
		CompletedTutorial: pgtype.Bool{Bool: boolValue(params.CompletedTutorial), Valid: params.CompletedTutorial != nil},
		AboutMe:           pgtype.Text{String: stringValue(params.AboutMe), Valid: params.AboutMe != nil},
		Note:              pgtype.Text{String: stringValue(params.Note), Valid: params.Note != nil},
	}

	user, err := r.queries.UpdateUser(ctx, sqlcParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) SearchUsers(ctx context.Context, currentUserID, username string) ([]db.User, error) {
	users, err := r.queries.SearchUsers(ctx, db.SearchUsersParams{
		ID:       currentUserID,
		UserName: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	if err := r.queries.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Helper functions
func stringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func boolValue(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}