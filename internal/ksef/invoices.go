package ksef

import (
	"errors"
	"time"
)

// support for the most basic types of invoices
// won't include everything from the schema here

// currencies and country codes will be validated just by checking their length and whether they're upper case letters

type InvoiceReceived struct {
	InvoiceType InvoiceType `json:"invoice_type"`
	InvoiceNumber string `json:"invoice_number"`
	IssueDate string `json:"issue_date"`
	Currency *string `json:"currency"`
	ExchangeRate *float64 `json:"exchange_rate"`
	TotalAmount float64 `json:"total_amount"`
	CorrectionReason *string `json:"correction_reason"`
	OriginalInvoiceNumber *string `json:"original_invoice_number"`
	OriginalKsefId *string `json:"original_ksef_id"`
	Seller Entity `json:"seller"`
	Buyer Entity `json:"buyer"`
	CorrectedBuyer *Entity `json:"corrected_buyer"`
	Items []LineItem `json:"items"`
	TaxBreakdowns []TaxBreakdown `json:"tax_breakdowns"` 
	Payment *Payment `json:"payment"`
	Flags *Flags `json:"flags"`
}

type Entity struct {
	Nip string `json:"nip"`
	Name *string `json:"name"`
	CountryCode *string `json:"country_code"`
	AddressLine1 *string `json:"address_line_1"`
	AddressLine2 *string `json:"address_line_2"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type LineItem struct {
	LineNumber int `json:"line_number"`
	Name string `json:"name"`
	Unit *string `json:"unit"`
	Quantity *float64 `json:"quantity"`
	UnitPriceNet *float64 `json:"unit_price_net"`
	NetAmount float64 `json:"net_amount"`
	TaxRate TaxRate `json:"tax_rate"`
}

type TaxBreakdown struct {
	TaxRate TaxRate `json:"tax_rate"`
	NetAmount float64 `json:"net_amount"`
	TaxAmount *float64 `json:"tax_amount"`
	TaxAmountPln *float64 `json:"tax_amount_pln"`
}

type Payment struct {
	DueDate *string `json:"due_date"`
	MethodCode *string `json:"method_code"`
	BankAccount *string `json:"bank_account"`
	BankName *string `json:"bank_name"`
}

type Flags struct {
	SplitPayment *bool `json:"split_payment"`
	CashMethod *bool `json:"cash_method"`
	ReverseCharge *bool `json:"reverse_charge"`
}

func (i *InvoiceReceived) ValidateInvoiceReceived() error {
	switch i.InvoiceType {
	case InvoiceTypeVAT, InvoiceTypeKOR, InvoiceTypeZAL, InvoiceTypeROZ, InvoiceTypeUPR, InvoiceTypeKORROZ, InvoiceTypeKORZAL:
	default:
		return errors.New("Invalid invoice type.")
	}
	_, err := time.Parse("2006-01-02", i.IssueDate)
	if err != nil {
		return errors.New("Invalid date format (issue date).")
	}
	if i.InvoiceType == InvoiceTypeKOR || i.InvoiceType == InvoiceTypeKORROZ || i.InvoiceType == InvoiceTypeKORZAL {
		if i.OriginalInvoiceNumber == nil || i.CorrectionReason == nil || i.OriginalKsefId == nil {
			return errors.New("Missing details (correction data).")
		}
	} else {
		if i.OriginalInvoiceNumber != nil || i.CorrectionReason != nil || i.OriginalKsefId != nil {
			return errors.New("Incorrect details (correction data on non-corrective invoice).")
		}
	}
	if i.Currency == nil || len(*i.Currency) != 3 {
		return errors.New("Incorrect currency data.")
	}
	if *i.Currency != "PLN" && i.ExchangeRate == nil {
		return errors.New("Missing exchange rate for the currency.")
	}
	if len(i.Items) == 0 {
		return errors.New("Empty invoice.")
	}
	for _, item := range i.Items {
		if i.InvoiceType != InvoiceTypeKOR && i.InvoiceType != InvoiceTypeKORZAL && i.InvoiceType != InvoiceTypeKORROZ {
			if item.NetAmount < 0 {
				return errors.New("Net amount can't be negative.")
			}
		}
		if item.Quantity != nil && item.UnitPriceNet == nil {
			return errors.New("Unit price cannot be empty.")
		}
		switch item.TaxRate {
		case TaxRate23, TaxRate22, TaxRate8, TaxRate7, TaxRate5, TaxRate4, TaxRate3, TaxRate0WDT, TaxRate0EX, TaxRate0KR, TaxRateZW, TaxRateOO, TaxRateNPI, TaxRateNPII:
		default:
			return errors.New("Invalid tax rate (line items.")
		}
	}
	if len(i.Buyer.Nip) != 10 || len(i.Seller.Nip) != 10 {
		return errors.New("Incorrect NIP values.")
	}
	if i.Seller.Name == nil || i.Seller.AddressLine1 == nil {
		return errors.New("Missing seller details.")
	}
	if i.InvoiceType != InvoiceTypeUPR {
		if i.Buyer.Name == nil || i.Buyer.AddressLine1 == nil {
			return errors.New("Missing buyer details on a non-UPR invoice.")
		}
	}
	if len(i.TaxBreakdowns) == 0 {
		return errors.New("Missing tax breakdowns.")
	}
	for _, taxbr := range i.TaxBreakdowns {
		switch taxbr.TaxRate {
		case TaxRate23, TaxRate22, TaxRate8, TaxRate7, TaxRate5, TaxRate4, TaxRate3, TaxRate0WDT, TaxRate0EX, TaxRate0KR, TaxRateZW, TaxRateOO, TaxRateNPI, TaxRateNPII:
		default:
			return errors.New("Invalid tax rate (tax breakdown).")
		}
	}
	return nil
}

func TransformToXML(invoice InvoiceReceived) ([]byte, error) {
	// TODO
	return nil, nil
}

// will have to create an Invoice object to insert it into sqlite

