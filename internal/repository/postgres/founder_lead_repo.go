package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/merraki/merraki-backend/internal/domain"
)

// ============================================================================
// FOUNDER LEAD REPOSITORY
// ============================================================================

type FounderLeadRepository struct {
	db *sqlx.DB  // was *sql.DB
}

func NewFounderLeadRepository(db *sqlx.DB) *FounderLeadRepository {
	return &FounderLeadRepository{db: db}
}

// ============================================================================
// HELPERS
// ============================================================================

func marshalSectionScores(scores []domain.SectionScore) ([]byte, error) {
	if scores == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(scores)
}

func scanFounderLead(row interface {
	Scan(dest ...any) error
}) (*domain.FounderLead, error) {
	var l domain.FounderLead
	var scoresJSON []byte
	var company, role, notes sql.NullString

	err := row.Scan(
		&l.ID, &l.Name, &l.Email, &company, &role, &l.IPAddress,
		&l.TotalScore, &l.TotalMax,
		&l.PersonalityType, &l.PersonalityTitle, &l.PersonalityBadge,
		&l.PersonalityColor, &l.PersonalityDescription,
		&scoresJSON,
		&l.Status, &notes,
		&l.SubmittedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if company.Valid {
		l.Company = company.String
	}
	if role.Valid {
		l.Role = role.String
	}
	if notes.Valid {
		l.Notes = notes.String
	}

	if err := json.Unmarshal(scoresJSON, &l.SectionScores); err != nil {
		l.SectionScores = []domain.SectionScore{}
	}

	return &l, nil
}

// ============================================================================
// CREATE
// ============================================================================

func (r *FounderLeadRepository) Create(ctx context.Context, req *domain.SubmitFounderLeadRequest, ip string) (*domain.FounderLead, error) {
	scoresJSON, err := marshalSectionScores(req.SectionScores)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal section scores: %w", err)
	}

	const q = `
		INSERT INTO founders_test_leads (
			name, email, company, role, ip_address,
			total_score, total_max,
			personality_type, personality_title, personality_badge,
			personality_color, personality_description,
			section_scores,
			status
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12,
			$13,
			'new'
		)
		RETURNING
			id, name, email, company, role, ip_address,
			total_score, total_max,
			personality_type, personality_title, personality_badge,
			personality_color, personality_description,
			section_scores,
			status, notes,
			submitted_at, updated_at`

	row := r.db.QueryRowContext(ctx, q,
		req.Name, req.Email,
		nullString(req.Company), nullString(req.Role),
		nullString(ip),
		req.TotalScore, req.TotalMax,
		req.PersonalityType, req.PersonalityTitle, req.PersonalityBadge,
		req.PersonalityColor, req.PersonalityDescription,
		scoresJSON,
	)

	return scanFounderLead(row)
}

// ============================================================================
// READ - List
// ============================================================================

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

func (r *FounderLeadRepository) List(ctx context.Context, f ListFounderLeadsFilter) (*ListFounderLeadsResult, error) {
	args := []any{}
	idx := 1
	where := []string{}

	// Filter by status
	if f.Status != "" && f.Status != "all" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}

	// Search by name, email, company
	if f.Search != "" {
		like := "%" + strings.ToLower(f.Search) + "%"
		where = append(where,
			fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(email) LIKE $%d OR LOWER(company) LIKE $%d)",
				idx, idx+1, idx+2))
		args = append(args, like, like, like)
		idx += 3
	}

	clause := ""
	if len(where) > 0 {
		clause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM founders_test_leads %s", clause)
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	if f.Limit <= 0 {
		f.Limit = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}

	offset := (f.Page - 1) * f.Limit
	pages := (total + f.Limit - 1) / f.Limit

	// List query
	listQ := fmt.Sprintf(`
		SELECT
			id, name, email, company, role, ip_address,
			total_score, total_max,
			personality_type, personality_title, personality_badge,
			personality_color, personality_description,
			section_scores,
			status, notes,
			submitted_at, updated_at
		FROM founders_test_leads
		%s
		ORDER BY submitted_at DESC
		LIMIT $%d OFFSET $%d`,
		clause, idx, idx+1)

	args = append(args, f.Limit, offset)
	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var leads []*domain.FounderLead
	for rows.Next() {
		l, err := scanFounderLead(rows)
		if err != nil {
			continue
		}
		leads = append(leads, l)
	}

	return &ListFounderLeadsResult{
		Leads:   leads,
		Total:   total,
		Page:    f.Page,
		Pages:   pages,
		Limit:   f.Limit,
		HasNext: f.Page < pages,
		HasPrev: f.Page > 1,
	}, nil
}

