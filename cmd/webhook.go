package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"ksef-integration/internal/models"
	"net/http"
	"time"
)

type webhookPayload struct {
	Event               string    `json:"event"`
	InvoiceId           int64     `json:"invoice_id"`
	ExternalId          string    `json:"external_id"`
	Status              string    `json:"status"` //couldnt it be an InvoiceStatus instead?
	KsefReferenceNumber *string   `json:"ksef_reference_number"`
	SubmissionReference *string   `json:"submission_reference"`
	ErrorMessage        *string   `json:"error_message"`
	Timestamp           time.Time `json:"timestamp"`
}

func (app *application) notifyWebhook(inv *models.Invoice) error {
	if inv.CallbackURL == nil {
		return errors.New("event=webhook_delivery_skipped reason=missing_callback_url")
	}
	callback := inv.CallbackURL
	var eventMessage string
	switch inv.Status {
	case models.StatusFailed:
		eventMessage = "invoice.failed"
	case models.StatusSent:
		eventMessage = "invoice.accepted"
	default:
		return fmt.Errorf("event=webhook_delivery_skipped reason=unexpected_invoice_status")
	}
	payload := webhookPayload{
		Event:               eventMessage,
		InvoiceId:           inv.Id,
		ExternalId:          inv.ExternalId,
		Status:              string(inv.Status),
		KsefReferenceNumber: inv.KsefId,
		SubmissionReference: inv.SubmissionReference,
		ErrorMessage:        inv.KsefErr,
		Timestamp:           time.Now().UTC(),
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return err
	}
	r, err := http.NewRequest("POST", *callback, &buf)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	res, err := app.httpClient.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode > 299 || res.StatusCode < 200 {
		return fmt.Errorf("event=webhook_delivery_rejected status_code=%d", res.StatusCode)
	}
	return nil
}
