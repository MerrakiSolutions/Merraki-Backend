package service

import (
	"context"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type TestService struct {
	testRepo *postgres.TestRepository
	logRepo  *postgres.ActivityLogRepository
	emailSvc *EmailService
}

func NewTestService(
	testRepo *postgres.TestRepository,
	logRepo *postgres.ActivityLogRepository,
	emailSvc *EmailService,
) *TestService {
	return &TestService{
		testRepo: testRepo,
		logRepo:  logRepo,
		emailSvc: emailSvc,
	}
}

// Questions Management
func (s *TestService) CreateQuestion(ctx context.Context, question *domain.TestQuestion, createdBy int64) error {
	if err := s.testRepo.CreateQuestion(ctx, question); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create question", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &createdBy,
		Action:     "create_test_question",
		EntityType: strPtr("test_question"),
		EntityID:   &question.ID,
		Details:    domain.JSONMap{"category": question.Category},
	})

	return nil
}

func (s *TestService) GetAllQuestions(ctx context.Context, activeOnly bool) ([]*domain.TestQuestion, error) {
	return s.testRepo.GetAllQuestions(ctx, activeOnly)
}

func (s *TestService) GetQuestionByID(ctx context.Context, id int64) (*domain.TestQuestion, error) {
	question, err := s.testRepo.GetQuestionByID(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find question", 500)
	}

	if question == nil {
		return nil, apperrors.ErrNotFound
	}

	return question, nil
}

func (s *TestService) UpdateQuestion(ctx context.Context, question *domain.TestQuestion, updatedBy int64) error {
	existing, err := s.testRepo.GetQuestionByID(ctx, question.ID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find question", 500)
	}

	if existing == nil {
		return apperrors.ErrNotFound
	}

	if err := s.testRepo.UpdateQuestion(ctx, question); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update question", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &updatedBy,
		Action:     "update_test_question",
		EntityType: strPtr("test_question"),
		EntityID:   &question.ID,
		Details:    domain.JSONMap{"category": question.Category},
	})

	return nil
}

func (s *TestService) DeleteQuestion(ctx context.Context, id, deletedBy int64) error {
	question, err := s.testRepo.GetQuestionByID(ctx, id)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find question", 500)
	}

	if question == nil {
		return apperrors.ErrNotFound
	}

	if err := s.testRepo.DeleteQuestion(ctx, id); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to delete question", 500)
	}

	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &deletedBy,
		Action:     "delete_test_question",
		EntityType: strPtr("test_question"),
		EntityID:   &id,
	})

	return nil
}

// Test Submissions
type SubmitTestRequest struct {
	Email     string
	Name      string
	Responses domain.JSONMap
}

type SubmitTestResponse struct {
	TestNumber      string
	PersonalityType string
	RiskAppetite    string
	Scores          domain.JSONMap
	ReportURL       string
}

func (s *TestService) SubmitTest(ctx context.Context, req *SubmitTestRequest) (*SubmitTestResponse, error) {
	// Generate test number
	testNumber := fmt.Sprintf("TEST%d", time.Now().Unix())

	// Calculate scores
	scores := s.calculateScores(req.Responses)

	// Determine personality type
	personalityType := s.determinePersonalityType(scores)

	// Determine risk appetite
	riskAppetite := s.determineRiskAppetite(scores)

	// Generate report URL (placeholder)
	reportURL := fmt.Sprintf("https://storage.example.com/reports/%s.pdf", testNumber)

	// Create submission
	submission := &domain.TestSubmission{
		TestNumber:      testNumber,
		Email:           req.Email,
		Name:            &req.Name,
		Responses:       req.Responses,
		PersonalityType: &personalityType,
		RiskAppetite:    &riskAppetite,
		Scores:          scores,
		ReportURL:       &reportURL,
		ReportSent:      false,
	}

	if err := s.testRepo.CreateSubmission(ctx, submission); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create submission", 500)
	}

	// Send results email
	_ = s.emailSvc.SendTestResults(ctx, req.Email, req.Name, testNumber, reportURL)

	return &SubmitTestResponse{
		TestNumber:      testNumber,
		PersonalityType: personalityType,
		RiskAppetite:    riskAppetite,
		Scores:          scores,
		ReportURL:       reportURL,
	}, nil
}

func (s *TestService) GetSubmissionByTestNumber(ctx context.Context, testNumber, email string) (*domain.TestSubmission, error) {
	submission, err := s.testRepo.FindSubmissionByTestNumber(ctx, testNumber)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find submission", 500)
	}

	if submission == nil {
		return nil, apperrors.ErrNotFound
	}

	// Verify email
	if submission.Email != email {
		return nil, apperrors.ErrUnauthorized
	}

	return submission, nil
}

func (s *TestService) GetAllSubmissions(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.TestSubmission, int, error) {
	return s.testRepo.GetAllSubmissions(ctx, filters, limit, offset)
}

func (s *TestService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return s.testRepo.GetAnalytics(ctx)
}

// Helper methods
func (s *TestService) calculateScores(responses domain.JSONMap) domain.JSONMap {
	// TODO: Implement actual scoring algorithm
	scores := domain.JSONMap{
		"planning":  75.0,
		"risk":      65.0,
		"behavior":  80.0,
		"knowledge": 70.0,
		"overall":   72.5,
	}
	return scores
}

func (s *TestService) determinePersonalityType(scores domain.JSONMap) string {
	// TODO: Implement actual personality type determination
	types := []string{
		"Strategic Planner",
		"Risk Taker",
		"Conservative Investor",
		"Growth Seeker",
		"Balanced Investor",
	}
	return types[0]
}

func (s *TestService) determineRiskAppetite(scores domain.JSONMap) string {
	// TODO: Implement actual risk appetite calculation
	riskScore := scores["risk"].(float64)
	
	if riskScore < 40 {
		return "low"
	} else if riskScore < 70 {
		return "moderate"
	}
	return "high"
}