package ksef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type InvoicePayload struct {
	InvoiceHash []byte `json:"invoiceHash"`
	InvoiceSize int64 `json:"invoiceSize"`
	EncryptedInvoiceHash []byte `json:"encryptedInvoiceHash"`
	EncryptedInvoiceSize int64 `json:"encryptedInvoiceSize"`
	EncryptedInvoiceContent []byte `json:"encryptedInvoiceContent"`
	OfflineMode bool `json:"offlineMode"`
}

type InvoiceResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
}

func (c *Client) SendInvoice(raw_xml []byte) (string, error) {
	invHash := hashSHA256(raw_xml)
	invSize := int64(len(raw_xml))
	encInvCon, err := c.encryptCBC(raw_xml)
	if err != nil {
		return "", fmt.Errorf("Błąd w szyfrowaniu faktury: %v", err)
	}
	encInvSize := int64(len(encInvCon))
	encInvHash := hashSHA256(encInvCon)
	payload := InvoicePayload{
		InvoiceHash: invHash,
		InvoiceSize: invSize,
		EncryptedInvoiceHash: encInvHash,
		EncryptedInvoiceSize: encInvSize,
		EncryptedInvoiceContent: encInvCon,
	}
	posturl := fmt.Sprintf("%s/sessions/online/%s/invoices", c.ApiURL, c.InSessionRef)
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&payload); err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", posturl, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		var errRes struct {
			Exception struct {
				ExceptionDetailList []struct {
					ExceptionCode int `json:"exceptionCode"`
					ExceptionDescription string `json:"exceptionDescription"`
				} `json:"exceptionDetailList"`
			} `json:"exception"`
		}
		json.NewDecoder(res.Body).Decode(&errRes)
		if errRes.Exception.ExceptionDetailList[0].ExceptionCode == 21180 {
			c.InSessionRef = ""
		}
		return "", fmt.Errorf("Ksef zwrócił błąd - kod statusu: %v - %v - %v", res.StatusCode, errRes.Exception.ExceptionDetailList[0].ExceptionDescription, errRes.Exception.ExceptionDetailList[0].ExceptionCode)
	}
	var invRes InvoiceResponse
	if err := json.NewDecoder(res.Body).Decode(&invRes); err != nil {
		return "", err
	}
	return invRes.ReferenceNumber, nil
}
