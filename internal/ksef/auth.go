package ksef

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
	Timestamp time.Time `json:"timestamp"`
	TimestampMs int64 `json:"timestampMs"`
}

type AuthenticationPayload struct {
	Challenge string `json:"challenge"`
	ContextIdentifier ContextIdentifier `json:"contextIdentifier"`
	EncryptedToken []byte `json:"encryptedToken"`
}

type ContextIdentifier struct {
	Type string `json:"type"`
	Value string `json:"value"`
}

type AuthenticationResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
	AuthenticationToken struct {
		Token string `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"authenticationToken"`
}

type RedeemResponse struct {
	AccessToken struct {
		Token string `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"accessToken"`
	RefreshToken struct {
		Token string `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	}
}

type RefreshResponse struct {
	AccessToken struct {
		Token string `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"accessToken"`
}

type InteractiveSessionPayload struct {
	FormCode SessionFormCode `json:"formCode"`
	Encryption Encryption `json:"encryption"`
}

type SessionFormCode struct {
	SystemCode string `json:"systemCode"`
	SchemaVersion string `json:"schemaVersion"`
	Value string `json:"value"`
}

type Encryption struct {
	EncryptedSymmetricKey []byte `json:"encryptedSymmetricKey"`
	InitializationVector []byte `json:"initializationVector"`
}

type InteractiveSessionResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
	ValidUntil time.Time `json:"validUntil"`
}

func (c *Client) getChallenge() (*ChallengeResponse, error) {
	posturl := c.ApiURL + "/auth/challenge"
	r, err := http.NewRequest("POST", posturl, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Wystąpił błąd - KSeF zwrócił odpowiedź o kodzie %d.", response.StatusCode)
	}
	var cha ChallengeResponse
	if err := json.NewDecoder(response.Body).Decode(&cha); err != nil {
		return nil, err
	}
	return &cha, nil
}

func (c *Client) startSession(encryptedToken []byte, cha *ChallengeResponse) (*AuthenticationResponse, error) {
	posturl := c.ApiURL + "/auth/ksef-token"
	payload := AuthenticationPayload{
		Challenge: cha.Challenge,
		ContextIdentifier: ContextIdentifier{
			Type: "Nip",
			Value: c.NIP,
		},
		EncryptedToken: encryptedToken,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&payload); err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", posturl, &buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("Wystąpił błąd - KSeF nie zaakceptował tokena - odpowiedź o kodzie %d.", response.StatusCode)
	}
	var authRes AuthenticationResponse
	if err := json.NewDecoder(response.Body).Decode(&authRes); err != nil {
		return nil, err
	}
	return &authRes, nil
}

func (c *Client) redeemToken(authRes *AuthenticationResponse) (*RedeemResponse, error) {
	posturl := c.ApiURL + "/auth/token/redeem"
	for i := 0; i < 10; i++ {
		r, err := http.NewRequest("POST", posturl, bytes.NewBuffer([]byte("{}")))
		
		if err != nil {
			return nil, err
		}
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Authorization", "Bearer " + authRes.AuthenticationToken.Token)
		r.Header.Set("Content-Type", "application/json")
		response, err := c.httpClient.Do(r)
		if err != nil {
			return nil, err
		}
		if response.StatusCode == http.StatusOK {
			defer response.Body.Close()
			var redeemRes RedeemResponse
			if err := json.NewDecoder(response.Body).Decode(&redeemRes); err != nil {
				return nil, err
			}
			return &redeemRes, nil
		}
		response.Body.Close()
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Second * 2)
			continue
		}
		return nil, fmt.Errorf("KSeF nie zwrócił tokena - błąd %d", response.StatusCode)

	}
	return nil, fmt.Errorf("Nie udało się pobrać tokena.")
}

func (c *Client) refreshToken() error {
	if c.RefreshToken == "" {
		return fmt.Errorf("Nie można było znaleźć tokena do odświeżenia.")
	}
	posturl := c.ApiURL + "/auth/token/refresh"
	r, err := http.NewRequest("POST", posturl, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer " + c.RefreshToken)
	r.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("KSeF nie odświeżył tokena - błąd %d.", response.StatusCode)
	}
	var refreshRes RefreshResponse
	if err := json.NewDecoder(response.Body).Decode(&refreshRes); err != nil {
		return err
	}
	c.SessionToken = refreshRes.AccessToken.Token
	c.SessionTokenValidity = refreshRes.AccessToken.ValidUntil
	return nil
}

func (c *Client) Login() error {
	cha, err := c.getChallenge()
	if err != nil {
		return err
	}
	encryptedToken, err := c.encryptWithPKey([]byte(fmt.Sprintf("%s|%d", c.ApiToken, cha.TimestampMs)))
	if err != nil {
		return err
	}
	authRes, err := c.startSession(encryptedToken, cha)
	if err != nil {
		return err
	}
	redeemRes, err := c.redeemToken(authRes)
	if err != nil {
		return err
	}
	c.SessionToken = redeemRes.AccessToken.Token
	c.SessionTokenValidity = redeemRes.AccessToken.ValidUntil
	c.RefreshToken = redeemRes.RefreshToken.Token
	c.RefreshTokenValidity = redeemRes.RefreshToken.ValidUntil
	return nil
}

func (c *Client) checkToken() error {
	if time.Until(c.SessionTokenValidity) < 2*time.Minute {
		if err := c.refreshToken(); err != nil {
			return c.Login()
		}
	}
	return nil
}

// this will be used instead of c.httpClient.Do
func (c *Client) ExecuteRequestTokenCheck(r *http.Request) (*http.Response, error) {
	if err := c.checkToken(); err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Bearer " + c.SessionToken)
	return c.httpClient.Do(r)
}

func (c *Client) OpenInSession() error {
	aesKey := make([]byte, 32)
	iv := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return fmt.Errorf("Błąd w generowaniu klucza: %v", err)
	}
	if _, err := rand.Read(iv); err != nil {
		return fmt.Errorf("Błąd w generowaniu wektora inicjalizującego: %v", err)
	}
	encryptedKey, err := c.encryptWithPKey(aesKey)
	if err != nil {
		return fmt.Errorf("Błąd przy enkrypcji klucza: %v", err)
	}
	payload := InteractiveSessionPayload{
		FormCode: SessionFormCode{
			SystemCode: "FA (3)",
			SchemaVersion: "1-0E",
			Value: "FA",
		},
		Encryption: Encryption{
			EncryptedSymmetricKey: encryptedKey,
			InitializationVector: iv,
		},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&payload); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.ApiURL + "/sessions/online", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := c.ExecuteRequestTokenCheck(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Nie można było rozpocząć sesji interaktywnej, Ksef zwrócił %v.", res.StatusCode)
	}
	var inRes InteractiveSessionResponse
	if err := json.NewDecoder(res.Body).Decode(&inRes); err != nil {
		return err
	}
	c.InSessionRef= inRes.ReferenceNumber
	c.InSessionAESKey = aesKey
	c.InSessionInitializationVector = iv
	c.InSessionValidity = inRes.ValidUntil
	fmt.Println(c.InSessionRef)
	return nil
}

func (c *Client) CloseInSession() error {
	// always remove the details, even if session is already closed
	defer func() {
		c.InSessionRef = ""
	}()
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/sessions/online/%s/close", c.ApiURL, c.InSessionRef), nil)
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
					ExceptionCode int `json:"exceptionCode"`
					ExceptionDescription string `json:"exceptionDescription"`
				} `json:"exceptionDetailList"`
			} `json:"exception"`
		}
		json.NewDecoder(res.Body).Decode(&errRes)
		return fmt.Errorf("Ksef nie zamknął sesji i zwrócił kod statusu %v - %v - %v", res.StatusCode, errRes.Exception.ExceptionDetailList[0].ExceptionDescription, errRes.Exception.ExceptionDetailList[0].ExceptionCode)
	}
	return nil
}
