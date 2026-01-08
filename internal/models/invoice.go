package models

import (
	"database/sql"
	"time"
)

type InvoiceStatus string

const (
	StatusPending InvoiceStatus = "PENDING"
	StatusSent InvoiceStatus = "SENT"
	StatusRetry InvoiceStatus = "RETRY"
	StatusFailed InvoiceStatus = "FAILED")

type Invoice struct {
	Id int `json:"id"` // or uint64? will have to mention capacity
	ExternalId string `json:"external_id"`
	RawJson string `json:"-"`
	Status InvoiceStatus `json:"status"`
	KsefId *string `json:"ksef_id"` // allows nil
	KsefErr *string `json:"ksef_error"` // allows nil
	AttemptCount int `json:"attempt_count"`// should i just use uint8 here and mention that max attempt count is 255?
	CreatedAt time.Time `json:"created_at`
	UpdatedAt time.Time `json:"updated_at`
}

type InvoiceModel struct {
	DB *sql.DB
}

func (m *InvoiceModel) InsertInvoice(inv *Invoice) (int, error) {
	// handling this here instead of the http handler due to the planned cli integration
	now := time.Now()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}
	inv.UpdatedAt = now
	inv.KsefErr = nil
	inv.KsefId = nil
	inv.AttemptCount = 0
	inv.Status = StatusPending
	stmt := "INSERT INTO Invoices(external_id, raw_json, status) VALUES (?, ?, ?) RETURNING id, created_at, updated_at;"
	err := m.DB.QueryRow(stmt, inv.ExternalId, inv.RawJson, inv.Status).Scan(&inv.Id, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return inv.Id, nil
}

func (m *InvoiceModel) GetInvoice(id int) (*Invoice, error) {
	stmt := "SELECT external_id, raw_json, status, ksef_id, ksef_error, attempt_count, created_at, updated_at FROM Invoices WHERE Id = ?"
	inv := &Invoice {Id: id}
	err := m.DB.QueryRow(stmt, id).Scan(&inv.ExternalId, &inv.RawJson, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}
