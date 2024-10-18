package twitch

type UserInfo struct {
	Id              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImg      string `json:"profile_image_url"`
	OfflineImg      string `json:"offline_image_url"`
	CreatedAt       string `json:"created_at"`
}

type TwitchResponse[T any] struct {
	Data []T `json:"data"`
}
type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	Expire       int32    `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	Scope        []string `json:"scope"`
}
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Expire       int32  `json:"expires_in"`
}
type tokenError struct {
	Status  int16  `json:"status"`
	Message string `json:"message"`
}
