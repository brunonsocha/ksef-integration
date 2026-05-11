package ksef

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type KeysResponse struct {
	Certificate string `json:"certificate"`
	Usage []string `json:"usage"`
}

func (c *Client) getBothKeys() error {
	fullUrl := fmt.Sprintf("%s/security/public-key-certificates", c.ApiURL)
	res, err := http.Get(fullUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("KSeF nie zwrócił klucza - kod statusu: %v", res.StatusCode)
	}
	var keyRes []KeysResponse
	if err := json.NewDecoder(res.Body).Decode(&keyRes); err != nil {
		return err
	}
	for _, key := range keyRes {
		certificateBytes, err := base64.StdEncoding.DecodeString(key.Certificate)
		if err != nil {
			continue
		}
		certificate, err := x509.ParseCertificate(certificateBytes)
		if err != nil {
			continue
		}
		pub := certificate.PublicKey.(*rsa.PublicKey)
		for _, usage := range key.Usage {
			if usage == "KsefTokenEncryption" {
				c.TokenPublicKey = pub
			} else if usage == "SymmetricKeyEncryption" {
				c.SessionPublicKey = pub
			}
		}
	}
	if c.TokenPublicKey == nil || c.SessionPublicKey == nil {
		return fmt.Errorf("Błąd przy pozyskiwaniu kluczy publicznych od KSeF.")
	}
	return nil
}

func (c *Client) encryptWithPKey(aesKey []byte, key *rsa.PublicKey) ([]byte, error) {
	hash := sha256.New()
	encryptedData, err := rsa.EncryptOAEP(hash, rand.Reader, key, aesKey, nil)
	if err != nil {
		return nil, err
	}
	return encryptedData, nil
}

//multiple of 16s
func padPKCS(data []byte, blockSize int) []byte {
	paddingLen := blockSize - (len(data) % blockSize)
	padding := bytes.Repeat([]byte{byte(paddingLen)}, paddingLen)
	return append(data, padding...)
}

func (c *Client) encryptCBC(text []byte, inSession *InSession) ([]byte, error) {
	block, err := aes.NewCipher(inSession.InSessionAESKey)
	if err != nil {
		return nil, err
	}
	paddedData := padPKCS(text, aes.BlockSize)
	ciphertxt := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, inSession.InSessionInitializationVector)
	mode.CryptBlocks(ciphertxt, paddedData)
	return ciphertxt, nil
}

func hashSHA256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}
