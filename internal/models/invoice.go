package models

import (
	"database/sql"
	"time"
)

type InvoiceStatus string

const (
	StatusPending InvoiceStatus = "PENDING"
	StatusSent InvoiceStatus = "SENT"
	StatusFailed InvoiceStatus = "FAILED")

type Invoice struct {
	Id int64 `json:"id"` // changed it to int64 - sqlite uses int64 for ids with autoincrement
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

func (m *InvoiceModel) InsertInvoice(inv *Invoice) (int64, error) {
	// handling this here instead of the http handler due to the planned cli integration
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	inv.UpdatedAt = time.Now()
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

func (m *InvoiceModel) GetInvoice(id int64) (*Invoice, error) {
	// can't think of an edgecase in which replacing the given id with the scanned one could cause problems.
	// added a limit 1 t obe consistent
	stmt := "SELECT * FROM Invoices WHERE Id = ? LIMIT 1"
	inv := &Invoice{}
	err := m.DB.QueryRow(stmt, id).Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (m *InvoiceModel) GetPendingInvoice() (*Invoice, error) {
	stmt := "SELECT * FROM Invoices WHERE status = ? ORDER BY created_at ASC LIMIT 1"
	inv := &Invoice{}
	err := m.DB.QueryRow(stmt, StatusPending).Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err // have to make sure the process checks whether this is ErrNoRows, if yes, it shouldn't crash - just stop working
	}
	return inv, nil
}

// could just create one singular function that accepts id, ksefErr and ksefId - pass string pointers - and the result will be the same, if the pointer is nil, the update will apply a NULL in the database

func (m *InvoiceModel) UpdateSentInvoice(id int64, ksefId string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_id = ?, ksef_error = NULL, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusSent, ksefId, time.Now(), id)
	return err
}

// don't know whether there's a point in having a seperate status for retry - for frontend purposes? could just check whether attempt_count > 0

func (m *InvoiceModel) UpdateRetryInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET attempt_count = attempt_count + 1, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, ksefErr, time.Now(), id)
	return err
}

func (m *InvoiceModel) UpdateFailedInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusFailed, ksefErr, time.Now(), id)
	return err
}
