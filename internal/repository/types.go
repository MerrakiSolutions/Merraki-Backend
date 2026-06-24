package repository

import "github.com/merraki/merraki-backend/internal/domain"

type ListFounderLeadsFilter struct {
	Status string
	Search string
	Page   int
	Limit  int
}

type ListFounderLeadsResult struct {
	Leads   []*domain.FounderLead
	Total   int
	Page    int
	Pages   int
	Limit   int
	HasNext bool
	HasPrev bool
}