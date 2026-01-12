package ksef

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
)

func (c *Client) readPublicKey() (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(c.PublicKeyPath)
	if err != nil {
		return nil, err
	}
	// we'll ignore the "rest"
	block, _ := pem.Decode(keyBytes)
	keyIfc, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := keyIfc.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Wystąpił błąd w trakcie odczytywania klucza publicznego.")
	}
	return key, nil
}

func (c *Client) encryptToken(cha *ChallengeResponse) (string, error) {
	chaStr := fmt.Sprintf("%s|%d", c.ApiToken, cha.TimestampMs)
	hash := sha256.New()
	key, err := c.readPublicKey()
	if err != nil {
		return "", err
	}
	encryptedToken, err := rsa.EncryptOAEP(hash, rand.Reader, key, []byte(chaStr), nil)
	if err != nil {
		return "", err
	}
	// at this point i'm wondering whether it would be easier to work with byte slices instead of strings
	return base64.StdEncoding.EncodeToString(encryptedToken), nil
}
