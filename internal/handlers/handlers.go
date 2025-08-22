package handlers

type Handlers struct {
	User     *UserHandler
	Friend   *FriendHandler
	Settings *SettingsHandler
	Visitor  *VisitorHandler
	Upload   *UploadHandler
	Webhook  *WebhookHandler
	Invite   *InviteHandler
}