package ksef

import (
	"encoding/xml"
)

type Faktura struct {
	XMLName  xml.Name  `xml:"Faktura"`
	Xmlns    string    `xml:"xmlns,attr"`
	Naglowek Naglowek  `xml:"Naglowek"`
	Podmiot1 Podmiot1  `xml:"Podmiot1"`
	Podmiot2 *Podmiot2 `xml:"Podmiot2"`
	Fa       Fa        `xml:"Fa"`
}

type Naglowek struct {
	KodFormularza     KodFormularza `xml:"KodFormularza"`
	WariantFormularza string        `xml:"WariantFormularza"`
	DataWytworzeniaFa string        `xml:"DataWytworzeniaFa"`
	SystemInfo        string        `xml:"SystemInfo,omitempty"`
}

type KodFormularza struct {
	KodSystemowy string `xml:"kodSystemowy,attr"`
	WersjaSchemy string `xml:"wersjaSchemy,attr"`
	Value        string `xml:",chardata"`
}

type Podmiot1 struct {
	DaneIdentyfikacyjne DaneIdentyfikacyjne `xml:"DaneIdentyfikacyjne"`
	Adres               Adres               `xml:"Adres"`
}

type Podmiot2 struct {
	DaneIdentyfikacyjne DaneIdentyfikacyjne `xml:"DaneIdentyfikacyjne"`
	Adres               *Adres              `xml:"Adres"`
	JST                 int                 `xml:"JST"`
	GV                  int                 `xml:"GV"`
}

type DaneIdentyfikacyjne struct {
	NIP   string `xml:"NIP"`
	Nazwa string `xml:"Nazwa"`
}

type Adres struct {
	KodKraju string `xml:"KodKraju"`
	AdresL1  string `xml:"AdresL1"`
	AdresL2  string `xml:"AdresL2,omitempty"`
}

type Fa struct {
	KodWaluty string `xml:"KodWaluty"`
	P_1       string `xml:"P_1"`           // data wytworzenia
	P_2       string `xml:"P_2"`           // numer faktury
	P_6       string `xml:"P_6,omitempty"` // data sprzedazy
	// taxbreakdowns
	P_13_1            *string            `xml:"P_13_1,omitempty"`
	P_14_1            *string            `xml:"P_14_1,omitempty"`
	P_14_1W           *string            `xml:"P_14_1W,omitempty"`
	P_13_2            *string            `xml:"P_13_2,omitempty"`
	P_14_2            *string            `xml:"P_14_2,omitempty"`
	P_14_2W           *string            `xml:"P_14_2W,omitempty"`
	P_13_3            *string            `xml:"P_13_3,omitempty"`
	P_14_3            *string            `xml:"P_14_3,omitempty"`
	P_14_3W           *string            `xml:"P_14_3W,omitempty"`
	P_13_4            *string            `xml:"P_13_4,omitempty"`
	P_14_4            *string            `xml:"P_14_4,omitempty"`
	P_14_4W           *string            `xml:"P_14_4W,omitempty"`
	P_13_5            *string            `xml:"P_13_5,omitempty"`
	P_14_5            *string            `xml:"P_14_5,omitempty"`
	P_13_6_1          *string            `xml:"P_13_6_1,omitempty"`
	P_13_6_2          *string            `xml:"P_13_6_2,omitempty"`
	P_13_6_3          *string            `xml:"P_13_6_3,omitempty"`
	P_13_7            *string            `xml:"P_13_7,omitempty"`
	P_13_8            *string            `xml:"P_13_8,omitempty"`
	P_13_9            *string            `xml:"P_13_9,omitempty"`
	P_13_10           *string            `xml:"P_13_10,omitempty"`
	P_15              string             `xml:"P_15"` // brutto
	Adnotacje         Adnotacje          `xml:"Adnotacje"`
	RodzajFaktury     string             `xml:"RodzajFaktury"`
	PrzyczynaKorekty  string             `xml:"PrzyczynaKorekty,omitempty"`
	TypKorekty        int                `xml:"TypKorekty,omitempty"`
	DaneFaKorygowanej *DaneFaKorygowanej `xml:"DaneFaKorygowanej,omitempty"` // omijane jeśli nie KOR
	FaWiersz          []FaWiersz         `xml:"FaWiersz"`
	Platnosc          *Platnosc          `xml:"Platnosc,omitempty"`
}

type Adnotacje struct {
	P_16                 int         `xml:"P_16"`
	P_17                 int         `xml:"P_17"`
	P_18                 int         `xml:"P_18"`
	P_18A                int         `xml:"P_18A"`
	Zwolnienie           ZwolnienieN `xml:"Zwolnienie"`
	NoweSrodkiTransportu NoweSrodkiN `xml:"NoweSrodkiTransportu"`
	P_23                 int         `xml:"P_23"`
	PMarzy               PMarzyN     `xml:"PMarzy"`
}

type ZwolnienieN struct {
	P_19N int `xml:"P_19N"`
}

type NoweSrodkiN struct {
	P_22N int `xml:"P_22N"`
}

type PMarzyN struct {
	P_PMarzyN int `xml:"P_PMarzyN"`
}

type DaneFaKorygowanej struct {
	DataWystFaKorygowanej string `xml:"DataWystFaKorygowanej"`
	NrFaKorygowanej       string `xml:"NrFaKorygowanej"`
	NrKSeF                int    `xml:"NrKSeF"`
	NrKSeFFaKorygowanej   string `xml:"NrKSeFFaKorygowanej"`
}

type FaWiersz struct {
	NrWierszaFa string  `xml:"NrWierszaFa"`
	UU_ID       string  `xml:"UU_ID"`
	P_7         string  `xml:"P_7"`            // nazwa produktyu
	P_8A        *string `xml:"P_8A,omitempty"` // jednostka
	P_8B        *string `xml:"P_8B,omitempty"` // ilosc
	P_9A        *string `xml:"P_9A,omitempty"` // cena netto/jednostka
	P_11        string  `xml:"P_11"`           // wartosc netto
	P_12        string  `xml:"P_12"`           // podatek
	KursWaluty  *string `xml:"KursWaluty,omitempty"`
}

type Platnosc struct {
	TerminPlatnosci *TerminPlatnosci `xml:"TerminPlatnosci,omitempty"`
	FormaPlatnosci  *string          `xml:"FormaPlatnosci,omitempty"`
	RachunekBankowy *RachunekBankowy `xml:"RachunekBankowy,omitempty"`
}

type TerminPlatnosci struct {
	Termin string `xml:"Termin"`
}

type RachunekBankowy struct {
	NrRB       string  `xml:"NrRB"`
	NazwaBanku *string `xml:"NazwaBanku,omitempty"`
}
