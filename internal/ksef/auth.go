package ksef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	ContextIdentifier struct {
		Type string `json:"type"`
		Value string `json:"value"`
	} `json:"contextIdentifier"`
	EncryptedToken string `json:"encryptedToken"`
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
		ValidUntil time.Time `json:"token"`
	} `json:"accessToken"`
}

// i'm sure this code can be prettier. will refactor.
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
	jsonBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
	}
	if err = json.Unmarshal(jsonBody, &cha); err != nil {
		return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
	}
	return &cha, nil
}

func (c *Client) startSession(encryptedToken string, cha *ChallengeResponse) (*AuthenticationResponse, error) {
	posturl := c.ApiURL + "/auth/ksef-token"
	payload := AuthenticationPayload{
		Challenge: cha.Challenge,
		// anonymous struct used here, didn't want to pollute the top of the file with an additional struct to nest within another one
		ContextIdentifier: struct {
			Type string `json:"type"`
			Value string `json:"value"`
		}{
			Type: "Nip",
			Value: c.NIP,
		},
		EncryptedToken: encryptedToken,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", posturl, bytes.NewBuffer(payloadJson))
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
		return nil, fmt.Errorf("Wystapił błąd - KSeF nie zaakceptował tokena - %d", response.StatusCode)
	}
	var authRes AuthenticationResponse
	responseJson, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
	}
	if err = json.Unmarshal(responseJson, &authRes); err != nil {
		return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
	}
	return &authRes, nil
}

func (c *Client) redeemToken(authRes *AuthenticationResponse) (*RedeemResponse, error) {
	// this function goes too fast. should remove the sleep in the Login function and implement a loop here
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
			responseJson, err := io.ReadAll(response.Body)
			if err != nil {
				return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
			}
			if err = json.Unmarshal(responseJson, &redeemRes); err != nil {
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
	var refreshRes RefreshResponse
	responseJson, err := io.ReadAll(response.Body)
	if err = json.Unmarshal(responseJson, &refreshRes); err != nil {
		return fmt.Errorf("Nie można było odczytać odpowiedzi na refresh request.")
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
	encryptedToken, err := c.encryptToken(cha)
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

// ill either have to call this every time i try to send an invoice, or create a function wrapper for sending requests
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
