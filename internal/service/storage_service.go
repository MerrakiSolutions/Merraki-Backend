package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/merraki/merraki-backend/internal/config"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
)

type StorageService struct {
	cloudinary *cloudinary.Cloudinary
	cfg        *config.Config
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	cld, err := cloudinary.NewFromParams(
		cfg.Storage.CloudinaryName,
		cfg.Storage.CloudinaryKey,
		cfg.Storage.CloudinarySecret,
	)
	if err != nil {
		return nil, err
	}

	return &StorageService{
		cloudinary: cld,
		cfg:        cfg,
	}, nil
}

type UploadResult struct {
	URL          string
	PublicID     string
	Format       string
	ResourceType string
	Bytes        int
}

func (s *StorageService) UploadFile(ctx context.Context, file *multipart.FileHeader, folder string) (*UploadResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, apperrors.Wrap(err, "STORAGE_ERROR", "Failed to open file", 500)
	}
	defer src.Close()

	uploadParams := uploader.UploadParams{
		Folder:       fmt.Sprintf("%s/%s", s.cfg.Storage.CloudinaryFolder, folder),
		ResourceType: "auto",
	}

	result, err := s.cloudinary.Upload.Upload(ctx, src, uploadParams)
	if err != nil {
		return nil, apperrors.Wrap(err, "STORAGE_ERROR", "Failed to upload file", 500)
	}

	return &UploadResult{
		URL:          result.SecureURL,
		PublicID:     result.PublicID,
		Format:       result.Format,
		ResourceType: result.ResourceType,
		Bytes:        result.Bytes,
	}, nil
}

func (s *StorageService) UploadFromReader(ctx context.Context, reader io.Reader, filename, folder string) (*UploadResult, error) {
	uploadParams := uploader.UploadParams{
		Folder:       fmt.Sprintf("%s/%s", s.cfg.Storage.CloudinaryFolder, folder),
		ResourceType: "auto",
		PublicID:     filename,
	}

	result, err := s.cloudinary.Upload.Upload(ctx, reader, uploadParams)
	if err != nil {
		return nil, apperrors.Wrap(err, "STORAGE_ERROR", "Failed to upload file", 500)
	}

	return &UploadResult{
		URL:          result.SecureURL,
		PublicID:     result.PublicID,
		Format:       result.Format,
		ResourceType: result.ResourceType,
		Bytes:        result.Bytes,
	}, nil
}

func (s *StorageService) DeleteFile(ctx context.Context, publicID string) error {
	_, err := s.cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return apperrors.Wrap(err, "STORAGE_ERROR", "Failed to delete file", 500)
	}
	return nil
}

// ============================================================================
// GENERATE SIGNED URL - Cloudinary Private CDN with Signature
// ============================================================================

func (s *StorageService) GenerateSignedURL(ctx context.Context, publicID string, duration time.Duration) (string, error) {
	// Calculate expiration timestamp
	expirationTime := time.Now().Add(duration).Unix()
	
	// Build the string to sign
	// Format: {public_id}{timestamp}{api_secret}
	stringToSign := fmt.Sprintf("%s%d%s", publicID, expirationTime, s.cfg.Storage.CloudinarySecret)
	
	// Generate SHA-256 signature
	hash := sha256.New()
	hash.Write([]byte(stringToSign))
	signature := hex.EncodeToString(hash.Sum(nil))[:16] // Use first 16 characters
	
	// Build authenticated URL
	// Cloudinary authenticated URL format:
	// https://res.cloudinary.com/{cloud_name}/image/authenticated/s--{signature}--/v1/{public_id}.{format}
	baseURL := fmt.Sprintf("https://res.cloudinary.com/%s", s.cfg.Storage.CloudinaryName)
	
	// Extract resource type and format from publicID
	// publicID format: folder/filename or folder/subfolder/filename
	resourceType := "raw" // Default for files
	
	// Build the signed URL
	signedURL := fmt.Sprintf(
		"%s/%s/authenticated/s--%s--/%s?exp=%d",
		baseURL,
		resourceType,
		signature,
		publicID,
		expirationTime,
	)
	
	return signedURL, nil
}

// ============================================================================
// ALTERNATIVE: Generate Simple Signed URL (More Compatible)
// ============================================================================

func (s *StorageService) GenerateSignedDownloadURL(ctx context.Context, publicID string, duration time.Duration) (string, error) {
	// For raw file downloads, use type: "upload" with attachment flag
	expirationTime := time.Now().Add(duration).Unix()
	
	// Cloudinary URL structure for raw files:
	// https://res.cloudinary.com/{cloud_name}/raw/upload/{public_id}
	
	// Generate signature for private download
	// Parameters to sign: public_id + timestamp
	paramsToSign := fmt.Sprintf("public_id=%s&timestamp=%d%s", 
		publicID, 
		expirationTime,
		s.cfg.Storage.CloudinarySecret,
	)
	
	hash := sha256.New()
	hash.Write([]byte(paramsToSign))
	signature := hex.EncodeToString(hash.Sum(nil))
	
	// Build download URL with signature
	downloadURL := fmt.Sprintf(
		"https://res.cloudinary.com/%s/raw/upload/fl_attachment/%s?timestamp=%d&signature=%s",
		s.cfg.Storage.CloudinaryName,
		publicID,
		expirationTime,
		signature,
	)
	
	return downloadURL, nil
}

// ============================================================================
// PUBLIC URL (No Authentication) - Use only for public resources
// ============================================================================

func (s *StorageService) GetPublicURL(publicID string, resourceType string) string {
	if resourceType == "" {
		resourceType = "raw"
	}
	
	return fmt.Sprintf(
		"https://res.cloudinary.com/%s/%s/upload/%s",
		s.cfg.Storage.CloudinaryName,
		resourceType,
		publicID,
	)
}