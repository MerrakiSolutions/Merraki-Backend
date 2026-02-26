package domain

import "time"

type CalculatorResult struct {
	ID             int64     `db:"id" json:"id"`
	CalculatorType string    `db:"calculator_type" json:"calculator_type"`
	Email          *string   `db:"email" json:"email,omitempty"`
	Inputs         JSONMap   `db:"inputs" json:"inputs"`
	Results        JSONMap   `db:"results" json:"results"`
	SavedName      *string   `db:"saved_name" json:"saved_name,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}