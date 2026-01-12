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
	err = json.Unmarshal(jsonBody, &cha)
	if err != nil {
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
	err = json.Unmarshal(responseJson, &authRes)
	if err != nil {
		return nil, fmt.Errorf("Nie można było odczytać odpowiedzi.")
	}
	return &authRes, nil
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
	c.SessionToken = authRes.AuthenticationToken.Token
	return nil
}
