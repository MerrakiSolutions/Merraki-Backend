package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
)

type PDFService struct {
	storageService *StorageService
}

func NewPDFService(storageService *StorageService) *PDFService {
	return &PDFService{
		storageService: storageService,
	}
}

func (s *PDFService) GenerateTestReport(ctx context.Context, submission *domain.TestSubmission) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(190, 10, "Financial Personality Test Results", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Test Info
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(190, 8, fmt.Sprintf("Test Number: %s", submission.TestNumber), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Name: %s", *submission.Name), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Email: %s", submission.Email), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, fmt.Sprintf("Date: %s", submission.CreatedAt.Format("January 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(10)

	// Personality Type
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Your Personality Type", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(190, 8, *submission.PersonalityType, "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Risk Appetite
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Risk Appetite", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(190, 8, fmt.Sprintf("%s", *submission.RiskAppetite), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Scores
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Category Scores", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)

	if submission.Scores != nil {
		scores := submission.Scores
		if planning, ok := scores["planning"].(float64); ok {
			pdf.CellFormat(190, 8, fmt.Sprintf("Planning: %.1f/100", planning), "", 1, "L", false, 0, "")
		}
		if risk, ok := scores["risk"].(float64); ok {
			pdf.CellFormat(190, 8, fmt.Sprintf("Risk Management: %.1f/100", risk), "", 1, "L", false, 0, "")
		}
		if behavior, ok := scores["behavior"].(float64); ok {
			pdf.CellFormat(190, 8, fmt.Sprintf("Financial Behavior: %.1f/100", behavior), "", 1, "L", false, 0, "")
		}
		if knowledge, ok := scores["knowledge"].(float64); ok {
			pdf.CellFormat(190, 8, fmt.Sprintf("Financial Knowledge: %.1f/100", knowledge), "", 1, "L", false, 0, "")
		}
		if overall, ok := scores["overall"].(float64); ok {
			pdf.Ln(5)
			pdf.SetFont("Arial", "B", 14)
			pdf.CellFormat(190, 8, fmt.Sprintf("Overall Score: %.1f/100", overall), "", 1, "L", false, 0, "")
		}
	}

	// Recommendations
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Recommendations", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(190, 8, "Based on your results, we recommend focusing on:\n\n1. Long-term financial planning\n2. Risk diversification strategies\n3. Regular financial reviews\n4. Continuous learning about investment options", "", "", false)

	// Generate PDF
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return "", apperrors.Wrap(err, "PDF_ERROR", "Failed to generate PDF", 500)
	}

	// Upload to storage
	filename := fmt.Sprintf("test-report-%s-%d.pdf", submission.TestNumber, time.Now().Unix())
	result, err := s.storageService.UploadFromReader(ctx, &buf, filename, "test-reports")
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

func (s *PDFService) GenerateCalculatorReport(ctx context.Context, calculatorType string, inputs, results map[string]interface{}) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	title := "Valuation Calculator Report"
	if calculatorType == "breakeven" {
		title = "Breakeven Calculator Report"
	}
	pdf.CellFormat(190, 10, title, "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Date
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(190, 8, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(10)

	// Inputs
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Input Parameters", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)

	for key, value := range inputs {
		pdf.CellFormat(190, 8, fmt.Sprintf("%s: %v", key, value), "", 1, "L", false, 0, "")
	}
	pdf.Ln(10)

	// Results
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(190, 10, "Results", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 12)

	for key, value := range results {
		pdf.CellFormat(190, 8, fmt.Sprintf("%s: %v", key, value), "", 1, "L", false, 0, "")
	}

	// Generate PDF
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return "", apperrors.Wrap(err, "PDF_ERROR", "Failed to generate PDF", 500)
	}

	// Upload to storage
	filename := fmt.Sprintf("%s-report-%d.pdf", calculatorType, time.Now().Unix())
	result, err := s.storageService.UploadFromReader(ctx, &buf, filename, "calculator-reports")
	if err != nil {
		return "", err
	}

	return result.URL, nil
}