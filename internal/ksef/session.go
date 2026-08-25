package ksef

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type InSession struct {
	InSessionRef                  string
	InSessionAESKey               []byte
	InSessionInitializationVector []byte
	InSessionValidity             time.Time
}

func (c *Client) OpenInSession() (*InSession, error) {
	if err := c.checkKeys(); err != nil {
		return nil, err
	}
	aesKey := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("could not generate the encryption key: %w", err)
	}
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("could not generate the initialization vector: %w", err)
	}
	encryptedKey, err := c.encryptWithPKey(aesKey, c.SessionPublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt the session key: %w", err)
	}
	payload := InteractiveSessionPayload{
		FormCode: SessionFormCode{
			SystemCode:    "FA (3)",
			SchemaVersion: "1-0E",
			Value:         "FA",
		},
		Encryption: Encryption{
			EncryptedSymmetricKey: encryptedKey,
			InitializationVector:  iv,
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&payload); err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.ApiURL+"/sessions/online", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("could not open a KSeF interactive session: HTTP %d", res.StatusCode)
	}
	var inRes InteractiveSessionResponse
	if err := json.NewDecoder(res.Body).Decode(&inRes); err != nil {
		return nil, err
	}
	inSession := &InSession{
		InSessionRef:                  inRes.ReferenceNumber,
		InSessionAESKey:               aesKey,
		InSessionInitializationVector: iv,
		InSessionValidity:             inRes.ValidUntil,
	}
	return inSession, nil
}

func (c *Client) CloseInSession(inSession *InSession) error {
	// always remove the details, even if session is already closed
	defer func() {
		inSession.InSessionRef = ""
	}()
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/sessions/online/%s/close", c.ApiURL, inSession.InSessionRef), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		var errRes struct {
			Exception struct {
				ExceptionDetailList []struct {
					ExceptionCode        int    `json:"exceptionCode"`
					ExceptionDescription string `json:"exceptionDescription"`
				} `json:"exceptionDetailList"`
			} `json:"exception"`
		}
		if err := json.NewDecoder(res.Body).Decode(&errRes); err == nil && len(errRes.Exception.ExceptionDetailList) > 0 {
			detail := errRes.Exception.ExceptionDetailList[0]
			return fmt.Errorf("could not close the KSeF interactive session: HTTP %d: %s (%d)", res.StatusCode, detail.ExceptionDescription, detail.ExceptionCode)
		}
		return fmt.Errorf("could not close the KSeF interactive session: HTTP %d", res.StatusCode)
	}
	return nil
}
