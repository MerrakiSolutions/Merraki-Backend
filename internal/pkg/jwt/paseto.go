package jwt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/o1egl/paseto"
)

type TokenClaims struct {
	AdminID     int64                  `json:"admin_id,omitempty"`
	Email       string                 `json:"email"`
	Role        string                 `json:"role"`
	Permissions map[string]interface{} `json:"permissions"`
	Type        string                 `json:"type"` // "access" or "admin_access"
	ExpiresAt   time.Time              `json:"exp"`
	IssuedAt    time.Time              `json:"iat"`
}

type PasetoMaker struct {
	symmetricKey []byte
}

func NewPasetoMaker(symmetricKey string) (*PasetoMaker, error) {
	if len(symmetricKey) != 64 {
		return nil, fmt.Errorf("symmetric key must be exactly 64 characters")
	}

	decodedKey, err := hex.DecodeString(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	if len(decodedKey) != 32 {
		return nil, fmt.Errorf("decoded key must be 32 bytes")
	}

	return &PasetoMaker{
		symmetricKey: decodedKey,
	}, nil
}

func (maker *PasetoMaker) CreateToken(claims TokenClaims) (string, error) {
	now := time.Now()
	claims.IssuedAt = now

	token, err := paseto.NewV2().Encrypt(maker.symmetricKey, claims, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return token, nil
}

func (maker *PasetoMaker) VerifyToken(token string) (*TokenClaims, error) {
	var claims TokenClaims

	err := paseto.NewV2().Decrypt(token, maker.symmetricKey, &claims, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if time.Now().After(claims.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

func (maker *PasetoMaker) CreateAccessToken(adminID int64, email, role string, permissions map[string]interface{}, duration time.Duration, tokenType string) (string, error) {
	claims := TokenClaims{
		AdminID:     adminID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		Type:        tokenType,
		ExpiresAt:   time.Now().Add(duration),
	}

	return maker.CreateToken(claims)
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
