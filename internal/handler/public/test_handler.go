package public

import (
	"github.com/gofiber/fiber/v2"
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

func (h *TestHandler) GetQuestions(c *fiber.Ctx) error {
	questions, err := h.testService.GetAllQuestions(c.Context(), true)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, fiber.Map{
		"questions":       questions,
		"total_questions": len(questions),
	})
}

type SubmitTestRequest struct {
	Email     string                 `json:"email" validate:"required,email"`
	Name      string                 `json:"name" validate:"required"`
	Responses map[string]interface{} `json:"responses" validate:"required"`
}

func (h *TestHandler) Submit(c *fiber.Ctx) error {
	var req SubmitTestRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	submitReq := &service.SubmitTestRequest{
		Email:     req.Email,
		Name:      req.Name,
		Responses: req.Responses,
	}

	result, err := h.testService.SubmitTest(c.Context(), submitReq)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Test completed successfully", result)
}

func (h *TestHandler) GetResults(c *fiber.Ctx) error {
	testNumber := c.Params("testNumber")
	email := c.Query("email")

	if email == "" {
		return response.Error(c, fiber.NewError(400, "Email required for verification"))
	}

	submission, err := h.testService.GetSubmissionByTestNumber(c.Context(), testNumber, email)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, submission)
}