package tickets

import "time"

type Ticket struct {
	ID      int64      `json:"id"`
	EventID int64      `json:"event_id"`
	Status  string     `json:"status"`
	UsedAt  *time.Time `json:"used_at,omitempty"`
}

type GenerateResponse struct {
	Code   string `json:"code"`
	Ticket Ticket `json:"ticket"`
}

type ValidateRequest struct {
	Code string `json:"code"`
}

type ValidateResponse struct {
	Valid  bool   `json:"valid"`
	Status string `json:"status"`
	Ticket Ticket `json:"ticket"`
}

type CheckInRequest struct {
	Code string `json:"code"`
}

type CheckInResponse struct {
	Ticket Ticket `json:"ticket"`
}
