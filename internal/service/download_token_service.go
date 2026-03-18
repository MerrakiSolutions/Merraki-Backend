package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// DOWNLOAD TOKEN SERVICE - Secure download system
// ============================================================================

type DownloadTokenService struct {
	tokenRepo      repository.DownloadTokenRepository
	downloadRepo   repository.DownloadRepository
	orderRepo      repository.OrderRepository
	orderItemRepo  repository.OrderItemRepository
	templateRepo   repository.TemplateRepository
	storageService *StorageService
}

func NewDownloadTokenService(
	tokenRepo repository.DownloadTokenRepository,
	downloadRepo repository.DownloadRepository,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	templateRepo repository.TemplateRepository,
	storageService *StorageService,
) *DownloadTokenService {
	return &DownloadTokenService{
		tokenRepo:      tokenRepo,
		downloadRepo:   downloadRepo,
		orderRepo:      orderRepo,
		orderItemRepo:  orderItemRepo,
		templateRepo:   templateRepo,
		storageService: storageService,
	}
}

// ============================================================================
// GENERATE TOKENS - Called after order approval
// ============================================================================

func (s *DownloadTokenService) GenerateTokensForOrder(ctx context.Context, orderID int64) error {
	// 1. Get order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Validate order is approved
	if order.Status != domain.OrderStatusApproved {
		return fmt.Errorf("order must be approved to generate tokens")
	}

	if !order.DownloadsEnabled {
		return fmt.Errorf("downloads not enabled for this order")
	}

	// 3. Get order items
	items, err := s.orderItemRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	// 4. Generate token for each item
	for _, item := range items {
		// Check if token already exists
		existing, _ := s.tokenRepo.GetByOrderID(ctx, orderID)
		alreadyExists := false
		for _, token := range existing {
			if token.OrderItemID == item.ID && !token.IsRevoked {
				alreadyExists = true
				break
			}
		}

		if alreadyExists {
			logger.Info("Token already exists for order item",
				zap.Int64("order_id", orderID),
				zap.Int64("item_id", item.ID),
			)
			continue
		}

		// Generate cryptographically secure token
		token := s.generateSecureToken()

		// Set expiration (use order's download expiry)
		expiresAt := time.Now().AddDate(0, 0, 30) // Default 30 days
		if order.DownloadsExpiresAt != nil {
			expiresAt = *order.DownloadsExpiresAt
		}

		downloadToken := &domain.DownloadToken{
			Token:         token,
			OrderID:       orderID,
			OrderItemID:   item.ID,
			TemplateID:    item.TemplateID,
			CustomerEmail: order.CustomerEmail,
			ExpiresAt:     expiresAt,
			MaxDownloads:  5, // Allow 5 downloads per token
		}

		if err := s.tokenRepo.Create(ctx, downloadToken); err != nil {
			logger.Error("Failed to create download token",
				zap.Int64("order_id", orderID),
				zap.Int64("item_id", item.ID),
				zap.Error(err),
			)
			continue
		}

		logger.Info("Download token generated",
			zap.Int64("order_id", orderID),
			zap.Int64("item_id", item.ID),
			zap.String("token", token[:16]+"..."), // Log partial token
		)
	}

	return nil
}

// ============================================================================
// VALIDATE TOKEN & INITIATE DOWNLOAD
// ============================================================================

type InitiateDownloadRequest struct {
	Token     string `json:"token" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	IPAddress string `json:"-"` // From request context
	UserAgent string `json:"-"` // From request context
	Country   string `json:"-"` // From geo-IP lookup
}

type DownloadResponse struct {
	DownloadURL string    `json:"download_url"`
	ExpiresAt   time.Time `json:"expires_at"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size_bytes"`
}

