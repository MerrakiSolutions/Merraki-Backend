package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/merraki/merraki-backend/internal/config"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
)

type CurrencyService struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewCurrencyService(cfg *config.Config) *CurrencyService {
	return &CurrencyService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type ExchangeRateResponse struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

func (s *CurrencyService) GetExchangeRate(ctx context.Context, from, to string) (float64, error) {
	// For now, return hardcoded rates
	// TODO: Implement actual API call to exchange rate service
	rates := map[string]map[string]float64{
		"INR": {
			"USD": 0.012,
			"EUR": 0.011,
			"GBP": 0.0095,
			"AUD": 0.018,
			"CAD": 0.016,
			"SGD": 0.016,
			"AED": 0.044,
		},
	}

	if fromRates, ok := rates[from]; ok {
		if rate, ok := fromRates[to]; ok {
			return rate, nil
		}
	}

	return 1.0, nil
}

func (s *CurrencyService) GetExchangeRateFromAPI(ctx context.Context, from, to string) (float64, error) {
	url := fmt.Sprintf("%s/%s", s.cfg.External.ExchangeRateAPIURL, from)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, apperrors.Wrap(err, "CURRENCY_ERROR", "Failed to create request", 500)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, apperrors.Wrap(err, "CURRENCY_ERROR", "Failed to fetch exchange rate", 500)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, apperrors.New("CURRENCY_ERROR", "Failed to fetch exchange rate", 500)
	}

	var result ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, apperrors.Wrap(err, "CURRENCY_ERROR", "Failed to parse response", 500)
	}

	if rate, ok := result.Rates[to]; ok {
		return rate, nil
	}

	return 0, apperrors.New("CURRENCY_ERROR", "Currency not found", 404)
}