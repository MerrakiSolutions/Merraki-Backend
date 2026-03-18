package currency

import (
	"github.com/merraki/merraki-backend/internal/domain"
)

// SyncTemplatePrice syncs prices across all currencies for a template
func SyncTemplatePrice(template *domain.Template) error {
	service := GetService()

	// If both prices are 0, nothing to sync
	if template.PriceINR == 0 && template.PriceUSD == 0 {
		return nil
	}

	// Determine which price to use as source
	var err error
	if template.PriceINR > 0 && template.PriceUSD == 0 {
		// Calculate USD from INR
		template.PriceUSD, err = service.Convert(template.PriceINR, "INR", "USD")
		if err != nil {
			return err
		}
	} else if template.PriceUSD > 0 && template.PriceINR == 0 {
		// Calculate INR from USD
		template.PriceINR, err = service.Convert(template.PriceUSD, "USD", "INR")
		if err != nil {
			return err
		}
	}
	// If both are provided, keep them as is (admin might have set custom prices)

	// Round prices
	template.PriceINR = Round(template.PriceINR)
	template.PriceUSD = Round(template.PriceUSD)

	// Sync sale prices if applicable
	if template.IsOnSale {
		if template.SalePriceINR != nil && template.SalePriceUSD == nil {
			usd, err := service.Convert(*template.SalePriceINR, "INR", "USD")
			if err != nil {
				return err
			}
			usd = Round(usd)
			template.SalePriceUSD = &usd
		} else if template.SalePriceUSD != nil && template.SalePriceINR == nil {
			inr, err := service.Convert(*template.SalePriceUSD, "USD", "INR")
			if err != nil {
				return err
			}
			inr = Round(inr)
			template.SalePriceINR = &inr
		}

		// Round sale prices
		if template.SalePriceINR != nil {
			rounded := Round(*template.SalePriceINR)
			template.SalePriceINR = &rounded
		}
		if template.SalePriceUSD != nil {
			rounded := Round(*template.SalePriceUSD)
			template.SalePriceUSD = &rounded
		}
	}

	return nil
}

// ConvertTemplatePrice returns template price in requested currency
func ConvertTemplatePrice(template *domain.Template, targetCurrency string) (float64, error) {
	service := GetService()

	// Get base price in USD (our base currency)
	basePrice := template.PriceUSD
	if template.IsOnSale && template.SalePriceUSD != nil {
		basePrice = *template.SalePriceUSD
	}

	// Convert to target currency
	return service.Convert(basePrice, "USD", targetCurrency)
}