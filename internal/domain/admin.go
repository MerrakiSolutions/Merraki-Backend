package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Admin struct {
	ID             int64                  `db:"id" json:"id"`
	Email          string                 `db:"email" json:"email"`
	Name           string                 `db:"name" json:"name"`
	PasswordHash   string                 `db:"password_hash" json:"-"`
	Role           string                 `db:"role" json:"role"`
	Permissions    JSONMap                `db:"permissions" json:"permissions"`
	IsActive       bool                   `db:"is_active" json:"is_active"`
	LastLoginAt    *time.Time             `db:"last_login_at" json:"last_login_at,omitempty"`
	LastLoginIP    *string                `db:"last_login_ip" json:"last_login_ip,omitempty"`
	FailedAttempts int                    `db:"failed_attempts" json:"-"`
	LockedUntil    *time.Time             `db:"locked_until" json:"locked_until,omitempty"`
	CreatedAt      time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time              `db:"updated_at" json:"updated_at"`
}

type AdminSession struct {
	ID          int64      `db:"id" json:"id"`
	AdminID     int64      `db:"admin_id" json:"admin_id"`
	TokenHash   string     `db:"token_hash" json:"-"`
	DeviceName  *string    `db:"device_name" json:"device_name,omitempty"`
	IPAddress   *string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent   *string    `db:"user_agent" json:"user_agent,omitempty"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	ExpiresAt   time.Time  `db:"expires_at" json:"expires_at"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

type AdminLoginAttempt struct {
	ID            int64     `db:"id"`
	Email         string    `db:"email"`
	AdminID       *int64    `db:"admin_id"`
	AttemptType   string    `db:"attempt_type"`
	FailureReason *string   `db:"failure_reason"`
	IPAddress     *string   `db:"ip_address"`
	UserAgent     *string   `db:"user_agent"`
	CreatedAt     time.Time `db:"created_at"`
}

type ActivityLog struct {
	ID         int64      `db:"id"`
	AdminID    *int64     `db:"admin_id"`
	Action     string     `db:"action"`
	EntityType *string    `db:"entity_type"`
	EntityID   *int64     `db:"entity_id"`
	Details    JSONMap    `db:"details"`
	IPAddress  *string    `db:"ip_address"`
	CreatedAt  time.Time  `db:"created_at"`
}

// JSONMap for JSONB fields
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	
	return json.Unmarshal(bytes, j)
}

// StringArray for PostgreSQL arrays
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil || len(s) == 0 {
		return "{}", nil // Return empty PostgreSQL array
	}
	
	// Properly format PostgreSQL array
	var result string
	for i, str := range s {
		// Escape quotes and backslashes
		escaped := strings.ReplaceAll(str, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		
		if i == 0 {
			result = fmt.Sprintf(`"%s"`, escaped)
		} else {
			result += fmt.Sprintf(`, "%s"`, escaped)
		}
	}
	
	return fmt.Sprintf("{%s}", result), nil
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StringArray: expected []byte, got %T", value)
	}
	
	str := string(bytes)
	// Remove surrounding braces
	str = strings.Trim(str, "{}")
	
	if str == "" {
		*s = []string{}
		return nil
	}
	
	// Split by comma (simplified - doesn't handle escaped commas in strings)
	parts := strings.Split(str, ",")
	result := make([]string, len(parts))
	
	for i, part := range parts {
		// Remove quotes and trim whitespace
		part = strings.Trim(part, `" `)
		// Unescape
		part = strings.ReplaceAll(part, `\"`, `"`)
		part = strings.ReplaceAll(part, `\\`, `\`)
		result[i] = part
	}
	
	*s = result
	return nil
}