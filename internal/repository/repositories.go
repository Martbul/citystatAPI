package repository

type Repositories struct {
	User     UserRepository
	Friend   FriendRepository
	Settings SettingsRepository
	Visitor  VisitorRepository
}
