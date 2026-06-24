package service

import (
	"context"
	"fmt"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"go.uber.org/zap"
)

// ============================================================================
// FOUNDER LEAD SERVICE
// ============================================================================

type FounderLeadService struct {
	repo         *postgres.FounderLeadRepository
	emailService *EmailService
}

func NewFounderLeadService(
	repo *postgres.FounderLeadRepository,
	emailService *EmailService,
) *FounderLeadService {
	return &FounderLeadService{
		repo:         repo,
		emailService: emailService,
	}
}

// ============================================================================
// SUBMIT - Public endpoint
// ============================================================================

func (s *FounderLeadService) SubmitLead(ctx context.Context, req *domain.SubmitFounderLeadRequest, ip string) (*domain.FounderLead, error) {
	// Validate
	if req.Name == "" || req.Email == "" {
		return nil, fmt.Errorf("name and email are required")
	}

	if req.TotalMax == 0 {
		req.TotalMax = 100
	}

	// Create lead
	lead, err := s.repo.Create(ctx, req, ip)
	if err != nil {
		logger.Error("Failed to create founder lead", zap.Error(err))
		return nil, fmt.Errorf("failed to save lead: %w", err)
	}

	logger.Info("Founder lead created",
		zap.Int64("lead_id", lead.ID),
		zap.String("email", lead.Email),
		zap.String("personality_type", lead.PersonalityType),
	)

	// Send notification email to founder
	go func() {
		if err := s.emailService.SendFounderLeadConfirmation(ctx, lead); err != nil {
			logger.Error("Failed to send founder lead confirmation",
				zap.Error(err),
				zap.Int64("lead_id", lead.ID),
			)
		}
	}()

	// Send notification email to admin
	go func() {
		if err := s.emailService.SendFounderLeadNotification(ctx, lead); err != nil {
			logger.Error("Failed to send founder lead notification to admin",
				zap.Error(err),
				zap.Int64("lead_id", lead.ID),
			)
		}
	}()

	return lead, nil
}

// ============================================================================
// ADMIN OPERATIONS
// ============================================================================

func (s *FounderLeadService) GetStats(ctx context.Context) (*domain.FounderLeadStats, error) {
	return s.repo.GetStats(ctx)
}

func (s *FounderLeadService) ListLeads(ctx context.Context, f postgres.ListFounderLeadsFilter) (*postgres.ListFounderLeadsResult, error) {
	return s.repo.List(ctx, f)
}

func (s *FounderLeadService) GetLead(ctx context.Context, id int64) (*domain.FounderLead, error) {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("lead not found")
	}
	return lead, nil
}

func (s *FounderLeadService) UpdateLeadStatus(ctx context.Context, id int64, status domain.LeadStatus) (*domain.FounderLead, error) {
	lead, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		logger.Error("Failed to update lead status",
			zap.Error(err),
			zap.Int64("lead_id", id),
		)
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	logger.Info("Lead status updated",
		zap.Int64("lead_id", id),
		zap.String("status", string(status)),
	)

	return lead, nil
}

func (s *FounderLeadService) SendFollowUp(ctx context.Context, id int64, subject, message string) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("lead not found")
	}

	// Send email
	if err := s.emailService.SendFounderFollowUp(ctx, lead.Email, lead.Name, subject, message); err != nil {
		logger.Error("Failed to send follow-up email",
			zap.Error(err),
			zap.Int64("lead_id", id),
		)
		return err
	}

	// Auto-mark as contacted if still new
	if lead.Status == domain.LeadStatusNew {
		_, _ = s.repo.UpdateStatus(ctx, id, domain.LeadStatusContacted)
	}

	logger.Info("Follow-up email sent",
		zap.Int64("lead_id", id),
		zap.String("to", lead.Email),
	)

	return nil
}

func (s *FounderLeadService) DeleteLead(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		logger.Error("Failed to delete lead",
			zap.Error(err),
			zap.Int64("lead_id", id),
		)
		return err
	}

	logger.Info("Lead deleted",
		zap.Int64("lead_id", id),
	)

	return nil
}
