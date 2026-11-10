package users

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
	AvatarURL string `json:"avatar_url"`
	Locale    string `json:"locale"`
}

type MeResponse struct {
	User User `json:"user"`
}

// UpdateRequest changes the display name. Email is accepted only to refuse it
// with a clear message: the address comes from Registre.
type UpdateRequest struct {
	Email *string `json:"email"`
	Name  *string `json:"name"`
}
