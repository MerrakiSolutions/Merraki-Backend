package admin

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/service"
)

type CalculatorHandler struct {
	calculatorService *service.CalculatorService
}

func NewCalculatorHandler(calculatorService *service.CalculatorService) *CalculatorHandler {
	return &CalculatorHandler{calculatorService: calculatorService}
}

func (h *CalculatorHandler) GetAll(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	params := &domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}
	params.Validate()

	filters := make(map[string]interface{})
	if calcType := c.Query("calculator_type"); calcType != "" {
		filters["calculator_type"] = calcType
	}
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	results, total, err := h.calculatorService.GetAll(c.Context(), filters, params.Limit, params.GetOffset())
	if err != nil {
		return response.Error(c, err.(error))
	}

	return response.Paginated(c, results, total.(int), params.Page, params.Limit)
}

func (h *CalculatorHandler) GetAnalytics(c *fiber.Ctx) error {
	analytics, err := h.calculatorService.GetAnalytics(c.Context())
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, analytics)
}
