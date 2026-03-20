package ksef

import "encoding/xml"

type Faktura struct {
	XMLName xml.Name `xml:"Faktura"`
	Xmlns string `xml:"xmlns,attr"`
	Naglowek Naglowek `xml:"Naglowek"`
	Podmiot1 Podmiot1 `xml:"Podmiot1"`
	Podmiot2 *Podmiot2 `xml:"Podmiot2"`
	Fa Fa `xml:"Fa"`
}

type Naglowek struct {
	KodFormularza KodFormularza `xml:"KodFormularza"`
	WariantFormularza string `xml:"WariantFormularza"`
	DataWytworzeniaFa string `xml:"DataWytworzeniaFa"`
	SystemInfo string `xml:"SystemInfo,omitempty"`
}

type KodFormularza struct {
	KodSystemowy string `xml:"kodSystemowy,attr"`
	WersjaSchemy string `xml:"wersjaSchemy,attr`
	Value string `xml:",chardata"`
}

type Podmiot1 struct {
	PrefiksPodatnika string `xml:"PrefiksPodatnika,omitempty"`
	NIP string `xml:"NIP"`
	DaneIdentyfikacyjne DaneIdentyfikacyjne `xml:"DaneIdentyfikacyjne"`
	Adres Adres `xml:"Adres"`
}

type Podmiot2 struct {
	PrefiksNabywcy string `xml:"PrefiksNabywcy,omitempty"`
	NIP string `xml:"NIP"`
	DaneIdentyfikacyjne DaneIdentyfikacyjne `xml:"DaneIdentyfikacyjne"`
	Adres *Adres `xml:"Adres"`
}

type DaneIdentyfikacyjne struct {
	Nazwa string `xml:"Nazwa"`
}

type Adres struct {
	KodKraju string `xml:"KodKraju"`
	AdresL1 string `xml:"AdresL1"`
	AdresL2 string `xml:"AdresL2,omitempty"`
}

type Fa struct {
	KodWaluty string `xml:"KodWaluty"`
	P_1 string `xml:"P_1"` // data wytworzenia
	P_2 string `xml:"P_2"` // numer faktury
	RodzajFaktury string `xml:"RodzajFaktury"`
	FakturaKorygowana *FakturaKorygowana `xml:"FakturaKorygowana,omitempty"` // omijane jeśli nie KOR
	// taxbreakdowns
	P_13_1 *string `xml:"P_13_1,omitempty"`
	P_14_1 *string `xml:"P_14_1,omitempty"`
	P_13_2 *string `xml:"P_13_2,omitempty"`
	P_14_2 *string `xml:"P_14_2,omitempty"`
	P_13_3 *string `xml:"P_13_3,omitempty"`
	P_14_3 *string `xml:"P_14_3,omitempty"`
	P_13_4 *string `xml:"P_13_4,omitempty"`
	P_14_4 *string `xml:"P_14_4,omitempty"`
	P_13_5 *string `xml:"P_13_5,omitempty"`
	P_14_5 *string `xml:"P_14_5,omitempty"`
	P_13_6_1 *string `xml:"P_13_6_1,omitempty"`
	P_13_6_2 *string `xml:"P_13_6_2,omitempty"`
	P_13_6_3 *string `xml:"P_13_6_3,omitempty"`
	P_13_7 *string `xml:"P_13_7,omitempty"`
	P_13_8 *string `xml:"P_13_8,omitempty"`
	P_15 string `xml:"P_15"` // brutto
	FaWiersz []FaWiersz `xml:"FaWiersz"`
}

type FakturaKorygowana struct {
	PrzyczynaKorekty string `xml:"PrzyczynaKorekty"`
	NrFaKorygowanej string `xml:"NrFaKorygowanej"`
	NrKSeF string `xml:"NrKSeF"`
}

type FaWiersz struct {
	NrWierszaFa string `xml:"NrWierszaFa"`
	P_7Z string `xml:"P_7Z"` // nazwa produktyu
	P_8A *string `xml:"P_8A,omitempty"` // jednostka
	P_8B *string `xml:"P_8B,omitempty"` // ilosc 
	P_9A *string `xml:"P_9A,omitempty"` // cena netto/jednostka
	P_11 string `xml:"P_11"` // wartosc netto
	P_12 string `xml:"P_12"` // podatek
}
