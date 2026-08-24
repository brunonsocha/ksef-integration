package ksef

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type InvoicePayload struct {
	InvoiceHash             []byte `json:"invoiceHash"`
	InvoiceSize             int64  `json:"invoiceSize"`
	EncryptedInvoiceHash    []byte `json:"encryptedInvoiceHash"`
	EncryptedInvoiceSize    int64  `json:"encryptedInvoiceSize"`
	EncryptedInvoiceContent []byte `json:"encryptedInvoiceContent"`
	OfflineMode             bool   `json:"offlineMode"`
}

type InvoiceResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
}

type InvoiceStatusRes struct {
	OrdinalNumber                int     `json:"ordinalNumber"`
	ReferenceNumber              string  `json:"referenceNumber"`
	InvoiceNumber                *string `json:"invoiceNumber"`
	KsefNumber                   *string `json:"ksefNumber"`
	InvoiceHash                  string  `json:"invoiceHash"`
	InvoiceFileName              *string `json:"invoiceFileName"`
	AcquisitionDate              *string `json:"acquisitionDate"`
	InvoicingDate                string  `json:"invoicingDate"`
	PermanentStorageDate         *string `json:"permanentStorageDate"`
	UpoDownloadUrl               *string `json:"upoDownloadUrl"`
	UpoDownloadUrlExpirationDate *string `json:"upoDownloadUrlExpirationDate"`
	InvoicingMode                *string `json:"invoicingMode"`
	Status                       Status  `json:"status"`
}

type Status struct {
	Code        int               `json:"code"`
	Description string            `json:"description"`
	Details     []string          `json:"details"`
	Extensions  map[string]string `json:"extensions"`
}

type pollingOutcome int

const (
	successRes pollingOutcome = iota
	processingRes
	rejected
	rateLimited
	temporaryFailure
	badResponse
	httpFail
)

type pollingOutcomeRes struct {
	outcome    pollingOutcome
	statusRes  *InvoiceStatusRes
	retryAfter time.Duration
	err        error
	httpStatus int
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
		InvoiceHash:             invHash,
		InvoiceSize:             invSize,
		EncryptedInvoiceHash:    encInvHash,
		EncryptedInvoiceSize:    encInvSize,
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
					ExceptionCode        int    `json:"exceptionCode"`
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

func (c *Client) getInvoiceStatus(sessionRef, invoiceRef string) pollingOutcomeRes {
	// int is the retry after in case of rate limiting
	fullUrl := fmt.Sprintf("%s/sessions/%s/invoices/%s", c.ApiURL, sessionRef, invoiceRef)
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return pollingOutcomeRes{
			outcome: httpFail,
			err:     err,
		}
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return pollingOutcomeRes{
			outcome: temporaryFailure,
			err:     err,
		}
	}
	defer res.Body.Close()
	var invStatRes InvoiceStatusRes

	switch res.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(res.Body).Decode(&invStatRes); err != nil {
			return pollingOutcomeRes{
				outcome:    badResponse,
				err:        err,
				httpStatus: res.StatusCode,
			}
		}
		switch invStatRes.Status.Code {
		case 100, 150:
			// can't use a 150 constant, as the KSeF API docs say that 150 is processing. added processing just in case (102)
			return pollingOutcomeRes{
				outcome:    processingRes,
				statusRes:  &invStatRes,
				httpStatus: res.StatusCode,
			}
		case 200:
			return pollingOutcomeRes{
				outcome:    successRes,
				statusRes:  &invStatRes,
				httpStatus: res.StatusCode,
			}
		case 405, 410, 415, 430, 435, 440, 450:
			return pollingOutcomeRes{
				outcome:    rejected,
				statusRes:  &invStatRes,
				httpStatus: res.StatusCode,
			}
		case 500, 550:
			return pollingOutcomeRes{
				outcome:    temporaryFailure,
				statusRes:  &invStatRes,
				err:        fmt.Errorf("KSeF zwrócił błąd: %s", invStatRes.Status.Description),
				httpStatus: res.StatusCode,
			}
		default:
			return pollingOutcomeRes{
				outcome:    badResponse,
				statusRes:  &invStatRes,
				err:        fmt.Errorf("KSeF zwrócił nieznany błąd: %s", invStatRes.Status.Description),
				httpStatus: res.StatusCode,
			}
		}
	case http.StatusTooManyRequests:
		retryAfter := 5 * time.Second
		if header := res.Header.Get("Retry-After"); header != "" {
			seconds, err := strconv.Atoi(header)
			if err != nil {
				return pollingOutcomeRes{
					outcome:    rateLimited,
					retryAfter: retryAfter,
					err:        fmt.Errorf("KSeF otrzymał za dużo żądań."),
					httpStatus: res.StatusCode,
				}
			}
			retryAfter = time.Duration(seconds) * time.Second
		}
		return pollingOutcomeRes{
			outcome:    rateLimited,
			retryAfter: retryAfter,
			err:        fmt.Errorf("KSeF otrzymał za dużo żądań."),
			httpStatus: res.StatusCode,
		}
	default:
		if res.StatusCode >= 500 {
			return pollingOutcomeRes{
				outcome:    temporaryFailure,
				err:        fmt.Errorf("Tymczasowy błąd: %d", res.StatusCode),
				httpStatus: res.StatusCode,
			}
		}
		return pollingOutcomeRes{
			outcome:    httpFail,
			err:        fmt.Errorf("KSeF zwrócił błąd: %d", res.StatusCode),
			httpStatus: res.StatusCode,
		}
	}
}

func (c *Client) WaitForSendingConfirmation(maxAttempts int, sessionRef, invoiceRef string) (*InvoiceStatusRes, error) {
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		pollingStatus := c.getInvoiceStatus(sessionRef, invoiceRef)
		switch pollingStatus.outcome {
		case processingRes:
			if i == maxAttempts-1 {
				return nil, errors.New("KSeF nie potwierdził statusu faktury.")
			}
			time.Sleep(c.PollingDelay)
		case successRes:
			if pollingStatus.statusRes.KsefNumber == nil {
				return nil, fmt.Errorf("KSeF potwierdził otrzymanie faktury, ale nie nadał numeru KSeF")
			}
			return pollingStatus.statusRes, nil
		case rateLimited:
			lastErr = pollingStatus.err
			time.Sleep(pollingStatus.retryAfter)
		case temporaryFailure:
			lastErr = pollingStatus.err
			if i == maxAttempts-1 {
				return nil, fmt.Errorf("Nie można było wysłać faktury")
			}
			time.Sleep(c.PollingDelay)
		case httpFail, badResponse:
			lastErr = pollingStatus.err
			return nil, fmt.Errorf("Wystąpił błąd: %v - %d", pollingStatus.err, pollingStatus.httpStatus)
		case rejected:
			return nil, fmt.Errorf("%w: %d - %s - %s - %s", INVOICE_REJECTED_ERR, pollingStatus.statusRes.Status.Code, pollingStatus.statusRes.Status.Description, pollingStatus.statusRes.Status.Extensions, pollingStatus.statusRes.Status.Details)
		}
	}
	return nil, fmt.Errorf("Doszło do timeoutu po %d próbach, ostatni znany błąd to %v.", maxAttempts, lastErr)
}

func (c *Client) DownloadUPO(upoUrl string) ([]byte, error) {
	res, err := c.httpClient.Get(upoUrl)
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
