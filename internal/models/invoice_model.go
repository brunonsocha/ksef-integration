package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type InvoiceStatus string

const (
	StatusPending    InvoiceStatus = "PENDING"
	StatusProcessing InvoiceStatus = "PROCESSING"
	StatusSent       InvoiceStatus = "SENT"
	StatusFailed     InvoiceStatus = "FAILED"
	StatusUnknown    InvoiceStatus = "UNKNOWN"
)

type Invoice struct {
	Id                  int64         `json:"id"`
	ExternalId          string        `json:"external_id"`
	RawJson             string        `json:"-"`
	RawXml              string        `json:"-"`
	Status              InvoiceStatus `json:"status"`
	KsefId              *string       `json:"ksef_id"`
	KsefErr             *string       `json:"ksef_error"`
	UpoXml              *string       `json:"upo_xml"`
	AttemptCount        int           `json:"attempt_count"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
	SessionReference    *string       `json:"session_reference"`
	SubmissionReference *string       `json:"submission_reference"`
	CallbackURL         *string       `json:"callback_url"`
	WebhookDelivered    bool          `json:"webhook_delivered"`
	WebhookAttemptCount int           `json:"webhook_attempt_count"`
	WebhookErr          *string       `json:"webhook_error"`
}

type InvoiceModel struct {
	DB *sql.DB
}

func (m *InvoiceModel) InsertInvoice(inv *Invoice) (int64, error) {
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now().UTC()
	}
	inv.UpdatedAt = time.Now().UTC()
	inv.KsefErr = nil
	inv.KsefId = nil
	inv.AttemptCount = 0
	inv.Status = StatusPending
	stmt := "INSERT INTO Invoices(external_id, raw_json, raw_xml, status, callback_url) VALUES (?, ?, ?, ?, ?) RETURNING id, created_at, updated_at;"
	if err := m.DB.QueryRow(stmt, inv.ExternalId, inv.RawJson, inv.RawXml, inv.Status, inv.CallbackURL).Scan(&inv.Id, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return 0, err
	}
	return inv.Id, nil
}

func (m *InvoiceModel) GetInvoice(id int64) (*Invoice, error) {
	stmt := "SELECT id, external_id, raw_json, raw_xml, status, ksef_id, ksef_error, upo_xml, attempt_count, created_at, updated_at, session_reference, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices WHERE id = ? LIMIT 1"
	inv := &Invoice{}
	if err := m.DB.QueryRow(stmt, id).Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.UpoXml, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SessionReference, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
		return nil, err
	}
	return inv, nil
}

func (m *InvoiceModel) GetInvoiceExternalId(externalId string) (*Invoice, error) {
	// even though this is used only in the getInvoiceStatusExternalId for now
	// i will make it return a full invoice struct for consistency with standard GetInvoice
	stmt := "SELECT id, external_id, raw_json, raw_xml, status, ksef_id, ksef_error, upo_xml, attempt_count, created_at, updated_at, session_reference, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices WHERE external_id = ? LIMIT 1"
	inv := &Invoice{}
	if err := m.DB.QueryRow(stmt, externalId).Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.UpoXml, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SessionReference, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
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
	stmt := "SELECT id, external_id, raw_json, raw_xml, status, ksef_id, ksef_error, upo_xml, attempt_count, created_at, updated_at, session_reference, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices WHERE status = ? ORDER BY created_at ASC LIMIT ?"
	rows, err := transaction.Query(stmt, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []*Invoice
	var idsToUpdate []int64
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.UpoXml, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SessionReference, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
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
		_, err := transaction.Exec("UPDATE Invoices SET status = ?, updated_at = ? WHERE id = ?", StatusProcessing, time.Now().UTC(), id)
		if err != nil {
			return nil, err
		}
	}
	if err = transaction.Commit(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (m *InvoiceModel) GetUnknownInvoicesConc(limit int) ([]*Invoice, error) {
	transaction, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer transaction.Rollback()
	stmt := "SELECT id, external_id, raw_json, raw_xml, status, ksef_id, ksef_error, upo_xml, attempt_count, created_at, updated_at, session_reference, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices WHERE status = ? ORDER BY created_at ASC LIMIT ?"
	rows, err := transaction.Query(stmt, StatusUnknown, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []*Invoice
	var idsToUpdate []int64
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.RawJson, &inv.RawXml, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.UpoXml, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SessionReference, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
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
		_, err := transaction.Exec("UPDATE Invoices SET status = ?, updated_at = ? WHERE id = ?", StatusProcessing, time.Now().UTC(), id)
		if err != nil {
			return nil, err
		}
	}
	if err = transaction.Commit(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (m *InvoiceModel) UpdateSentInvoice(id int64, ksefId, upo_xml, submissionReference string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_id = ?, upo_xml = ?, ksef_error = NULL, updated_at = ?, submission_reference = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusSent, ksefId, upo_xml, time.Now().UTC(), submissionReference, id)
	return err
}

func (m *InvoiceModel) UpdateRetryInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET status = ?, attempt_count = attempt_count + 1, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusPending, ksefErr, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) UpdateFailedInvoice(id int64, ksefErr string) error {
	stmt := "UPDATE Invoices SET status = ?, ksef_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusFailed, ksefErr, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) UpdatePendingInvoice(id int64) error {
	stmt := "UPDATE Invoices SET status = ?, updated_at = ?  WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusPending, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) UpdateUnknownInvoice(id int64, sessionReference, submissionReference string) error {
	stmt := "UPDATE Invoices SET status = ?, session_reference = ?, submission_reference = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusUnknown, sessionReference, submissionReference, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) RestoreUnknownInvoice(id int64) error {
	stmt := "UPDATE Invoices SET status = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, StatusUnknown, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) RecoverProcessingInvoices() error {
	stmt := "UPDATE Invoices SET status = CASE WHEN session_reference IS NOT NULL AND submission_reference IS NOT NULL THEN ? ELSE ? END, updated_at = ? WHERE status = ?"
	_, err := m.DB.Exec(stmt, StatusUnknown, StatusPending, time.Now().UTC(), StatusProcessing)
	return err
}

func (m *InvoiceModel) DeleteInvoice(id int64) error {
	stmt := "DELETE FROM Invoices WHERE id = ? AND status = ?"
	rows, err := m.DB.Exec(stmt, id, StatusFailed)
	if err != nil {
		return err
	}
	num, err := rows.RowsAffected()
	if num == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (m *InvoiceModel) DeleteInvoiceExternalId(externalId string) error {
	stmt := "DELETE FROM Invoices WHERE external_id = ? AND status = ?"
	rows, err := m.DB.Exec(stmt, externalId, StatusFailed)
	if err != nil {
		return err
	}
	num, err := rows.RowsAffected()
	if num == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (m *InvoiceModel) GetAllInvoices(filter, query string, page, limit int) ([]*Invoice, error) {
	stmt := "SELECT id, external_id, status, ksef_id, ksef_error, attempt_count, created_at, updated_at, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices "
	var args []any
	var conditions []string
	status := InvoiceStatus(filter)
	if filter != "" && filter != "all" {
		switch status {
		case StatusFailed, StatusPending, StatusProcessing, StatusSent, StatusUnknown:
			conditions = append(conditions, "status = ?")
			args = append(args, status)
		default:
			return nil, fmt.Errorf("Niepoprawny filtr: %s.", filter)
		}
	}
	if query != "" {
		conditions = append(conditions, "external_id LIKE ?")
		args = append(args, "%"+query+"%")
	}
	if len(conditions) > 0 {
		stmt += "WHERE " + strings.Join(conditions, " AND ") + " "
	}
	pageOffset := (page - 1) * limit
	limit++
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
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (m *InvoiceModel) GetInvoicesToWebhook(webhook_attempt_limit, limit int) ([]*Invoice, error) {
	stmt := "SELECT id, external_id, status, ksef_id, ksef_error, attempt_count, created_at, updated_at, submission_reference, callback_url, webhook_delivered, webhook_attempt_count, webhook_error FROM Invoices WHERE status IN ('FAILED', 'SENT') AND webhook_delivered = 0 AND callback_url IS NOT NULL AND webhook_attempt_count < ? ORDER BY updated_at ASC LIMIT ?"
	rows, err := m.DB.Query(stmt, webhook_attempt_limit, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var invoices []*Invoice
	for rows.Next() {
		inv := &Invoice{}
		if err := rows.Scan(&inv.Id, &inv.ExternalId, &inv.Status, &inv.KsefId, &inv.KsefErr, &inv.AttemptCount, &inv.CreatedAt, &inv.UpdatedAt, &inv.SubmissionReference, &inv.CallbackURL, &inv.WebhookDelivered, &inv.WebhookAttemptCount, &inv.WebhookErr); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(invoices) == 0 {
		return nil, sql.ErrNoRows
	}
	return invoices, nil
}

func (m *InvoiceModel) UpdateWebhookDelivered(id int64) error {
	stmt := "UPDATE Invoices SET webhook_delivered = ?, webhook_error = ?, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, true, nil, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) UpdateWebhookFailed(id int64, errTxt string) error {
	stmt := "UPDATE Invoices SET webhook_error = ?, webhook_attempt_count = webhook_attempt_count + 1, updated_at = ? WHERE id = ?"
	_, err := m.DB.Exec(stmt, errTxt, time.Now().UTC(), id)
	return err
}

func (m *InvoiceModel) ResetWebhookAttemptCount(id int64) error {
	stmt := "UPDATE Invoices SET webhook_attempt_count = ?, updated_at = ? WHERE id = ?"
	rows, err := m.DB.Exec(stmt, 0, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	num, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if num == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *InvoiceModel) ReplaceInvoice(externalId string, rawJson, rawXml string, callbackUrl *string) (int64, error) {
	var id int64
	stmt := "UPDATE Invoices SET status = ?, ksef_error = ?, attempt_count = ?, updated_at = ?, session_reference = ?, submission_reference = ?, callback_url = ?, webhook_delivered = ?, webhook_attempt_count = ?, webhook_error = ?, raw_json = ?, raw_xml = ? WHERE external_id = ? AND status = ? RETURNING id"
	if err := m.DB.QueryRow(stmt, StatusPending, nil, 0, time.Now().UTC(), nil, nil, callbackUrl, false, 0, nil, rawJson, rawXml, externalId, StatusFailed).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
