package domain

import "time"

type PaginationParams struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

func (p *PaginationParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
}

func (p *PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

type Timestamps struct {
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}