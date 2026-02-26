package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/pkg/validator"
	"github.com/merraki/merraki-backend/internal/service"
)

type CalculatorHandler struct {
	calculatorService *service.CalculatorService
}

func NewCalculatorHandler(calculatorService *service.CalculatorService) *CalculatorHandler {
	return &CalculatorHandler{calculatorService: calculatorService}
}

type ValuationRequest struct {
	RevenueYear1 float64 `json:"revenue_year_1" validate:"required,min=0"`
	RevenueYear2 float64 `json:"revenue_year_2" validate:"required,min=0"`
	RevenueYear3 float64 `json:"revenue_year_3" validate:"required,min=0"`
	RevenueYear4 float64 `json:"revenue_year_4" validate:"required,min=0"`
	RevenueYear5 float64 `json:"revenue_year_5" validate:"required,min=0"`
	ExitMultiple float64 `json:"exit_multiple" validate:"required,min=0"`
	DiscountRate float64 `json:"discount_rate" validate:"required,min=0,max=100"`
	Industry     string  `json:"industry" validate:"required"`
}

func (h *CalculatorHandler) CalculateValuation(c *fiber.Ctx) error {
	var req ValuationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	input := service.ValuationInput{
		RevenueYear1: req.RevenueYear1,
		RevenueYear2: req.RevenueYear2,
		RevenueYear3: req.RevenueYear3,
		RevenueYear4: req.RevenueYear4,
		RevenueYear5: req.RevenueYear5,
		ExitMultiple: req.ExitMultiple,
		DiscountRate: req.DiscountRate,
		Industry:     req.Industry,
	}

	output, err := h.calculatorService.CalculateValuation(c.Context(), input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, output)
}

type BreakevenRequest struct {
	FixedCosts          float64 `json:"fixed_costs" validate:"required,min=0"`
	VariableCostPerUnit float64 `json:"variable_cost_per_unit" validate:"required,min=0"`
	PricePerUnit        float64 `json:"price_per_unit" validate:"required,min=0"`
	MonthsToForecast    int     `json:"months_to_forecast" validate:"required,min=1,max=60"`
}

func (h *CalculatorHandler) CalculateBreakeven(c *fiber.Ctx) error {
	var req BreakevenRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	if err := validator.Validate(req); err != nil {
		return response.ValidationError(c, validator.FormatValidationErrors(err))
	}

	input := service.BreakevenInput{
		FixedCosts:          req.FixedCosts,
		VariableCostPerUnit: req.VariableCostPerUnit,
		PricePerUnit:        req.PricePerUnit,
		MonthsToForecast:    req.MonthsToForecast,
	}

	output, err := h.calculatorService.CalculateBreakeven(c.Context(), input)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, output)
}

// Add these methods to the existing calculator_handler.go

func (h *CalculatorHandler) SaveResult(c *fiber.Ctx) error {
	var req struct {
		CalculatorType string                 `json:"calculator_type" validate:"required"`
		Email          string                 `json:"email"`
		SavedName      string                 `json:"saved_name"`
		Inputs         map[string]interface{} `json:"inputs" validate:"required"`
		Results        map[string]interface{} `json:"results" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, err)
	}

	result := &domain.CalculatorResult{
		CalculatorType: req.CalculatorType,
		Email:          &req.Email,
		SavedName:      &req.SavedName,
		Inputs:         req.Inputs,
		Results:        req.Results,
	}

	if err := h.calculatorService.SaveResult(c.Context(), result); err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, "Calculation saved successfully", result)
}

func (h *CalculatorHandler) GetResults(c *fiber.Ctx) error {
	email := c.Query("email")
	calcType := c.Query("calculator_type")

	if email == "" {
		return response.Error(c, fiber.NewError(400, "Email required"))
	}

	results, err := h.calculatorService.GetResultsByEmail(c.Context(), email, calcType)
	if err != nil {
		return response.Error(c, err)
	}

	return response.SuccessData(c, results)
}

func (h *CalculatorHandler) ExportPDF(c *fiber.Ctx) error {
	// TODO: Implement PDF export
	return response.Error(c, fiber.NewError(501, "PDF export not yet implemented"))
}