package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/merraki/merraki-backend/internal/domain"
)

type PDFService struct {
	storageService *StorageService
}

func NewPDFService(storageService *StorageService) *PDFService {
	return &PDFService{storageService: storageService}
}

// ============================================================================
// GENERATE ORDER INVOICE
// ============================================================================

func (s *PDFService) GenerateOrderInvoice(ctx context.Context, order *domain.Order, items []*domain.OrderItem) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	pdf.Cell(0, 10, "INVOICE")
	pdf.Ln(15)

	// Company Info
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "Merraki Solutions")
	pdf.Ln(5)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, "www.merrakisolutions.com")
	pdf.Ln(5)
	pdf.Cell(0, 5, "info@merrakisolutions.com")
	pdf.Ln(10)

	// Order Info
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, fmt.Sprintf("Order #%s", order.OrderNumber))
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, fmt.Sprintf("Date: %s", order.CreatedAt.Format("January 2, 2006")))
	pdf.Ln(10)

	// Customer Info
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 6, "Bill To:")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, order.CustomerName)
	pdf.Ln(5)
	pdf.Cell(0, 5, order.CustomerEmail)
	pdf.Ln(10)

	// Items Table Header
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(130, 7, "Item")
	pdf.Cell(30, 7, "Price (USD)")
	pdf.Ln(7)

	// Items — qty removed, price is per-item flat
	pdf.SetFont("Arial", "", 10)
	for _, item := range items {
		pdf.Cell(130, 6, item.TemplateName)
		pdf.Cell(30, 6, fmt.Sprintf("$%.2f", domain.CentsToUSD(item.PriceUSDCents)))
		pdf.Ln(6)
	}
	pdf.Ln(5)

	// Totals
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(160, 6, "Subtotal:")
	pdf.Cell(30, 6, fmt.Sprintf("$%.2f", domain.CentsToUSD(order.SubtotalUSDCents)))
	pdf.Ln(6)

	if order.TaxAmountUSDCents > 0 {
		pdf.Cell(160, 6, "Tax:")
		pdf.Cell(30, 6, fmt.Sprintf("$%.2f", domain.CentsToUSD(order.TaxAmountUSDCents)))
		pdf.Ln(6)
	}

	if order.DiscountAmountUSDCents > 0 {
		pdf.Cell(160, 6, "Discount:")
		pdf.Cell(30, 6, fmt.Sprintf("-$%.2f", domain.CentsToUSD(order.DiscountAmountUSDCents)))
		pdf.Ln(6)
	}

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(160, 8, "Total (USD):")
	pdf.Cell(30, 8, fmt.Sprintf("$%.2f", domain.CentsToUSD(order.TotalAmountUSDCents)))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ============================================================================
// UPLOAD PDF TO STORAGE
// ============================================================================

func (s *PDFService) UploadInvoice(ctx context.Context, pdfBytes []byte, orderNumber string) (*UploadResult, error) {
	filename := fmt.Sprintf("invoice_%s.pdf", orderNumber)
	reader := bytes.NewReader(pdfBytes)
	return s.storageService.UploadFromReader(ctx, reader, filename, "invoices")
}