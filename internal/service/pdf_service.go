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
	return &PDFService{
		storageService: storageService,
	}
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
	pdf.Cell(100, 7, "Item")
	pdf.Cell(30, 7, "Quantity")
	pdf.Cell(30, 7, "Price")
	pdf.Cell(30, 7, "Total")
	pdf.Ln(7)

	// Items
	pdf.SetFont("Arial", "", 10)
	for _, item := range items {
		pdf.Cell(100, 6, item.TemplateName)
		pdf.Cell(30, 6, fmt.Sprintf("%d", item.Quantity))
		pdf.Cell(30, 6, fmt.Sprintf("%.2f", item.UnitPrice))
		pdf.Cell(30, 6, fmt.Sprintf("%.2f", item.Subtotal))
		pdf.Ln(6)
	}

	pdf.Ln(5)

	// Totals
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(160, 6, "Subtotal:")
	pdf.Cell(30, 6, fmt.Sprintf("%s %.2f", order.Subtotal))
	pdf.Ln(6)

	if order.TaxAmount > 0 {
		pdf.Cell(160, 6, "Tax:")
		pdf.Cell(30, 6, fmt.Sprintf("%s %.2f",order.TaxAmount))
		pdf.Ln(6)
	}

	if order.DiscountAmount > 0 {
		pdf.Cell(160, 6, "Discount:")
		pdf.Cell(30, 6, fmt.Sprintf("-%s %.2f", order.DiscountAmount))
		pdf.Ln(6)
	}

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(160, 8, "Total:")
	pdf.Cell(30, 8, fmt.Sprintf("%s %.2f", order.TotalAmount))

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
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