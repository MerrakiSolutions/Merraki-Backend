package service

import (
	"context"
	"fmt"
	"math"

	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
	"github.com/valyala/fasthttp"
)

type CalculatorService struct {
	calculatorRepo *postgres.CalculatorRepository
}

func (s *CalculatorService) GetAll(ctx *fasthttp.RequestCtx, filters map[string]interface{}, limit int, i int) (any, any, any) {
	panic("unimplemented")
}

func NewCalculatorService(calculatorRepo *postgres.CalculatorRepository) *CalculatorService {
	return &CalculatorService{
		calculatorRepo: calculatorRepo,
	}
}

// Valuation Calculator
type ValuationInput struct {
	RevenueYear1 float64
	RevenueYear2 float64
	RevenueYear3 float64
	RevenueYear4 float64
	RevenueYear5 float64
	ExitMultiple float64
	DiscountRate float64
	Industry     string
}

type ValuationOutput struct {
	ExitValuation   float64
	PresentValue    float64
	YearlyBreakdown []YearValuation
	ChartData       ChartData
	Metrics         ValuationMetrics
}

type YearValuation struct {
	Year         int
	Revenue      float64
	Valuation    float64
	PVFactor     float64
	PresentValue float64
}

type ValuationMetrics struct {
	CAGR           float64
	AvgGrowthRate  float64
	Recommendation string
}

type ChartData struct {
	Labels    []string
	Revenue   []float64
	Valuation []float64
}

func (s *CalculatorService) CalculateValuation(ctx context.Context, input ValuationInput) (*ValuationOutput, error) {
	revenues := []float64{input.RevenueYear1, input.RevenueYear2, input.RevenueYear3, input.RevenueYear4, input.RevenueYear5}

	// Calculate exit valuation
	exitValuation := revenues[4] * input.ExitMultiple

	// Calculate yearly breakdown
	var breakdown []YearValuation
	var totalPV float64

	for i, revenue := range revenues {
		year := i + 1
		valuation := revenue * input.ExitMultiple
		pvFactor := 1.0 / math.Pow(1+input.DiscountRate/100, float64(year))
		pv := valuation * pvFactor
		totalPV += pv

		breakdown = append(breakdown, YearValuation{
			Year:         year,
			Revenue:      revenue,
			Valuation:    valuation,
			PVFactor:     pvFactor,
			PresentValue: pv,
		})
	}

	// Calculate CAGR
	cagr := (math.Pow(revenues[4]/revenues[0], 1.0/4.0) - 1) * 100

	// Calculate average growth rate
	var totalGrowth float64
	for i := 1; i < len(revenues); i++ {
		growth := ((revenues[i] - revenues[i-1]) / revenues[i-1]) * 100
		totalGrowth += growth
	}
	avgGrowthRate := totalGrowth / 4

	// Determine recommendation
	recommendation := "Strong growth trajectory"
	if cagr < 20 {
		recommendation = "Moderate growth - consider optimization strategies"
	} else if cagr > 100 {
		recommendation = "Exceptional growth - ensure sustainable scaling"
	}

	// Prepare chart data
	chartData := ChartData{
		Labels:    []string{"Year 1", "Year 2", "Year 3", "Year 4", "Year 5"},
		Revenue:   revenues,
		Valuation: make([]float64, 5),
	}
	for i := range revenues {
		chartData.Valuation[i] = revenues[i] * input.ExitMultiple
	}

	return &ValuationOutput{
		ExitValuation:   exitValuation,
		PresentValue:    totalPV,
		YearlyBreakdown: breakdown,
		ChartData:       chartData,
		Metrics: ValuationMetrics{
			CAGR:           cagr,
			AvgGrowthRate:  avgGrowthRate,
			Recommendation: recommendation,
		},
	}, nil
}

// Breakeven Calculator
type BreakevenInput struct {
	FixedCosts          float64
	VariableCostPerUnit float64
	PricePerUnit        float64
	MonthsToForecast    int
}

