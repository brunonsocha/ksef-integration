package ksef

import (
	"crypto/rsa"
	"net/http"
	"time"
)

type Client struct {
	NIP                  string
	ApiToken             string
	TokenPublicKey       *rsa.PublicKey
	SessionPublicKey     *rsa.PublicKey
	ApiURL               string
	AuthRetryDelay       time.Duration
	PollingDelay         time.Duration
	httpClient           *http.Client
	SessionToken         string
	SessionTokenValidity time.Time
	RefreshToken         string
	RefreshTokenValidity time.Time
}

func NewClient(nip, apiToken, apiURL string, httpTimeoutSec, authRetryDelaySec, pollingDelaySec int) *Client {
	return &Client{
		NIP:            nip,
		ApiToken:       apiToken,
		ApiURL:         apiURL,
		AuthRetryDelay: time.Duration(authRetryDelaySec) * time.Second,
		PollingDelay:   time.Duration(pollingDelaySec) * time.Second,
		httpClient:     &http.Client{Timeout: time.Duration(httpTimeoutSec) * time.Second},
	}
}
