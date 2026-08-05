package users

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type MeResponse struct {
	User User `json:"user"`
}

type UpdateRequest struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
}
