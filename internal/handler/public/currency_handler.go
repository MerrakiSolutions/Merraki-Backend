package public

import (
	"github.com/gofiber/fiber/v2"
	"github.com/merraki/merraki-backend/internal/pkg/currency"
	"github.com/merraki/merraki-backend/internal/pkg/response"
)

type CurrencyHandler struct {
	service *currency.ExchangeRateService
}

func NewCurrencyHandler() *CurrencyHandler {
	return &CurrencyHandler{
		service: currency.GetService(),
	}
}

// GetExchangeRates returns all current exchange rates
func (h *CurrencyHandler) GetExchangeRates(c *fiber.Ctx) error {
	rates := h.service.GetAllRates()
	lastUpdate := h.service.GetLastUpdate()

	return response.SuccessData(c, fiber.Map{
		"base_currency": "USD",
		"rates":         rates,
		"last_update":   lastUpdate,
	})
}

// GetSupportedCurrencies returns list of supported currencies
func (h *CurrencyHandler) GetSupportedCurrencies(c *fiber.Ctx) error {
	currencies := h.service.GetSupportedCurrencies()

	return response.SuccessData(c, fiber.Map{
		"currencies": currencies,
	})
}

// ConvertCurrency converts amount from one currency to another
func (h *CurrencyHandler) ConvertCurrency(c *fiber.Ctx) error {
	var req struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
		From   string  `json:"from" validate:"required"`
		To     string  `json:"to" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.NewError(400, "Invalid request body"))
	}

	result, err := h.service.Convert(req.Amount, req.From, req.To)
	if err != nil {
		return response.Error(c, fiber.NewError(400, err.Error()))
	}

	fromRate, _ := h.service.GetRate(req.From)
	toRate, _ := h.service.GetRate(req.To)

	return response.SuccessData(c, fiber.Map{
		"amount":        req.Amount,
		"from":          req.From,
		"to":            req.To,
		"result":        result,
		"exchange_rate": toRate / fromRate,
	})
}

// RefreshRates forces a refresh of exchange rates (admin only)
func (h *CurrencyHandler) RefreshRates(c *fiber.Ctx) error {
	if err := h.service.RefreshRates(c.Context()); err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, "Exchange rates refreshed successfully", nil)
}