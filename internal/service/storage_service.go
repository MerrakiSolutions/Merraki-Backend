package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

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