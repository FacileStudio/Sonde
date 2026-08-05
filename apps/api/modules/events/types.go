package events

import "time"

type Event struct {
	ID      int64     `json:"id"`
	Name    string    `json:"name"`
	Starts  time.Time `json:"starts"`
	Ends    time.Time `json:"ends"`
	OwnerID string    `json:"owner_id"`
}

type CreateRequest struct {
	Name   string    `json:"name"`
	Starts time.Time `json:"starts"`
	Ends   time.Time `json:"ends"`
}

type CreateResponse struct {
	ID int64 `json:"id"`
}

type ListResponse struct {
	Events []*Event `json:"events"`
}
