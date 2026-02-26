package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
)

type TestRepository struct {
	db *Database
}

func NewTestRepository(db *Database) *TestRepository {
	return &TestRepository{db: db}
}

// Questions
func (r *TestRepository) CreateQuestion(ctx context.Context, question *domain.TestQuestion) error {
	query := `
		INSERT INTO test_questions 
		(question_text, question_type, options, category, weight, order_number, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		question.QuestionText, question.QuestionType, question.Options,
		question.Category, question.Weight, question.OrderNumber, question.IsActive,
	).Scan(&question.ID, &question.CreatedAt, &question.UpdatedAt)
}

func (r *TestRepository) GetAllQuestions(ctx context.Context, activeOnly bool) ([]*domain.TestQuestion, error) {
	var questions []*domain.TestQuestion
	query := `SELECT * FROM test_questions WHERE 1=1`

	if activeOnly {
		query += ` AND is_active = true`
	}

	query += ` ORDER BY order_number ASC`

	err := r.db.DB.SelectContext(ctx, &questions, query)
	return questions, err
}

func (r *TestRepository) GetQuestionByID(ctx context.Context, id int64) (*domain.TestQuestion, error) {
	var question domain.TestQuestion
	query := `SELECT * FROM test_questions WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &question, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &question, err
}

func (r *TestRepository) UpdateQuestion(ctx context.Context, question *domain.TestQuestion) error {
	query := `
		UPDATE test_questions 
		SET question_text = $1, question_type = $2, options = $3, category = $4,
		    weight = $5, order_number = $6, is_active = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		question.QuestionText, question.QuestionType, question.Options, question.Category,
		question.Weight, question.OrderNumber, question.IsActive, question.ID,
	).Scan(&question.UpdatedAt)
}

func (r *TestRepository) DeleteQuestion(ctx context.Context, id int64) error {
	query := `DELETE FROM test_questions WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

// Submissions
func (r *TestRepository) CreateSubmission(ctx context.Context, submission *domain.TestSubmission) error {
	query := `
		INSERT INTO test_submissions 
		(test_number, email, name, responses, personality_type, risk_appetite, scores, report_url, report_sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		submission.TestNumber, submission.Email, submission.Name, submission.Responses,
		submission.PersonalityType, submission.RiskAppetite, submission.Scores,
		submission.ReportURL, submission.ReportSent,
	).Scan(&submission.ID, &submission.CreatedAt)
}

func (r *TestRepository) FindSubmissionByTestNumber(ctx context.Context, testNumber string) (*domain.TestSubmission, error) {
	var submission domain.TestSubmission
	query := `SELECT * FROM test_submissions WHERE test_number = $1`

	err := r.db.DB.GetContext(ctx, &submission, query, testNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &submission, err
}

func (r *TestRepository) GetAllSubmissions(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.TestSubmission, int, error) {
	var submissions []*domain.TestSubmission
	
	query := `SELECT * FROM test_submissions WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM test_submissions WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if personalityType, ok := filters["personality_type"].(string); ok && personalityType != "" {
		query += fmt.Sprintf(" AND personality_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND personality_type = $%d", argCount)
		args = append(args, personalityType)
		argCount++
	}

	if search, ok := filters["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d)", argCount, argCount)
		countQuery += fmt.Sprintf(" AND (email ILIKE $%d OR name ILIKE $%d)", argCount, argCount)
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
	if err := r.db.DB.SelectContext(ctx, &submissions, query, args...); err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}

func (r *TestRepository) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	var total int
	totalQuery := `SELECT COUNT(*) FROM test_submissions`
	if err := r.db.DB.GetContext(ctx, &total, totalQuery); err != nil {
		return nil, err
	}
	analytics["total_submissions"] = total

	var personalityDist []struct {
		PersonalityType string `db:"personality_type"`
		Count           int    `db:"count"`
	}
	personalityQuery := `
		SELECT personality_type, COUNT(*) as count 
		FROM test_submissions 
		WHERE personality_type IS NOT NULL
		GROUP BY personality_type
		ORDER BY count DESC`
	
	if err := r.db.DB.SelectContext(ctx, &personalityDist, personalityQuery); err != nil {
		return nil, err
	}
	analytics["personality_distribution"] = personalityDist

	return analytics, nil
}