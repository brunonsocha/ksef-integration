package ksef

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type ChallengeResponse struct {
	Challenge   string    `json:"challenge"`
	Timestamp   time.Time `json:"timestamp"`
	TimestampMs int64     `json:"timestampMs"`
}

type AuthenticationPayload struct {
	Challenge         string            `json:"challenge"`
	ContextIdentifier ContextIdentifier `json:"contextIdentifier"`
	EncryptedToken    []byte            `json:"encryptedToken"`
}

type ContextIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AuthenticationResponse struct {
	ReferenceNumber     string `json:"referenceNumber"`
	AuthenticationToken struct {
		Token      string    `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"authenticationToken"`
}

type RedeemResponse struct {
	AccessToken struct {
		Token      string    `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"accessToken"`
	RefreshToken struct {
		Token      string    `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	}
}

type RefreshResponse struct {
	AccessToken struct {
		Token      string    `json:"token"`
		ValidUntil time.Time `json:"validUntil"`
	} `json:"accessToken"`
}

type InteractiveSessionPayload struct {
	FormCode   SessionFormCode `json:"formCode"`
	Encryption Encryption      `json:"encryption"`
}

type SessionFormCode struct {
	SystemCode    string `json:"systemCode"`
	SchemaVersion string `json:"schemaVersion"`
	Value         string `json:"value"`
}

type Encryption struct {
	EncryptedSymmetricKey []byte `json:"encryptedSymmetricKey"`
	InitializationVector  []byte `json:"initializationVector"`
}

type InteractiveSessionResponse struct {
	ReferenceNumber string    `json:"referenceNumber"`
	ValidUntil      time.Time `json:"validUntil"`
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
		return nil, fmt.Errorf("could not obtain an authentication challenge from KSeF: HTTP %d", response.StatusCode)
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
			Type:  "Nip",
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
		return nil, fmt.Errorf("KSeF rejected the authentication token: HTTP %d", response.StatusCode)
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
		r.Header.Set("Authorization", "Bearer "+authRes.AuthenticationToken.Token)
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
			time.Sleep(c.AuthRetryDelay)
			continue
		}
		return nil, fmt.Errorf("KSeF did not return an access token: HTTP %d", response.StatusCode)

	}
	return nil, errors.New("could not obtain an access token from KSeF")
}

func (c *Client) refreshToken() error {
	if c.RefreshToken == "" {
		return errors.New("could not refresh the KSeF token: no current token is available")
	}
	posturl := c.ApiURL + "/auth/token/refresh"
	r, err := http.NewRequest("POST", posturl, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Authorization", "Bearer "+c.RefreshToken)
	r.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not refresh the KSeF token: HTTP %d", response.StatusCode)
	}
	var refreshRes RefreshResponse
	if err := json.NewDecoder(response.Body).Decode(&refreshRes); err != nil {
		return err
	}
	c.SessionToken = refreshRes.AccessToken.Token
	c.SessionTokenValidity = refreshRes.AccessToken.ValidUntil
	return nil
}

func (c *Client) checkKeys() error {
	if c.TokenPublicKey == nil || c.SessionPublicKey == nil {
		return c.getBothKeys()
	}
	return nil
}

func (c *Client) Login() error {
	if err := c.checkKeys(); err != nil {
		return err
	}
	cha, err := c.getChallenge()
	if err != nil {
		return err
	}
	tokenData := []byte(fmt.Sprintf("%s|%d", c.ApiToken, cha.TimestampMs))
	encryptedToken, err := c.encryptWithPKey(tokenData, c.TokenPublicKey)
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
	c.mu.Lock()
	defer c.mu.Unlock()
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
	r.Header.Set("Authorization", "Bearer "+c.SessionToken)
	return c.httpClient.Do(r)
}
