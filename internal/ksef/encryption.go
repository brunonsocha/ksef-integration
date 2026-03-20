package ksef

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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
	keyIfc, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := keyIfc.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Wystąpił błąd w trakcie odczytywania klucza publicznego.")
	}
	return key, nil
}

func (c *Client) encryptWithPKey(aesKey []byte) ([]byte, error) {
	hash := sha256.New()
	key, err := c.readPublicKey()
	if err != nil {
		return nil, err
	}
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

func (c *Client) encryptCBC(text []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.InSessionAESKey)
	if err != nil {
		return nil, err
	}
	paddedData := padPKCS(text, aes.BlockSize)
	ciphertxt := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, c.InSessionInitializationVector)
	mode.CryptBlocks(ciphertxt, paddedData)
	return ciphertxt, nil
}

func hashSHA256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}
