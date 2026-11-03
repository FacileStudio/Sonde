package users

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
	AvatarURL string `json:"avatar_url"`
}

type MeResponse struct {
	User User `json:"user"`
}

type UpdateRequest struct {
	Email *string `json:"email"`
	Name  *string `json:"name"`
}
