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

func (s *SettingsService) GetUserSettings(ctx context.Context, clerkUserID string) (*db.SettingsModel, error) {
    settings, err := s.client.Settings.FindUnique(
            db.Settings.UserID.Equals(clerkUserID),
        ).Exec(ctx)

    if err != nil {
        return nil, fmt.Errorf("failed to retrieve user settings: %w", err)
    }
    if settings == nil {
        return nil, fmt.Errorf("settings not found for user ID: %s", clerkUserID)
    }
    return settings, nil
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

//TODO: add validation and error hanling also in the client

func (s *SettingsService) EditPhoneNumber(ctx context.Context, clerkUserID string, updates map[string]interface{}) (*db.UserModel, error) {
    phoneNumber, ok := updates["phone"].(string)
	fmt.Println(phoneNumber)
    if !ok {
        return nil, fmt.Errorf("username field is required and must be a string")
    }
    
    updatedUser, err := s.client.User.FindUnique(
        db.User.ID.Equals(clerkUserID),
    ).Update(
        db.User.PhoneNumber.Set(phoneNumber),
    ).Exec(ctx)
    
    if err != nil {
        return nil, fmt.Errorf("failed to update phone number: %w", err)
    }
    
    return updatedUser, nil
}