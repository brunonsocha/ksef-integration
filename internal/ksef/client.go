package ksef

import (
	"crypto/rsa"
	"net/http"
	"time"
)

type Client struct {
	NIP string
	ApiToken string
	TokenPublicKey *rsa.PublicKey
	SessionPublicKey *rsa.PublicKey
	ApiURL string
	httpClient *http.Client
	SessionToken string
	SessionTokenValidity time.Time
	RefreshToken string
	RefreshTokenValidity time.Time
}

func NewClient(nip, apiToken, publicKeyPath, apiURL string) *Client {
	return &Client{
		NIP: nip,
		ApiToken: apiToken,
		ApiURL: apiURL,
		httpClient: &http.Client{Timeout: time.Second * 100},
	}
}
