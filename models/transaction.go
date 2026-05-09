package models

import "time"

type Transaction struct {
	ID              string    `json:"id"`
	UserID          int64     `json:"user_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	TransactionType string    `json:"transaction_type"`
	CreatedAt       time.Time `json:"created_at"`
	Reference       string    `json:"references"`
}
