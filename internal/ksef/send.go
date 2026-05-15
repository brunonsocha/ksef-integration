package ksef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
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

type InvoiceStatusRes struct {
	OrdinalNumber int `json:"ordinalNumber"`
	ReferenceNumber string `json:"referenceNumber"`
	InvoiceNumber *string `json:"invoiceNumber"`
	KsefNumber *string `json:"ksefNumber"`
	InvoiceHash string `json:"invoiceHash"`
	InvoiceFileName *string `json:"invoiceFileName"`
	AcquisitionDate *string `json:"acquisitionDate"`
	InvoicingDate string `json:"invoicingDate"`
	PermanentStorageDate *string `json:"permanentStorageDate"`
	UpoDownloadUrl *string `json:"upoDownloadUrl"`
	UpoDownloadUrlExpirationDate *string `json:"upoDownloadUrlExpirationDate"`
	InvoicingMode *string `json:"invoicingMode"`
	Status Status `json:"status"`
}

type Status struct {
	Code int `json:"code"`
	Description string `json:"description"`
	Details []string `json:"details"`
	Extensions map[string]string `json:"extensions"`
}

func (c *Client) SendInvoice(raw_xml []byte, inSession *InSession) (string, error) {
	invHash := hashSHA256(raw_xml)
	invSize := int64(len(raw_xml))
	encInvCon, err := c.encryptCBC(raw_xml, inSession)
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
	posturl := fmt.Sprintf("%s/sessions/online/%s/invoices", c.ApiURL, inSession.InSessionRef)
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
		if err := json.NewDecoder(res.Body).Decode(&errRes); err != nil {
			return "", err
		}
		if len(errRes.Exception.ExceptionDetailList) > 0 {
			if errRes.Exception.ExceptionDetailList[0].ExceptionCode == 21180 {
				return "", INVALID_SESSION_ERR
			}
			return "", fmt.Errorf("Ksef zwrócił błąd - kod statusu: %v - %v - %v", res.StatusCode, errRes.Exception.ExceptionDetailList[0].ExceptionDescription, errRes.Exception.ExceptionDetailList[0].ExceptionCode)
		}
		return "", fmt.Errorf("KSeF zwrócił błąd - kod statusu: %v - bez detali.", res.StatusCode)
	}
	var invRes InvoiceResponse
	if err := json.NewDecoder(res.Body).Decode(&invRes); err != nil {
		return "", err
	}
	return invRes.ReferenceNumber, nil
}

func (c *Client) getInvoiceStatus(sessionRef, invoiceRef string) (*InvoiceStatusRes, int, error) {
	// int is the retry after in case of rate limiting
	fullUrl := fmt.Sprintf("%s/sessions/%s/invoices/%s", c.ApiURL, sessionRef, invoiceRef)
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		var invStatRes InvoiceStatusRes
		if err := json.NewDecoder(res.Body).Decode(&invStatRes); err != nil {
			return nil, 0, err
		}
		return &invStatRes, 0, nil
	}
	if res.StatusCode == http.StatusTooManyRequests {
		retryAfter := res.Header.Get("Retry-After")
		retryAfterInt, err := strconv.Atoi(retryAfter)
		if err != nil || retryAfterInt == 0 {
			retryAfterInt = 5 // hardcoded fallback of 5 seconds
		}
		return nil, retryAfterInt, fmt.Errorf("Limit żądań przekroczony - następne zapytanie można wykonać za %d sekund.", retryAfterInt)
	}

	return nil, 0, fmt.Errorf("Wystąpił błąd w sprawdzaniu statusu wysyłki - KSeF zwrócił kod statusu %d.", res.StatusCode)
}

func (c *Client) WaitForSendingConfirmation(maxAttempts int, sessionRef, invoiceRef string) (*InvoiceStatusRes, error) {
	for i := 0; i < maxAttempts; i++ {
		invStatRes, retryAfter, err := c.getInvoiceStatus(sessionRef, invoiceRef)
		if err != nil {
			if retryAfter > 0 {
				time.Sleep(time.Duration(retryAfter) * time.Second)
				continue
			}
			time.Sleep(time.Second * 5)
			continue
		}
		switch invStatRes.Status.Code {
		// stupid design. these mean ksef might be still processing, yet if such response was received on the last attempt, the invoice will be marked as FAILED.
		// new status and polling for UPO?
		case 100, 150:
			time.Sleep(time.Second * 5)
			if i == maxAttempts-1 {
				return nil, UNKNOWN_STATE_ERR
			}
			continue
		case 200:
			if invStatRes.KsefNumber == nil {
				return nil, fmt.Errorf("KSeF potwierdził otrzymanie faktury - ale nie nadał numeru KSeF.")
			}
			return invStatRes, nil
		default:
			return nil, fmt.Errorf("Faktura została odrzucona przez KSeF - KSeF zwrócił kod %d - błąd %s", invStatRes.Status.Code, invStatRes.Status.Description)
		}
	}
	return nil, fmt.Errorf("Nie udało się pozyskać statusu wysłanej faktury w %d prób.", maxAttempts)
}

func (c *Client) DownloadUPO(upoUrl string) ([]byte, error) {
	res, err := http.Get(upoUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Wystąpił błąd przy pobieraniu UPO - kod statusu %d.", res.StatusCode)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
