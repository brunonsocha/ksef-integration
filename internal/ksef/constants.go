package ksef

import "errors"

type InvoiceType string

const (
	InvoiceTypeVAT    InvoiceType = "VAT"
	InvoiceTypeKOR    InvoiceType = "KOR"
	InvoiceTypeZAL    InvoiceType = "ZAL"
	InvoiceTypeROZ    InvoiceType = "ROZ"
	InvoiceTypeUPR    InvoiceType = "UPR"
	InvoiceTypeKORZAL InvoiceType = "KOR_ZAL"
	InvoiceTypeKORROZ InvoiceType = "KOR_ROZ"
)

var INVALID_SESSION_ERR = errors.New("Invalid session reference.")
var INVOICE_REJECTED_ERR = errors.New("KSeF rejected the invoice.")

type TaxRate string

const (
	TaxRate23   TaxRate = "23"
	TaxRate22   TaxRate = "22"
	TaxRate8    TaxRate = "8"
	TaxRate7    TaxRate = "7"
	TaxRate5    TaxRate = "5"
	TaxRate4    TaxRate = "4"
	TaxRate3    TaxRate = "3"
	TaxRate0KR  TaxRate = "0 KR"
	TaxRate0WDT TaxRate = "0 WDT"
	TaxRate0EX  TaxRate = "0 EX"
	TaxRateZW   TaxRate = "zw"
	TaxRateOO   TaxRate = "oo"
	TaxRateNPI  TaxRate = "np I"
	TaxRateNPII TaxRate = "np II"
)
