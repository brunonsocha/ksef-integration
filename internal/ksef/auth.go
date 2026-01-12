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
	
	if response.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("Nie można było wykonać żądania.")
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("Za dużo żądań.")
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
