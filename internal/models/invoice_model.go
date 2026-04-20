package models

import (
	"database/sql"
	"fmt"
	"time"
)

type InvoiceStatus string

const (
	StatusPending InvoiceStatus = "PENDING"
	StatusProcessing InvoiceStatus = "PROCESSING"
	StatusSent InvoiceStatus = "SENT"
	StatusFailed InvoiceStatus = "FAILED")

type Invoice struct {
	Id int64 `json:"id"` 
	ExternalId string `json:"external_id"`
	RawXml string `json:"-"`
	Status InvoiceStatus `json:"status"`
	KsefId *string `json:"ksef_id"` 
	KsefErr *string `json:"ksef_error"` 
	UpoXml *string `json:"upo_xml"`
	AttemptCount int `json:"attempt_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type InvoiceModel struct {
	DB *sql.DB
}

func (m *InvoiceModel) InsertInvoice(inv *Invoice) (int64, error) {
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	inv.UpdatedAt = time.Now()
	inv.KsefErr = nil
	inv.KsefId = nil
	inv.AttemptCount = 0
	inv.Status = StatusPending
	stmt := "INSERT INTO Invoices(external_id, raw_xml, status) VALUES (?, ?, ?) RETURNING id, created_at, updated_at;"
	if err := m.DB.QueryRow(stmt, inv.ExternalId, inv.RawXml, inv.Status).Scan(&inv.Id, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return 0, err
	}
	return inv.Id, nil
}

func (m *InvoiceModel) GetInvoice(id int64) (*Invoice, error) {
	stmt := "SELECT * FROM Invoices WHERE Id = ? LIMIT 1"
	inv := &Invoice{}
	if err := m.DB.QueryRow(stmt, id).Scan(&inv.Id, &inv.ExternalId, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return nil, err
	}
	return inv, nil
}

func (m *InvoiceModel) GetPendingInvoicesConc(limit int) ([]*Invoice, error) {
	transaction, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback()
	stmt := "SELECT id, external_id, raw_xml, status, ksef_id, ksef_error, upo_xml, attempt_count, created_at, updated_at FROM Invoices WHERE status = ? ORDER BY created_at ASC LIMIT ?"
	rows, err := transaction.Query(stmt, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []*Invoice
	var idsToUpdate []int64
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.UpoXml, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
		idsToUpdate = append(idsToUpdate, inv.Id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(invoices) == 0 {
		return nil, sql.ErrNoRows
	}
	for _, id := range idsToUpdate {
		_, err := transaction.Exec("UPDATE Invoices SET status = ?, updated_at = ? WHERE id = ?", StatusProcessing, time.Now(), id)
		if err != nil {
			return nil, err
		}
	}
	if err = transaction.Commit(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (m *InvoiceModel) UpdateSentInvoice(id int64, ksefId, upo_xml string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_id = ?, upo_xml = ?, ksef_error = NULL, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusSent, ksefId, upo_xml, time.Now(), id)
	return err
}

func (m *InvoiceModel) UpdateRetryInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET status = ?, attempt_count = attempt_count + 1, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusPending, ksefErr, time.Now(), id)
	return err
}

func (m *InvoiceModel) UpdateFailedInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusFailed, ksefErr, time.Now(), id)
	return err
}

func (m *InvoiceModel) UpdatePendingInvoice(id int64) error {
	stmt := "UPDATE Invoices SET status = ?, updated_at = ?  WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusPending, time.Now())
	return err
}

// hardcoded limit 50 - might make it configurable
func (m *InvoiceModel) GetAllInvoices(filter string, page, limit int) ([]*Invoice, error) {
	stmt := "SELECT id, external_id, status, ksef_id, ksef_error, attempt_count, created_at, updated_at FROM Invoices " 
	var args []any
	status := InvoiceStatus(filter)
	if filter != ""  && filter != "all" {
		stmt += "WHERE status = ? "
		switch status {
		case StatusFailed, StatusPending, StatusProcessing, StatusSent:
			args = append(args, status)
		default:
			return nil, fmt.Errorf("Niepoprawny filtr: %s.", filter)
		}
	} 
	pageOffset := (page-1) * (limit-1)
	args = append(args, limit, pageOffset)	
	stmt += "ORDER BY created_at DESC LIMIT ? OFFSET ?"
	rows, err := m.DB.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []*Invoice
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoices, nil
}
