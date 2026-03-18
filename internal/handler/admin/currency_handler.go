package admin

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

// RefreshRates forces a refresh of exchange rates (admin only)
func (h *CurrencyHandler) RefreshRates(c *fiber.Ctx) error {
	if err := h.service.RefreshRates(c.Context()); err != nil {
		return response.Error(c, err)
	}

	rates := h.service.GetAllRates()
	lastUpdate := h.service.GetLastUpdate()

	return response.Success(c, "Exchange rates refreshed successfully", fiber.Map{
		"rates":       rates,
		"last_update": lastUpdate,
	})
}

// GetExchangeRates returns all current exchange rates (admin view)
func (h *CurrencyHandler) GetExchangeRates(c *fiber.Ctx) error {
	rates := h.service.GetAllRates()
	lastUpdate := h.service.GetLastUpdate()

	return response.SuccessData(c, fiber.Map{
		"base_currency": "USD",
		"rates":         rates,
		"last_update":   lastUpdate,
		"supported":     h.service.GetSupportedCurrencies(),
	})
}