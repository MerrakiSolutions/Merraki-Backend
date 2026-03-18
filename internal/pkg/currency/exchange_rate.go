package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// ExchangeRateAPIResponse represents the API response
type ExchangeRateAPIResponse struct {
	Result             string             `json:"result"`
	Provider           string             `json:"provider"`
	Documentation      string             `json:"documentation"`
	TermsOfUse         string             `json:"terms_of_use"`
	TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
	TimeLastUpdateUTC  string             `json:"time_last_update_utc"`
	TimeNextUpdateUnix int64              `json:"time_next_update_unix"`
	TimeNextUpdateUTC  string             `json:"time_next_update_utc"`
	TimeEolUnix        int64              `json:"time_eol_unix"`
	BaseCode           string             `json:"base_code"`
	Rates              map[string]float64 `json:"rates"`
}

// ExchangeRateService handles currency conversion
type ExchangeRateService struct {
	apiURL     string
	apiKey     string
	baseCode   string // e.g., "USD"
	rates      map[string]float64
	lastUpdate time.Time
	cacheTTL   time.Duration
	mu         sync.RWMutex
	httpClient *http.Client
}

var globalService *ExchangeRateService
var serviceOnce sync.Once

// InitExchangeRateService initializes the global exchange rate service
func InitExchangeRateService() *ExchangeRateService {
	serviceOnce.Do(func() {
		apiURL := os.Getenv("EXCHANGE_RATE_API_URL")
		if apiURL == "" {
			apiURL = "https://api.exchangerate-api.com/v4/latest"
		}

		apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")

		globalService = &ExchangeRateService{
			apiURL:   apiURL,
			apiKey:   apiKey,
			baseCode: "USD", // Base currency
			rates:    make(map[string]float64),
			cacheTTL: 24 * time.Hour, // Cache for 24 hours
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}

		// Fetch rates on initialization
		if err := globalService.FetchRates(context.Background()); err != nil {
			// Log error but don't fail - use fallback rates
			fmt.Printf("Warning: Failed to fetch exchange rates: %v\n", err)
			globalService.setFallbackRates()
		}
	})

	return globalService
}

// GetService returns the global exchange rate service
func GetService() *ExchangeRateService {
	if globalService == nil {
		return InitExchangeRateService()
	}
	return globalService
}

// FetchRates fetches the latest exchange rates from the API
func (s *ExchangeRateService) FetchRates(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if cache is still valid
	if time.Since(s.lastUpdate) < s.cacheTTL && len(s.rates) > 0 {
		return nil
	}

	url := fmt.Sprintf("%s/%s", s.apiURL, s.baseCode)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key if provided
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp ExchangeRateAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Result == "error" {
		return fmt.Errorf("API error response")
	}

	s.rates = apiResp.Rates
	s.lastUpdate = time.Now()

	return nil
}

// setFallbackRates sets default fallback rates if API fails
func (s *ExchangeRateService) setFallbackRates() {
	s.rates = map[string]float64{
		"USD": 1.0,
		"INR": 83.0,
		"EUR": 0.92,
		"GBP": 0.79,
		"AUD": 1.52,
		"CAD": 1.36,
		"SGD": 1.35,
		"AED": 3.67,
	}
	s.lastUpdate = time.Now()
}

// GetRate returns the exchange rate for a given currency
func (s *ExchangeRateService) GetRate(currency string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rate, ok := s.rates[currency]
	if !ok {
		return 0, fmt.Errorf("unsupported currency: %s", currency)
	}

	return rate, nil
}

// Convert converts an amount from one currency to another
func (s *ExchangeRateService) Convert(amount float64, from, to string) (float64, error) {
	if from == to {
		return amount, nil
	}

	// Ensure rates are fresh
	if time.Since(s.lastUpdate) > s.cacheTTL {
		if err := s.FetchRates(context.Background()); err != nil {
			// Continue with cached rates on error
			fmt.Printf("Warning: Using cached rates due to error: %v\n", err)
		}
	}

	fromRate, err := s.GetRate(from)
	if err != nil {
		return 0, err
	}

	toRate, err := s.GetRate(to)
	if err != nil {
		return 0, err
	}

	// Convert to base currency (USD) first, then to target currency
	amountInBase := amount / fromRate
	result := amountInBase * toRate

	return Round(result), nil
}

// GetAllRates returns all available exchange rates
func (s *ExchangeRateService) GetAllRates() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	ratesCopy := make(map[string]float64, len(s.rates))
	for k, v := range s.rates {
		ratesCopy[k] = v
	}

	return ratesCopy
}

// GetSupportedCurrencies returns a list of supported currency codes
func (s *ExchangeRateService) GetSupportedCurrencies() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	currencies := make([]string, 0, len(s.rates))
	for currency := range s.rates {
		currencies = append(currencies, currency)
	}

	return currencies
}

// GetLastUpdate returns the last time rates were updated
func (s *ExchangeRateService) GetLastUpdate() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// RefreshRates forces a refresh of exchange rates
func (s *ExchangeRateService) RefreshRates(ctx context.Context) error {
	s.mu.Lock()
	s.lastUpdate = time.Time{} // Force refresh
	s.mu.Unlock()

	return s.FetchRates(ctx)
}

// Round rounds a float to 2 decimal places
func Round(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

// Helper functions for common conversions

func INRtoUSD(inr float64) (float64, error) {
	return GetService().Convert(inr, "INR", "USD")
}

func USDtoINR(usd float64) (float64, error) {
	return GetService().Convert(usd, "USD", "INR")
}

func EURtoUSD(eur float64) (float64, error) {
	return GetService().Convert(eur, "EUR", "USD")
}

func USDtoEUR(usd float64) (float64, error) {
	return GetService().Convert(usd, "USD", "EUR")
}