// ============================================================================
// READ - Get By ID
// ============================================================================

func (r *FounderLeadRepository) GetByID(ctx context.Context, id int64) (*domain.FounderLead, error) {
	const q = `
		SELECT
			id, name, email, company, role, ip_address,
			total_score, total_max,
			personality_type, personality_title, personality_badge,
			personality_color, personality_description,
			section_scores,
			status, notes,
			submitted_at, updated_at
		FROM founders_test_leads
		WHERE id = $1`

	return scanFounderLead(r.db.QueryRowContext(ctx, q, id))
}

// ============================================================================
// UPDATE - Patch Status
// ============================================================================

func (r *FounderLeadRepository) UpdateStatus(ctx context.Context, id int64, status domain.LeadStatus) (*domain.FounderLead, error) {
	const q = `
		UPDATE founders_test_leads
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING
			id, name, email, company, role, ip_address,
			total_score, total_max,
			personality_type, personality_title, personality_badge,
			personality_color, personality_description,
			section_scores,
			status, notes,
			submitted_at, updated_at`

	return scanFounderLead(r.db.QueryRowContext(ctx, q, status, id))
}

// ============================================================================
// DELETE
// ============================================================================

func (r *FounderLeadRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM founders_test_leads WHERE id = $1", id)
	return err
}

// ============================================================================
// STATS
// ============================================================================

func (r *FounderLeadRepository) GetStats(ctx context.Context) (*domain.FounderLeadStats, error) {
	const q = `
		SELECT
			COUNT(*)                                                              AS total,
			COUNT(*) FILTER (WHERE submitted_at >= NOW() - INTERVAL '7 days')    AS new_this_week,
			COUNT(*) FILTER (WHERE status IN ('contacted'))                      AS contacted,
			COUNT(*) FILTER (WHERE status IN ('qualified'))                      AS qualified,
			COUNT(*) FILTER (WHERE status IN ('converted'))                      AS converted,
			COALESCE(ROUND(AVG(total_score::numeric / NULLIF(total_max,0) * 100)::numeric, 1), 0) AS avg_score
		FROM founders_test_leads`

	var stats domain.FounderLeadStats
	err := r.db.QueryRowContext(ctx, q).Scan(
		&stats.Total,
		&stats.NewThisWeek,
		&stats.Contacted,
		&stats.Qualified,
		&stats.Converted,
		&stats.AvgScore,
	)
	if err != nil {
		return nil, fmt.Errorf("stats query failed: %w", err)
	}

	// Calculate conversion rate
	if stats.Total > 0 {
		stats.ConversionRate = (float64(stats.Converted) / float64(stats.Total)) * 100
	}

	return &stats, nil
}

// ============================================================================
// EXPORT - Get All (no pagination)
// ============================================================================

func (r *FounderLeadRepository) ListAll(ctx context.Context, f ListFounderLeadsFilter) ([]*domain.FounderLead, error) {
	f.Page = 1
	f.Limit = 10000

	res, err := r.List(ctx, f)
	if err != nil {
		return nil, err
	}

	return res.Leads, nil
}

// ============================================================================
// HELPER
// ============================================================================

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
