// internal/services/services.go
package services

type Services struct {
	User     *UserService
	Friend   *FriendService
	Settings *SettingsService
	Visitor  *VisitorService
}