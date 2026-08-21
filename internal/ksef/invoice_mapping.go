package ksef

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// support for the most basic types of invoices
// won't include everything from the schema here

// currencies and country codes will be validated just by checking their length and whether they're upper case letters

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
		if i.OriginalIssueDate == nil {
			return errors.New("Missing original issue date on a corrective invoice.")
		}
		_, err := time.Parse("2006-01-02", *i.OriginalIssueDate)
		if err != nil {
			return errors.New("Invalid date format (original issue date).")
		}
		if i.CorrectedBuyer != nil && (i.CorrectedBuyer.Name == nil || i.CorrectedBuyer.AddressLine1 == nil || !validateNIP(i.CorrectedBuyer.Nip)) {
			return errors.New("Missing data for corrected buyer/")
		}
		if i.CorrectedBuyer != nil && i.CorrectedBuyer.CountryCode == nil {
			cc := "PL"
			i.CorrectedBuyer.CountryCode = &cc
		}
	} else {
		if i.OriginalInvoiceNumber != nil || i.CorrectionReason != nil || i.OriginalKsefId != nil || i.OriginalIssueDate != nil || i.CorrectedBuyer != nil {
			return errors.New("Incorrect details (correction data on non-corrective invoice).")
		}
	}
	if i.Currency == nil || len(*i.Currency) != 3 {
		return errors.New("Incorrect currency data.")
	}
	if (*i.Currency != "PLN" && i.ExchangeRate == nil) || (*i.Currency == "PLN" && i.ExchangeRate != nil) {
		return errors.New("Missing data for the chosen currency (exchange rate was specified for PLN or unspecified for foreign).")
	}
	if i.ExchangeRate != nil && *i.ExchangeRate <= 0 {
		return errors.New("Exchange rate must be greater than zero.")
	}
	if i.Payment != nil {
		if i.Payment.DueDate == nil && i.Payment.MethodCode == nil && i.Payment.BankAccount == nil && i.Payment.BankName == nil {
			return errors.New("Payment details cannot be empty.")
		}
		if i.Payment.DueDate != nil {
			if _, err := time.Parse("2006-01-02", *i.Payment.DueDate); err != nil {
				return errors.New("Invalid payment due date format.")
			}
		}
		if i.Payment.MethodCode != nil {
			methodCode, err := strconv.Atoi(*i.Payment.MethodCode)
			if err != nil || methodCode < 1 || methodCode > 7 {
				return errors.New("Invalid payment method code.")
			}
		}
		if i.Payment.BankName != nil && i.Payment.BankAccount == nil {
			return errors.New("Bank account is required when bank name is provided.")
		}
		if i.Payment.BankAccount != nil && (len(*i.Payment.BankAccount) < 10 || len(*i.Payment.BankAccount) > 34) {
			return errors.New("Bank account number must contain between 10 and 34 characters.")
		}
	}
	if len(i.Items) == 0 {
		return errors.New("Empty invoice.")
	}
	lineNumbers := make(map[int]struct{})
	for _, item := range i.Items {
		if item.LineNumber <= 0 {
			return errors.New("Line number must be greater than zero.")
		}
		if _, exists := lineNumbers[item.LineNumber]; exists {
			return errors.New("Line numbers must be unique.")
		}
		lineNumbers[item.LineNumber] = struct{}{}
		if i.InvoiceType != InvoiceTypeKOR && i.InvoiceType != InvoiceTypeKORZAL && i.InvoiceType != InvoiceTypeKORROZ {
			if item.NetAmount < 0 {
				return errors.New("Net amount can't be negative.")
			}
		}
		if item.Quantity != nil && item.UnitPriceNet == nil {
			return errors.New("Unit price cannot be empty.")
		}
		switch item.TaxRate {
		case TaxRate23, TaxRate22, TaxRate8, TaxRate7, TaxRate5, TaxRate4, TaxRate0WDT, TaxRate0EX, TaxRate0KR, TaxRateZW, TaxRateOO, TaxRateNPI, TaxRateNPII:
		default:
			return errors.New("Invalid tax rate (line items.")
		}
	}
	if !validateNIP(i.Buyer.Nip) || !validateNIP(i.Seller.Nip) {
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
		case TaxRate23, TaxRate22, TaxRate8, TaxRate7, TaxRate5, TaxRate4, TaxRate0WDT, TaxRate0EX, TaxRate0KR, TaxRateZW, TaxRateOO, TaxRateNPI, TaxRateNPII:
		default:
			return errors.New("Invalid tax rate (tax breakdown).")
		}
		taxable := taxbr.TaxRate == TaxRate23 || taxbr.TaxRate == TaxRate22 || taxbr.TaxRate == TaxRate8 || taxbr.TaxRate == TaxRate7 || taxbr.TaxRate == TaxRate5 || taxbr.TaxRate == TaxRate4
		if *i.Currency != "PLN" && taxable && taxbr.TaxAmountPln == nil {
			return errors.New("Missing tax amount in PLN for foreign-currency invoice.")
		}
		if *i.Currency == "PLN" && taxbr.TaxAmountPln != nil {
			return errors.New("Tax amount in PLN must not be provided for a PLN invoice.")
		}
		if !taxable && taxbr.TaxAmountPln != nil {
			return errors.New("Tax amount in PLN must not be provided for a non-taxable breakdown.")
		}
	}
	defaultCountryCode := "PL"
	if i.Seller.CountryCode == nil {
		i.Seller.CountryCode = &defaultCountryCode
	}
	if i.Buyer.Name != nil && i.Buyer.CountryCode == nil {
		i.Buyer.CountryCode = &defaultCountryCode
	}
	totalNet := 0.0
	itemNetByTaxRate := make(map[TaxRate]float64)
	for _, j := range i.Items {
		totalNet += j.NetAmount
		itemNetByTaxRate[j.TaxRate] += j.NetAmount
	}
	totalNetTax := 0.0
	totalTax := 0.0
	breakdownNetByTaxRate := make(map[TaxRate]float64)
	for _, k := range i.TaxBreakdowns {
		requiresTax := true
		totalNetTax += k.NetAmount
		breakdownNetByTaxRate[k.TaxRate] += k.NetAmount
		switch k.TaxRate {
		case TaxRateZW, TaxRateOO, TaxRateNPII, TaxRateNPI, TaxRate0KR, TaxRate0EX, TaxRate0WDT:
			if k.TaxAmount != nil {
				return errors.New("Tax amount incorrect for the 0% tax rate.")
			}
			requiresTax = false
		}
		if k.TaxAmount != nil && requiresTax {
			totalTax += *k.TaxAmount
		} else if k.TaxAmount == nil && requiresTax {
			return errors.New("Tax amount missing for non-0 tax rate.")
		}
	}
	for taxRate, itemNet := range itemNetByTaxRate {
		breakdownNet, exists := breakdownNetByTaxRate[taxRate]
		if !exists {
			return fmt.Errorf("Missing tax breakdown for tax rate %s.", taxRate)
		}
		if itemNet != breakdownNet {
			return fmt.Errorf("Net amount for tax rate %s does not match its tax breakdown.", taxRate)
		}
	}
	for taxRate := range breakdownNetByTaxRate {
		if _, exists := itemNetByTaxRate[taxRate]; !exists {
			return fmt.Errorf("Tax breakdown for tax rate %s has no corresponding line items.", taxRate)
		}
	}
	if totalNet != totalNetTax || totalNet+totalTax != i.TotalAmount {
		return fmt.Errorf("Incorrect line items. Total netto in line items: %f, total netto in tax breakdowns: %f, total tax: %f, total amount: %f.", totalNet, totalNetTax, totalTax, i.TotalAmount)
	}
	return nil
}

