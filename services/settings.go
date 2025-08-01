package services

import (
	"citystatAPI/prisma/db"
	"context"
	"fmt"
)


type SettingsService struct {
	client *db.PrismaClient
}


func NewSettingsService(client *db.PrismaClient) *SettingsService {
	return &SettingsService{client: client}
}

func (s *SettingsService) EditUsername(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
    username, ok := updates["username"].(string)
    if !ok {
        return nil, fmt.Errorf("username field is required and must be a string")
    }
    
    updatedUser, err := s.client.User.FindUnique(
        db.User.ID.Equals(clerkUserID),
    ).Update(
        db.User.UserName.Set(username),
    ).Exec(ctx)
    
    if err != nil {
        return nil, fmt.Errorf("failed to update username: %w", err)
    }
    
    return updatedUser, nil
}