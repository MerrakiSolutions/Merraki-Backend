package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/response"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"github.com/merraki/merraki-backend/internal/repository/redis"
)

type UtilityHandler struct {
	db    *postgres.Database
	redis *redis.Client
}

func NewUtilityHandler(db *postgres.Database, redis *redis.Client) *UtilityHandler {
	return &UtilityHandler{
		db:    db,
		redis: redis,
	}
}

func (h *UtilityHandler) Health(c *fiber.Ctx) error {
	status := "healthy"
	services := fiber.Map{}

	// Check database
	if err := h.db.Health(); err != nil {
		status = "unhealthy"
		services["database"] = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	// Check redis
	if err := h.redis.Health(); err != nil {
		status = "unhealthy"
		services["redis"] = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	services["storage"] = "healthy" // Placeholder

	statusCode := fiber.StatusOK
	if status == "unhealthy" {
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"success":  status == "healthy",
		"status":   status,
		"services": services,
	})
}

type CurrencyConvertRequest struct {
	Amount int    `query:"amount" validate:"required,min=1"`
	To     string `query:"to" validate:"required"`
}

func (h *UtilityHandler) CurrencyConvert(c *fiber.Ctx) error {
	// TODO: Implement actual currency conversion
	// For now return mock data
	amountINR := c.QueryInt("amount", 0)
	toCurrency := c.Query("to", "USD")

	rates := map[string]float64{
		"USD": 0.012,
		"EUR": 0.011,
		"GBP": 0.0095,
	}

	rate := rates[toCurrency]
	if rate == 0 {
		rate = 1.0
	}

	amountLocal := (float64(amountINR) / 100.0) * rate

	return response.SuccessData(c, fiber.Map{
		"amount_inr":    amountINR,
		"amount_local":  amountLocal,
		"currency_code": toCurrency,
		"currency_symbol": getCurrencySymbol(toCurrency),
		"exchange_rate": rate,
	})
}

func getCurrencySymbol(code string) string {
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"INR": "₹",
	}
	if symbol, ok := symbols[code]; ok {
		return symbol
	}
	return code
}