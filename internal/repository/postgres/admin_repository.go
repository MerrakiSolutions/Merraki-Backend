package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
)

type AdminRepository struct {
	db *Database
}

func NewAdminRepository(db *Database) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) Create(ctx context.Context, admin *domain.Admin) error {
	query := `
		INSERT INTO admins (email, name, password_hash, role, permissions, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.db.DB.QueryRowContext(
		ctx, query,
		admin.Email, admin.Name, admin.PasswordHash,
		admin.Role, admin.Permissions, admin.IsActive,
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)

	return err
}

func (r *AdminRepository) FindByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	var admin domain.Admin
	query := `SELECT * FROM admins WHERE email = $1`

	err := r.db.DB.GetContext(ctx, &admin, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &admin, err
}

func (r *AdminRepository) FindByID(ctx context.Context, id int64) (*domain.Admin, error) {
	var admin domain.Admin
	query := `SELECT * FROM admins WHERE id = $1`

	err := r.db.DB.GetContext(ctx, &admin, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &admin, err
}

func (r *AdminRepository) Update(ctx context.Context, admin *domain.Admin) error {
	query := `
		UPDATE admins 
		SET name = $1, role = $2, permissions = $3, is_active = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`

	return r.db.DB.QueryRowContext(
		ctx, query,
		admin.Name, admin.Role, admin.Permissions, admin.IsActive, admin.ID,
	).Scan(&admin.UpdatedAt)
}

func (r *AdminRepository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	query := `UPDATE admins SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.DB.ExecContext(ctx, query, passwordHash, id)
	return err
}

func (r *AdminRepository) UpdateLastLogin(ctx context.Context, id int64, ip string) error {
	query := `
		UPDATE admins 
		SET last_login_at = NOW(), last_login_ip = $1, failed_attempts = 0, updated_at = NOW()
		WHERE id = $2`

	_, err := r.db.DB.ExecContext(ctx, query, ip, id)
	return err
}

func (r *AdminRepository) IncrementFailedAttempts(ctx context.Context, id int64) error {
	query := `UPDATE admins SET failed_attempts = failed_attempts + 1 WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}

func (r *AdminRepository) LockAccount(ctx context.Context, id int64, until time.Time) error {
	query := `UPDATE admins SET locked_until = $1 WHERE id = $2`
	_, err := r.db.DB.ExecContext(ctx, query, until, id)
	return err
}

func (r *AdminRepository) GetAll(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Admin, int, error) {
	var admins []*domain.Admin
	
	query := `SELECT * FROM admins WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM admins WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if role, ok := filters["role"].(string); ok && role != "" {
		query += fmt.Sprintf(" AND role = $%d", argCount)
		countQuery += fmt.Sprintf(" AND role = $%d", argCount)
		args = append(args, role)
		argCount++
	}

	if isActive, ok := filters["is_active"].(bool); ok {
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		countQuery += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, isActive)
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
	if err := r.db.DB.SelectContext(ctx, &admins, query, args...); err != nil {
		return nil, 0, err
	}

	return admins, total, nil
}

func (r *AdminRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM admins WHERE id = $1`
	_, err := r.db.DB.ExecContext(ctx, query, id)
	return err
}