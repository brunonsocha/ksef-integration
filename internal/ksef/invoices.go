package ksef
/*
import (
	"net/http"
	"time"
)


type InteractiveSessionPayload struct {
	FormCode struct {
		SystemCode string `json:"systemCode"`
		SchemaVersion string `json:"schemaVersion"`
		Value string `json:"value"`
	} `json:"formCode"`
	Encryption struct {
		EncryptedSymmetricKey string `json:"encryptedSymmetricKey"`
		InitializationVector string `json:"initializationVector"`
	} `json:"encryption"`
}

type InteractiveSessionResponse struct {
	ReferenceNumber string `json:"referenceNumber"`
	ValidUntil time.Time `json:"validUntil"`
}


func (c *Client) OpenInteractiveSession(systemCode, schemaVersion, value string) (*InteractiveSessionResponse, error) {
	posturl := c.ApiURL + "/sessions/online"
	payload := InteractiveSessionPayload{
		FormCode: struct {
			SystemCode string `json:"systemCode"`
			SchemaVersion string `json:"schemaVersion"`
			Value string `json:"value"`
		}{
			SystemCode: systemCode,
			SchemaVersion: schemaVersion,
			Value: value,
		},
		Encryption: struct {
			EncryptedSymmetricKey string `json:"encryptedSymmetricKey"`
			InitializationVector string `json:"initializationVector"`
		}{
			EncryptedSymmetricKey: ,
			InitializationVector: ,
		}
	}
}
*/