func TransformToXML(inv *InvoiceReceived) ([]byte, error) {
	fa := Faktura{
		Xmlns: "http://crd.gov.pl/wzor/2025/06/25/13775/",
		Naglowek: Naglowek{
			KodFormularza: KodFormularza{
				KodSystemowy: "FA (3)",
				WersjaSchemy: "1-0E",
				Value:        "FA",
			},
			WariantFormularza: "3",
			DataWytworzeniaFa: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		},
		Podmiot1: Podmiot1{
			DaneIdentyfikacyjne: DaneIdentyfikacyjne{
				NIP:   inv.Seller.Nip,
				Nazwa: *inv.Seller.Name,
			},
			Adres: Adres{
				KodKraju: *inv.Seller.CountryCode,
				AdresL1:  *inv.Seller.AddressLine1,
			},
		},
		Fa: Fa{
			KodWaluty:     *inv.Currency,
			P_1:           inv.IssueDate,
			P_2:           inv.InvoiceNumber,
			P_6:           inv.IssueDate,
			P_15:          fmt.Sprintf("%.2f", inv.TotalAmount),
			RodzajFaktury: string(inv.InvoiceType),
			Adnotacje: Adnotacje{ // 2 is "nie dotyczy"
				P_16:  2,
				P_17:  2,
				P_18:  2,
				P_18A: 2,
				P_23:  2,
				Zwolnienie: ZwolnienieN{
					P_19N: 1,
				},
				NoweSrodkiTransportu: NoweSrodkiN{
					P_22N: 1,
				},
				PMarzy: PMarzyN{
					P_PMarzyN: 1,
				},
			},
		},
	}
	if inv.Seller.AddressLine2 != nil {
		fa.Podmiot1.Adres.AdresL2 = *inv.Seller.AddressLine2
	}
	if inv.Flags != nil {
		if inv.Flags.CashMethod != nil && *inv.Flags.CashMethod {
			fa.Fa.Adnotacje.P_16 = 1
		}
		if inv.Flags.ReverseCharge != nil && *inv.Flags.ReverseCharge {
			fa.Fa.Adnotacje.P_18 = 1
		}
		if inv.Flags.SplitPayment != nil && *inv.Flags.SplitPayment {
			fa.Fa.Adnotacje.P_18A = 1
		}
	}
	if inv.Payment != nil {
		fa.Fa.Platnosc = &Platnosc{}
		if inv.Payment.DueDate != nil {
			fa.Fa.Platnosc.TerminPlatnosci = &TerminPlatnosci{Termin: *inv.Payment.DueDate}
		}
		if inv.Payment.MethodCode != nil {
			fa.Fa.Platnosc.FormaPlatnosci = inv.Payment.MethodCode
		}
		if inv.Payment.BankAccount != nil {
			fa.Fa.Platnosc.RachunekBankowy = &RachunekBankowy{
				NrRB:       *inv.Payment.BankAccount,
				NazwaBanku: inv.Payment.BankName,
			}
		}
	}

	if (inv.InvoiceType == InvoiceTypeKOR || inv.InvoiceType == InvoiceTypeKORZAL || inv.InvoiceType == InvoiceTypeKORROZ) && inv.CorrectedBuyer != nil {
		fa.Podmiot2 = &Podmiot2{
			DaneIdentyfikacyjne: DaneIdentyfikacyjne{
				NIP:   inv.CorrectedBuyer.Nip,
				Nazwa: *inv.CorrectedBuyer.Name,
			},
			Adres: &Adres{
				KodKraju: *inv.CorrectedBuyer.CountryCode,
				AdresL1:  *inv.CorrectedBuyer.AddressLine1,
			},
			JST: 2,
			GV:  2,
		}
		if inv.CorrectedBuyer.AddressLine2 != nil {
			fa.Podmiot2.Adres.AdresL2 = *inv.CorrectedBuyer.AddressLine2
		}
	} else {
		if inv.Buyer.Name != nil && inv.Buyer.AddressLine1 != nil {
			fa.Podmiot2 = &Podmiot2{
				DaneIdentyfikacyjne: DaneIdentyfikacyjne{
					NIP:   inv.Buyer.Nip,
					Nazwa: *inv.Buyer.Name,
				},
				Adres: &Adres{
					KodKraju: *inv.Buyer.CountryCode,
					AdresL1:  *inv.Buyer.AddressLine1,
				},
				JST: 2,
				GV:  2,
			}
			if inv.Buyer.AddressLine2 != nil {
				fa.Podmiot2.Adres.AdresL2 = *inv.Buyer.AddressLine2
			}
		}
	}

	for _, item := range inv.Items {
		w := FaWiersz{
			NrWierszaFa: fmt.Sprintf("%d", item.LineNumber),
			UU_ID:       fmt.Sprintf("item-%d", item.LineNumber),
			P_7:         item.Name,
			P_11:        fmt.Sprintf("%.2f", item.NetAmount),
			P_12:        string(item.TaxRate),
		}
		if item.Quantity != nil {
			q := fmt.Sprintf("%.2f", *item.Quantity)
			w.P_8B = &q
		}
		if item.UnitPriceNet != nil {
			up := fmt.Sprintf("%.2f", *item.UnitPriceNet)
			w.P_9A = &up
		}
		if item.Unit != nil {
			w.P_8A = item.Unit
		}
		if inv.ExchangeRate != nil {
			rate := fmt.Sprintf("%.6f", *inv.ExchangeRate)
			w.KursWaluty = &rate
		}
		fa.Fa.FaWiersz = append(fa.Fa.FaWiersz, w)
	}
	if inv.InvoiceType == InvoiceTypeKOR || inv.InvoiceType == InvoiceTypeKORZAL || inv.InvoiceType == InvoiceTypeKORROZ {
		fa.Fa.PrzyczynaKorekty = *inv.CorrectionReason
		fa.Fa.TypKorekty = 3
		fa.Fa.DaneFaKorygowanej = &DaneFaKorygowanej{
			DataWystFaKorygowanej: *inv.OriginalIssueDate,
			NrFaKorygowanej:       *inv.OriginalInvoiceNumber,
			NrKSeF:                1,
			NrKSeFFaKorygowanej:   *inv.OriginalKsefId,
		}
	}
	type taxGroup struct {
		net       float64
		tax       float64
		taxPLN    float64
		hasTax    bool
		hasTaxPLN bool
	}
	groups := make(map[int]*taxGroup)
	for _, taxbr := range inv.TaxBreakdowns {
		groupID := 0
		switch taxbr.TaxRate {
		case TaxRate23, TaxRate22:
			groupID = 1
		case TaxRate8, TaxRate7:
			groupID = 2
		case TaxRate5:
			groupID = 3
		case TaxRate4:
			groupID = 4
		case TaxRate0KR:
			groupID = 61
		case TaxRate0WDT:
			groupID = 62
		case TaxRate0EX:
			groupID = 63
		case TaxRateZW:
			groupID = 7
		case TaxRateNPI:
			groupID = 8
		case TaxRateNPII:
			groupID = 9
		case TaxRateOO:
			groupID = 10
		}
		if groups[groupID] == nil {
			groups[groupID] = &taxGroup{}
		}
		groups[groupID].net += taxbr.NetAmount
		if taxbr.TaxAmount != nil {
			groups[groupID].tax += *taxbr.TaxAmount
			groups[groupID].hasTax = true
		}
		if taxbr.TaxAmountPln != nil {
			groups[groupID].taxPLN += *taxbr.TaxAmountPln
			groups[groupID].hasTaxPLN = true
		}
	}
	amountPtr := func(value float64) *string {
		formatted := fmt.Sprintf("%.2f", value)
		return &formatted
	}
	if group := groups[1]; group != nil {
		fa.Fa.P_13_1 = amountPtr(group.net)
		if group.hasTax {
			fa.Fa.P_14_1 = amountPtr(group.tax)
		}
		if group.hasTaxPLN {
			fa.Fa.P_14_1W = amountPtr(group.taxPLN)
		}
	}
	if group := groups[2]; group != nil {
		fa.Fa.P_13_2 = amountPtr(group.net)
		if group.hasTax {
			fa.Fa.P_14_2 = amountPtr(group.tax)
		}
		if group.hasTaxPLN {
			fa.Fa.P_14_2W = amountPtr(group.taxPLN)
		}
	}
	if group := groups[3]; group != nil {
		fa.Fa.P_13_3 = amountPtr(group.net)
		if group.hasTax {
			fa.Fa.P_14_3 = amountPtr(group.tax)
		}
		if group.hasTaxPLN {
			fa.Fa.P_14_3W = amountPtr(group.taxPLN)
		}
	}
	if group := groups[4]; group != nil {
		fa.Fa.P_13_4 = amountPtr(group.net)
		if group.hasTax {
			fa.Fa.P_14_4 = amountPtr(group.tax)
		}
		if group.hasTaxPLN {
			fa.Fa.P_14_4W = amountPtr(group.taxPLN)
		}
	}
	if group := groups[61]; group != nil {
		fa.Fa.P_13_6_1 = amountPtr(group.net)
	}
	if group := groups[62]; group != nil {
		fa.Fa.P_13_6_2 = amountPtr(group.net)
	}
	if group := groups[63]; group != nil {
		fa.Fa.P_13_6_3 = amountPtr(group.net)
	}
	if group := groups[7]; group != nil {
		fa.Fa.P_13_7 = amountPtr(group.net)
	}
	if group := groups[8]; group != nil {
		fa.Fa.P_13_8 = amountPtr(group.net)
	}
	if group := groups[9]; group != nil {
		fa.Fa.P_13_9 = amountPtr(group.net)
	}
	if group := groups[10]; group != nil {
		fa.Fa.P_13_10 = amountPtr(group.net)
	}

	xmlBytes, err := xml.Marshal(fa)
	if err != nil {
		return nil, err
	}
	payload := append([]byte(xml.Header), xmlBytes...)

	return payload, nil
}

func validateNIP(nip string) bool {
	if len(nip) != 10 {
		return false
	}
	weights := []int{6, 5, 7, 2, 3, 4, 5, 6, 7}
	sum := 0
	for i := 0; i < len(weights); i++ {
		val, err := strconv.Atoi(string(nip[i]))
		if err != nil {
			return false
		}
		sum += val * weights[i]
	}
	return sum%11 != 10 && strconv.Itoa(sum%11) == string(nip[len(nip)-1])
}
