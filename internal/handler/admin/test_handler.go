package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/middleware"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type TestHandler struct {
	testService *service.TestService
}

func NewTestHandler(testService *service.TestService) *TestHandler {
	return &TestHandler{testService: testService}
}

// Questions Management
func (h *TestHandler) GetAllQuestions(c *fiber.Ctx) error {
	questions, err := h.testService.GetAllQuestions(c.Context(), false)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, questions)
}

func (h *TestHandler) GetQuestionByID(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid question ID"))
	}

	question, err := h.testService.GetQuestionByID(c.Context(), int64(id))
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, question)
}

type CreateQuestionRequest struct {
	QuestionText string                 `json:"question_text" validate:"required"`
	QuestionType string                 `json:"question_type" validate:"required,oneof=multiple_choice slider scenario"`
	Options      map[string]interface{} `json:"options" validate:"required"`
	Category     string                 `json:"category" validate:"required"`
	Weight       int                    `json:"weight" validate:"min=1"`
	OrderNumber  int                    `json:"order_number" validate:"required,min=1"`
	IsActive     bool                   `json:"is_active"`
}

func (h *TestHandler) CreateQuestion(c *fiber.Ctx) error {
	var req CreateQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	question := &domain.TestQuestion{
		QuestionText: req.QuestionText,
		QuestionType: req.QuestionType,
		Options:      req.Options,
		Category:     req.Category,
		Weight:       req.Weight,
		OrderNumber:  req.OrderNumber,
		IsActive:     req.IsActive,
	}

	if err := h.testService.CreateQuestion(c.Context(), question, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Question created successfully", question)
}

func (h *TestHandler) UpdateQuestion(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid question ID"))
	}

	var req CreateQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	adminID := middleware.GetAdminID(c)

	question := &domain.TestQuestion{
		ID:           int64(id),
		QuestionText: req.QuestionText,
		QuestionType: req.QuestionType,
		Options:      req.Options,
		Category:     req.Category,
		Weight:       req.Weight,
		OrderNumber:  req.OrderNumber,
		IsActive:     req.IsActive,
	}

	if err := h.testService.UpdateQuestion(c.Context(), question, adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Question updated successfully", question)
}

func (h *TestHandler) DeleteQuestion(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid question ID"))
	}

	adminID := middleware.GetAdminID(c)

	if err := h.testService.DeleteQuestion(c.Context(), int64(id), adminID); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Question deleted successfully", nil)
}

// Submissions Management
func (h *TestHandler) GetAllSubmissions(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})
	if personalityType := c.Query("personality_type"); personalityType != "" {
		filters["personality_type"] = personalityType
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	submissions, total, err := h.testService.GetAllSubmissions(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err)
	}

	return response.Paginated(c, submissions, total, params.Page, params.Limit)
}

func (h *TestHandler) GetAnalytics(c *fiber.Ctx) error {
	analytics, err := h.testService.GetAnalytics(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}

// Add this method to existing test_handler.go

func (h *TestHandler) Export(c *fiber.Ctx) error {
	// TODO: Export test submissions to CSV
	return response.Error(c, fiber.NewError(501, "Export not yet implemented"))
}