func (s *DownloadTokenService) InitiateDownload(ctx context.Context, req *InitiateDownloadRequest) (*DownloadResponse, error) {
	// 1. Find token
	token, err := s.tokenRepo.FindByToken(ctx, req.Token)
	if err != nil {
		logger.Warn("Invalid download token",
			zap.String("token", req.Token[:16]+"..."),
		)
		return nil, domain.ErrNotFound
	}

	// 2. Validate token
	if err := s.validateToken(token, req.Email); err != nil {
		return nil, err
	}

	// 3. Get order item (for file details)
	item, err := s.orderItemRepo.GetByID(ctx, token.OrderItemID)
	if err != nil {
		return nil, err
	}

	// 4. Get template (for current file URL)
	template, err := s.templateRepo.FindByID(ctx, token.TemplateID)
	if err != nil {
		return nil, err
	}

	// Ensure file exists
	if template.FileURL == nil || *template.FileURL == "" {
		return nil, fmt.Errorf("file not available")
	}

	// 5. Create download log
	download := &domain.Download{
		TokenID:       token.ID,
		OrderID:       token.OrderID,
		OrderItemID:   token.OrderItemID,
		TemplateID:    token.TemplateID,
		CustomerEmail: req.Email,
		IPAddress:     &req.IPAddress,
		UserAgent:     &req.UserAgent,
		Country:       &req.Country,
		FileURL:       template.FileURL,
	}

	if err := s.downloadRepo.Create(ctx, download); err != nil {
		logger.Error("Failed to create download log", zap.Error(err))
		// Continue - logging is not critical
	}

	// 6. Increment download count
	if err := s.tokenRepo.IncrementDownloadCount(ctx, token.ID); err != nil {
		logger.Error("Failed to increment token download count", zap.Error(err))
	}

	if err := s.orderItemRepo.IncrementDownloadCount(ctx, item.ID); err != nil {
		logger.Error("Failed to increment item download count", zap.Error(err))
	}

	if err := s.templateRepo.IncrementDownloads(ctx, template.ID); err != nil {
		logger.Error("Failed to increment template download count", zap.Error(err))
	}

	// 7. Generate signed URL (short-lived)
	signedURL, err := s.storageService.GenerateSignedDownloadURL(ctx, *template.FileURL, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	// 8. Prepare response
	fileName := fmt.Sprintf("%s_%s.%s", template.Slug, template.CurrentVersion, getFileExtension(*template.FileFormat))
	fileSize := int64(0)
	if template.FileSizeMB != nil {
		fileSize = int64(*template.FileSizeMB * 1024 * 1024)
	}

	response := &DownloadResponse{
		DownloadURL: signedURL,
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		FileName:    fileName,
		FileSize:    fileSize,
	}

	logger.Info("Download initiated",
		zap.String("token", req.Token[:16]+"..."),
		zap.String("email", req.Email),
		zap.Int64("template_id", template.ID),
		zap.String("ip", req.IPAddress),
	)

	return response, nil
}

// ============================================================================
// TOKEN VALIDATION
// ============================================================================

func (s *DownloadTokenService) validateToken(token *domain.DownloadToken, email string) error {
	// Check if revoked
	if token.IsRevoked {
		logger.Warn("Revoked token used",
			zap.String("token", token.Token[:16]+"..."),
			zap.String("email", email),
		)
		return fmt.Errorf("token has been revoked")
	}

	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		logger.Warn("Expired token used",
			zap.String("token", token.Token[:16]+"..."),
			zap.String("email", email),
		)
		return fmt.Errorf("token has expired")
	}

	// Check download limit
	if token.DownloadCount >= token.MaxDownloads {
		logger.Warn("Download limit exceeded",
			zap.String("token", token.Token[:16]+"..."),
			zap.String("email", email),
			zap.Int("count", token.DownloadCount),
			zap.Int("max", token.MaxDownloads),
		)
		return fmt.Errorf("download limit exceeded")
	}

	// Verify email matches
	if token.CustomerEmail != email {
		logger.Warn("Email mismatch for token",
			zap.String("token", token.Token[:16]+"..."),
			zap.String("provided_email", email),
			zap.String("expected_email", token.CustomerEmail),
		)
		return fmt.Errorf("invalid email for this token")
	}

	return nil
}

// ============================================================================
// GET TOKENS BY EMAIL (Customer lookup)
// ============================================================================

func (s *DownloadTokenService) GetTokensByEmail(ctx context.Context, email string) ([]*domain.DownloadToken, error) {
	tokens, err := s.tokenRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Filter out expired/revoked
	var validTokens []*domain.DownloadToken
	for _, token := range tokens {
		if token.IsValid() {
			validTokens = append(validTokens, token)
		}
	}

	return validTokens, nil
}

// ============================================================================
// ADMIN ACTIONS
// ============================================================================

func (s *DownloadTokenService) RevokeToken(ctx context.Context, tokenID int64, adminID int64, reason string) error {
	return s.tokenRepo.Revoke(ctx, tokenID, adminID, reason)
}

func (s *DownloadTokenService) GetTokensByOrderID(ctx context.Context, orderID int64) ([]*domain.DownloadToken, error) {
	return s.tokenRepo.GetByOrderID(ctx, orderID)
}

func (s *DownloadTokenService) GetDownloadHistory(ctx context.Context, orderID int64) ([]*domain.Download, error) {
	return s.downloadRepo.GetByOrderID(ctx, orderID)
}

// ============================================================================
// CLEANUP EXPIRED TOKENS
// ============================================================================

func (s *DownloadTokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	return s.tokenRepo.CleanupExpired(ctx)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *DownloadTokenService) generateSecureToken() string {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based if crypto/rand fails
		logger.Error("Failed to generate secure random token", zap.Error(err))
		return s.generateFallbackToken()
	}

	// Hash to create 64-character hex string
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func (s *DownloadTokenService) generateFallbackToken() string {
	// Less secure fallback
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%d-%d", timestamp, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func getFileExtension(format string) string {
	switch format {
	case "XLSX":
		return "xlsx"
	case "PDF":
		return "pdf"
	case "PPTX":
		return "pptx"
	case "DOCX":
		return "docx"
	default:
		return "zip"
	}
}
