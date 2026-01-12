package ksef

import (
	"net/http"
	"time"
)

type Client struct {
	NIP string
	ApiToken string
	PublicKeyPath string
	ApiURL string
	httpClient *http.Client
	SessionToken string
	RefreshToken string
}

func NewClient(nip, apiToken, publicKeyPath, apiURL string) *Client {
	return &Client{
		NIP: nip,
		ApiToken: apiToken,
		PublicKeyPath: publicKeyPath,
		ApiURL: apiURL,
		httpClient: &http.Client{Timeout: time.Second * 100},
	}
}
