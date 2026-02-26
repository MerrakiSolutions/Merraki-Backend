package service

import (
	"context"

	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type NewsletterService struct {
	newsletterRepo *postgres.NewsletterRepository
	emailSvc       *EmailService
}

func NewNewsletterService(
	newsletterRepo *postgres.NewsletterRepository,
	emailSvc *EmailService,
) *NewsletterService {
	return &NewsletterService{
		newsletterRepo: newsletterRepo,
		emailSvc:       emailSvc,
	}
}

type SubscribeRequest struct {
	Email     string
	Name      string
	Source    string
	IPAddress string
}

func (s *NewsletterService) Subscribe(ctx context.Context, req *SubscribeRequest) error {
	// Check if already subscribed
	existing, err := s.newsletterRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to check subscription", 500)
	}

	if existing != nil {
		if existing.Status == "active" {
			return apperrors.New("ALREADY_SUBSCRIBED", "Email already subscribed", 200)
		}
		// Reactivate if previously unsubscribed
		// TODO: Add reactivate method
		return apperrors.New("ALREADY_SUBSCRIBED", "Email already subscribed", 200)
	}

	// Create new subscription
	subscriber := &domain.NewsletterSubscriber{
		Email:     req.Email,
		Name:      &req.Name,
		Status:    "active",
		Source:    req.Source,
		IPAddress: &req.IPAddress,
	}

	if err := s.newsletterRepo.Create(ctx, subscriber); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create subscription", 500)
	}

	// Send confirmation email
	_ = s.emailSvc.SendNewsletterConfirmation(ctx, req.Email, req.Name)

	return nil
}

func (s *NewsletterService) Unsubscribe(ctx context.Context, email string) error {
	subscriber, err := s.newsletterRepo.FindByEmail(ctx, email)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find subscriber", 500)
	}

	if subscriber == nil {
		return apperrors.ErrNotFound
	}

	if err := s.newsletterRepo.Unsubscribe(ctx, email); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to unsubscribe", 500)
	}

	return nil
}

func (s *NewsletterService) GetAllSubscribers(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.NewsletterSubscriber, int, error) {
	return s.newsletterRepo.GetAll(ctx, filters, limit, offset)
}

func (s *NewsletterService) DeleteSubscriber(ctx context.Context, id int64) error {
	return s.newsletterRepo.Delete(ctx, id)
}

func (s *NewsletterService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return s.newsletterRepo.GetAnalytics(ctx)
}

