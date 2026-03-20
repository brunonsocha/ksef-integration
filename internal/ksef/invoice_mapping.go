package ksef

import (
	"encoding/xml"
	"errors"
	"fmt"
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
	defaultCountryCode := "PL"
	if i.Seller.CountryCode == nil {
		i.Seller.CountryCode = &defaultCountryCode
	}
	if i.Buyer.Name != nil && i.Buyer.CountryCode == nil {
		i.Buyer.CountryCode = &defaultCountryCode
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
				Value: "FA",
			},
			WariantFormularza: "3",
			DataWytworzeniaFa: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		},
		Podmiot1: Podmiot1{
			NIP: inv.Seller.Nip,
			DaneIdentyfikacyjne: DaneIdentyfikacyjne{
				Nazwa: *inv.Seller.Name,
			},
			Adres: Adres{
				KodKraju: *inv.Seller.CountryCode,
				AdresL1: *inv.Seller.AddressLine1,
				},
			},
		Fa: Fa{
			KodWaluty: *inv.Currency,
			P_1: inv.IssueDate,
			P_2: inv.InvoiceNumber,
			RodzajFaktury: string(inv.InvoiceType),
			P_15: fmt.Sprintf("%.2f", inv.TotalAmount),
		},
	}
	if inv.Seller.AddressLine2 != nil {
		fa.Podmiot1.Adres.AdresL2 = *inv.Seller.AddressLine2
	}

	if inv.Buyer.Name != nil && inv.Buyer.AddressLine1 != nil {
		fa.Podmiot2 = &Podmiot2{
			NIP: inv.Buyer.Nip,
			DaneIdentyfikacyjne: DaneIdentyfikacyjne{
				Nazwa: *inv.Buyer.Name,
			},
			Adres: &Adres{
				KodKraju: *inv.Buyer.CountryCode,
				AdresL1: *inv.Buyer.AddressLine1,
			},
		}
		if inv.Buyer.AddressLine2 != nil {
			fa.Podmiot2.Adres.AdresL2 = *inv.Buyer.AddressLine2
		}
	}
	

	for _, item := range inv.Items {
		w := FaWiersz{
			NrWierszaFa: fmt.Sprintf("%d", item.LineNumber),
			P_7Z: item.Name,
			P_11: fmt.Sprintf("%.2f", item.NetAmount),
			P_12: string(item.TaxRate),
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
		fa.Fa.FaWiersz = append(fa.Fa.FaWiersz, w)
	}
	
	for _, taxbr := range inv.TaxBreakdowns {
		net := fmt.Sprintf("%.2f", taxbr.NetAmount)
		var tax string
		if taxbr.TaxAmount != nil {
			tax = fmt.Sprintf("%.2f", *taxbr.TaxAmount)
		}
		switch taxbr.TaxRate {
		case TaxRate23, TaxRate22:
			fa.Fa.P_13_1 = &net
			if taxbr.TaxAmount != nil {
				fa.Fa.P_14_1 = &tax
			}
		case TaxRate8, TaxRate7:
			fa.Fa.P_13_2 = &net
			if taxbr.TaxAmount != nil {
				fa.Fa.P_14_2 = &tax
			}
		case TaxRate5:
			fa.Fa.P_13_3 = &net
			if taxbr.TaxAmount != nil {
				fa.Fa.P_14_3 = &tax
			}
		case TaxRate4: // Taxis
			fa.Fa.P_13_4 = &net
			if taxbr.TaxAmount != nil {
				fa.Fa.P_14_4 = &tax
			}
		case TaxRate0KR:
			fa.Fa.P_13_6_1 = &net
		case TaxRate0WDT:
			fa.Fa.P_13_6_2 = &net
		case TaxRate0EX:
			fa.Fa.P_13_6_3 = &net
		case TaxRateZW:
			fa.Fa.P_13_7 = &net
		case TaxRateNPI, TaxRateNPII, TaxRateOO:
			fa.Fa.P_13_8 = &net
		}
	}

	xmlBytes, err := xml.Marshal(fa)
	if err != nil {
		return nil, err
	}
	payload := append([]byte(xml.Header), xmlBytes...)

	return payload, nil
}

