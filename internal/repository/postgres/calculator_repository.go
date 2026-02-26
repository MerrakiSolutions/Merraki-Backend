package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type CalculatorRepository struct {
	db *Database
}

func NewCalculatorRepository(db *Database) *CalculatorRepository {
	return &CalculatorRepository{db: db}
}

func (r *CalculatorRepository) Create(ctx context.Context, result *domain.CalculatorResult) error {
	query := `
		INSERT INTO calculator_results (calculator_type, email, inputs, results, saved_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		result.CalculatorType, result.Email, result.Inputs, result.Results, result.SavedName,
	).Scan(&result.ID, &result.CreatedAt)
}

func (r *CalculatorRepository) FindByID(ctx context.Context, id int64) (*domain.CalculatorResult, error) {
	var result domain.CalculatorResult
	query := `SELECT * FROM calculator_results WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &result, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &result, err
}

func (r *CalculatorRepository) GetByEmail(ctx context.Context, email string, calculatorType string) ([]*domain.CalculatorResult, error) {
	var results []*domain.CalculatorResult
	query := `SELECT * FROM calculator_results WHERE email = $1`

	args := []interface{}{email}
	if calculatorType != "" {
		query += ` AND calculator_type = $2`
		args = append(args, calculatorType)
	}

	query += ` ORDER BY created_at DESC`

	err := r.db.DB.SelectContext(ctx, &results, query, args...)
	return results, err
}

func (r *CalculatorRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.CalculatorResult, int, error) {
	var results []*domain.CalculatorResult
	
	query := `SELECT * FROM calculator_results WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM calculator_results WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if calcType, ok := filters["calculator_type"].(string); ok && calcType != "" {
		query += fmt.Sprintf(" AND calculator_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND calculator_type = $%d", argCount)
		args = append(args, calcType)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND email ILIKE $%d", argCount)
		countQuery += fmt.Sprintf(" AND email ILIKE $%d", argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)

	var total int
	if err := r.db.DB.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	if err := r.db.DB.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (r *CalculatorRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var usageByType []struct {
		CalculatorType string `db:"calculator_type"`
		Count          int    `db:"count"`
	}
	usageQuery := `
		SELECT calculator_type, COUNT(*) as count 
		FROM calculator_results 
		GROUP BY calculator_type`
	
	if err := r.db.DB.SelectContext(ctx, &usageByType, usageQuery); err != nil {
		return nil, err
	}
	analytics["usage_by_type"] = usageByType

	var total int
	totalQuery := `SELECT COUNT(*) FROM calculator_results`
	if err := r.db.DB.GetContext(ctx, &total, totalQuery); err != nil {
		return nil, err
	}
	analytics["total_calculations"] = total

	return analytics, nil
}

