package ksef

type InvoiceReceived struct {
	InvoiceType           InvoiceType    `json:"invoice_type"`
	InvoiceNumber         string         `json:"invoice_number"`
	IssueDate             string         `json:"issue_date"`
	OriginalIssueDate     *string        `json:"original_issue_date"`
	Currency              *string        `json:"currency"`
	ExchangeRate          *float64       `json:"exchange_rate"`
	TotalAmount           float64        `json:"total_amount"`
	CorrectionReason      *string        `json:"correction_reason"`
	OriginalInvoiceNumber *string        `json:"original_invoice_number"`
	OriginalKsefId        *string        `json:"original_ksef_id"`
	Seller                Entity         `json:"seller"`
	Buyer                 Entity         `json:"buyer"`
	CorrectedBuyer        *Entity        `json:"corrected_buyer"`
	Items                 []LineItem     `json:"items"`
	TaxBreakdowns         []TaxBreakdown `json:"tax_breakdowns"`
	Payment               *Payment       `json:"payment"`
	Flags                 *Flags         `json:"flags"`
	CallbackURL           *string        `json:"callback_url"`
}

type Entity struct {
	Nip          string  `json:"nip"`
	Name         *string `json:"name"`
	CountryCode  *string `json:"country_code"`
	AddressLine1 *string `json:"address_line_1"`
	AddressLine2 *string `json:"address_line_2"`
	Email        *string `json:"email"`
	Phone        *string `json:"phone"`
}

type LineItem struct {
	LineNumber   int      `json:"line_number"`
	Name         string   `json:"name"`
	Unit         *string  `json:"unit"`
	Quantity     *float64 `json:"quantity"`
	UnitPriceNet *float64 `json:"unit_price_net"`
	NetAmount    float64  `json:"net_amount"`
	TaxRate      TaxRate  `json:"tax_rate"`
}

type TaxBreakdown struct {
	TaxRate      TaxRate  `json:"tax_rate"`
	NetAmount    float64  `json:"net_amount"`
	TaxAmount    *float64 `json:"tax_amount"`
	TaxAmountPln *float64 `json:"tax_amount_pln"`
}

type Payment struct {
	DueDate     *string `json:"due_date"`
	MethodCode  *string `json:"method_code"`
	BankAccount *string `json:"bank_account"`
	BankName    *string `json:"bank_name"`
}

type Flags struct {
	SplitPayment  *bool `json:"split_payment"`
	CashMethod    *bool `json:"cash_method"`
	ReverseCharge *bool `json:"reverse_charge"`
}
