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
	Id int // or uint64? will have to mention capacity
	ExternalId string
	RawJson string
	Status InvoiceStatus
	KsefId string
	KsefErr string
	AttemptCount int // should i just use uint8 here and mention that max attempt count is 255?
	CreatedAt time.Time
	UpdatedAt time.Time
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
	if inv.Status == "" {
		inv.Status = StatusPending
	}
	stmt := "INSERT INTO Invoices(external_id, raw_json, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?) RETURNING id;"
	var invId int
	err := m.DB.QueryRow(stmt, inv.ExternalId, inv.RawJson, inv.Status, inv.CreatedAt, inv.UpdatedAt).Scan(&invId)
	if err != nil {
		return 0, err
	}
	inv.Id = invId
	return invId, nil
}

// to fix: this will crash if ksef_id or ksef_error are NULL
func (m *InvoiceModel) GetInvoice(id int) (*Invoice, error) {
	stmt := "SELECT external_id, raw_json, status, ksef_id, ksef_error, attempt_count, created_at, updated_at FROM Invoices WHERE Id = ?"
	inv := &Invoice {Id: id}
	err := m.DB.QueryRow(stmt, id).Scan(&inv.ExternalId, &inv.RawJson, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}
