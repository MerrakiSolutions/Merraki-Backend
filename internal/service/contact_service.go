package service

import (
	"context"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type ContactService struct {
	contactRepo *postgres.ContactRepository
	logRepo     *postgres.ActivityLogRepository
	emailSvc    *EmailService
}

func NewContactService(
	contactRepo *postgres.ContactRepository,
	logRepo *postgres.ActivityLogRepository,
	emailSvc *EmailService,
) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		logRepo:     logRepo,
		emailSvc:    emailSvc,
	}
}

type CreateContactRequest struct {
	Name      string
	Email     string
	Phone     string
	Subject   string
	Message   string
	IPAddress string
}

func (s *ContactService) CreateContact(ctx context.Context, req *CreateContactRequest) (*domain.Contact, error) {
	contact := &domain.Contact{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     &req.Phone,
		Subject:   req.Subject,
		Message:   req.Message,
		Status:    "new",
		IPAddress: &req.IPAddress,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create contact", 500)
	}

	// TODO: Send notification to admin

	return contact, nil
}

func (s *ContactService) GetContactByID(ctx context.Context, id int64) (*domain.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find contact", 500)
	}

	if contact == nil {
		return nil, apperrors.ErrNotFound
	}

	return contact, nil
}

func (s *ContactService) GetAllContacts(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Contact, int, error) {
	return s.contactRepo.GetAll(ctx, filters, limit, offset)
}

func (s *ContactService) UpdateContact(ctx context.Context, contact *domain.Contact, updatedBy int64) error {
	existing, err := s.contactRepo.FindByID(ctx, contact.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find contact", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update contact", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_contact",
		EntityType: strPtr("contact"),
		EntityID:   &contact.ID,
		Details:    domain.JSONMap{"status": contact.Status},
	})

	return nil
}

func (s *ContactService) ReplyToContact(ctx context.Context, contactID int64, replyMessage string, repliedBy int64) error {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find contact", 500)
	}

	if contact == nil {
		return apperrors.ErrNotFound
	}

	// Update contact
	now := timePtr(time.Now())
	contact.Status = "replied"
	contact.RepliedBy = &repliedBy
	contact.RepliedAt = now
	contact.ReplyNotes = &replyMessage

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update contact", 500)
	}

	// Send reply email
	if err := s.emailSvc.SendContactReply(ctx, contact.Email, contact.Name, replyMessage); err != nil {
		return apperrors.Wrap(err, "EMAIL_ERROR", "Failed to send reply email", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &repliedBy,
		Action:     "reply_contact",
		EntityType: strPtr("contact"),
		EntityID:   &contactID,
		Details:    domain.JSONMap{"email": contact.Email},
	})

	return nil
}

func (s *ContactService) DeleteContact(ctx context.Context, id, deletedBy int64) error {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find contact", 500)
	}

	if contact == nil {
		return apperrors.ErrNotFound
	}

	if err := s.contactRepo.Delete(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete contact", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_contact",
		EntityType: strPtr("contact"),
		EntityID:   &id,
		Details:    domain.JSONMap{"email": contact.Email},
	})

	return nil
}

func (s *ContactService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return s.contactRepo.GetAnalytics(ctx)
}