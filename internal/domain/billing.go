package domain

// BillingAddress represents customer billing information
type BillingAddress struct {
	Name         string `json:"name" validate:"required"`
	Email        string `json:"email" validate:"required,email"`
	Phone        string `json:"phone"`
	AddressLine1 string `json:"address_line1" validate:"required"`
	AddressLine2 string `json:"address_line2"`
	City         string `json:"city" validate:"required"`
	State        string `json:"state" validate:"required"`
	Country      string `json:"country" validate:"required,len=2"`
	PostalCode   string `json:"postal_code" validate:"required"`
}