type BreakevenOutput struct {
	BreakevenUnits            int
	BreakevenRevenue          float64
	BreakevenMonth            int
	ContributionMargin        float64
	ContributionMarginPercent float64
	MonthlyForecast           []MonthForecast
	ChartData                 BreakevenChartData
}

type MonthForecast struct {
	Month            int
	UnitsSold        int
	Revenue          float64
	VariableCosts    float64
	FixedCosts       float64
	TotalCosts       float64
	Profit           float64
	CumulativeProfit float64
	IsBreakeven      bool
}

type BreakevenChartData struct {
	Labels  []string
	Revenue []float64
	Costs   []float64
	Profit  []float64
}

func (s *CalculatorService) CalculateBreakeven(ctx context.Context, input BreakevenInput) (*BreakevenOutput, error) {
	// Calculate contribution margin
	contributionMargin := input.PricePerUnit - input.VariableCostPerUnit
	contributionMarginPercent := (contributionMargin / input.PricePerUnit) * 100

	// Calculate breakeven units
	breakevenUnits := int(math.Ceil(input.FixedCosts / contributionMargin))
	breakevenRevenue := float64(breakevenUnits) * input.PricePerUnit

	// Monthly forecast (assuming linear growth)
	var forecast []MonthForecast
	var cumulativeProfit float64
	breakevenMonth := 0
	unitsPerMonth := breakevenUnits / 12 // Distribute evenly

	for month := 1; month <= input.MonthsToForecast; month++ {
		unitsSold := unitsPerMonth * month
		revenue := float64(unitsSold) * input.PricePerUnit
		variableCosts := float64(unitsSold) * input.VariableCostPerUnit
		totalCosts := input.FixedCosts + variableCosts
		profit := revenue - totalCosts
		cumulativeProfit += profit

		isBreakeven := cumulativeProfit >= 0 && breakevenMonth == 0
		if isBreakeven {
			breakevenMonth = month
		}

		forecast = append(forecast, MonthForecast{
			Month:            month,
			UnitsSold:        unitsSold,
			Revenue:          revenue,
			VariableCosts:    variableCosts,
			FixedCosts:       input.FixedCosts,
			TotalCosts:       totalCosts,
			Profit:           profit,
			CumulativeProfit: cumulativeProfit,
			IsBreakeven:      isBreakeven,
		})
	}

	// Prepare chart data
	chartData := BreakevenChartData{
		Labels:  make([]string, input.MonthsToForecast),
		Revenue: make([]float64, input.MonthsToForecast),
		Costs:   make([]float64, input.MonthsToForecast),
		Profit:  make([]float64, input.MonthsToForecast),
	}

	for i, f := range forecast {
		chartData.Labels[i] = fmt.Sprintf("Month %d", f.Month)
		chartData.Revenue[i] = f.Revenue
		chartData.Costs[i] = f.TotalCosts
		chartData.Profit[i] = f.CumulativeProfit
	}

	return &BreakevenOutput{
		BreakevenUnits:            breakevenUnits,
		BreakevenRevenue:          breakevenRevenue,
		BreakevenMonth:            breakevenMonth,
		ContributionMargin:        contributionMargin,
		ContributionMarginPercent: contributionMarginPercent,
		MonthlyForecast:           forecast,
		ChartData:                 chartData,
	}, nil
}

// Save Results
func (s *CalculatorService) SaveResult(ctx context.Context, result *domain.CalculatorResult) error {
	if err := s.calculatorRepo.Create(ctx, result); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to save result", 500)
	}
	return nil
}

func (s *CalculatorService) GetResultsByEmail(ctx context.Context, email, calculatorType string) ([]*domain.CalculatorResult, error) {
	return s.calculatorRepo.GetByEmail(ctx, email, calculatorType)
}

func (s *CalculatorService) GetAnalytics(ctx context.Context) (map[string]interface{}, error) {
	return s.calculatorRepo.GetAnalytics(ctx)
}
