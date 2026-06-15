package main

import (
	"context"
	"database/sql"
	"errors"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"net/url"
	"sync"
	"time"
)

func (app *application) startSender(ctx context.Context) {
	app.infoLog.Printf("event=sender_started interval_sec=%d worker_limit=%d batch_size=%d", app.config.PollingInterval, app.config.SenderWorkerLimit, app.config.SenderBatchSize)
	ticker := time.NewTicker(time.Duration(app.config.PollingInterval) * time.Second)
	defer ticker.Stop()
	taskLimit := app.config.SenderWorkerLimit
	taskChan := make(chan struct{}, taskLimit)
	for {
		select {
		case <-ctx.Done():
			app.infoLog.Printf("event=sender_stopped")
			return
		case <-ticker.C:
			app.infoLog.Printf("event=sender_tick")
			app.checkUnknownInvoices(taskChan)
			app.sendInvoice(taskChan)
		}
	}
}

func (app *application) sendInvoice(c chan struct{}) {
	invoices, err := app.invoices.GetPendingInvoicesConc(app.config.SenderBatchSize)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.errorLog.Printf("event=pending_invoices_load_failed error=%q", err.Error())
		}
		return
	}
	app.infoLog.Printf("event=pending_invoices_loaded count=%d", len(invoices))
	inSession, err := app.ksefClient.OpenInSession()
	if err != nil {
		app.errorLog.Printf("event=session_open_failed kind=%q error=%q", "pending", err.Error())
		for _, inv := range invoices {
			if networkCheck(err) {
				if err := app.invoices.UpdatePendingInvoice(inv.Id); err != nil {
					app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
				}
			} else {
				app.handleInvoiceFailure(inv, "Błąd autoryzacji sesji.")
			}
		}
		return
	}
	app.infoLog.Printf("event=session_opened kind=%q session_ref=%q count=%d", "pending", inSession.InSessionRef, len(invoices))
	var wg sync.WaitGroup
	for _, inv := range invoices {
		wg.Add(1)
		go func(invoiceToProcess *models.Invoice) {
			defer wg.Done()
			c <- struct{}{}
			defer func() {
				<-c
			}()
			app.processInvoice(invoiceToProcess, inSession)
		}(inv)
	}
	wg.Wait()
	if err := app.ksefClient.CloseInSession(inSession); err != nil {
		app.errorLog.Printf("event=session_close_failed kind=%q session_ref=%q error=%q", "pending", inSession.InSessionRef, err.Error())
		return
	}
	app.infoLog.Printf("event=session_closed kind=%q session_ref=%q", "pending", inSession.InSessionRef)
}

func (app *application) checkUnknownInvoices(c chan struct{}) {
	invoices, err := app.invoices.GetUnknownInvoicesConc(app.config.SenderBatchSize)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			app.errorLog.Printf("event=unknown_invoices_load_failed error=%q", err.Error())
		}
		return
	}
	app.infoLog.Printf("event=unknown_invoices_loaded count=%d", len(invoices))
	inSession, err := app.ksefClient.OpenInSession()
	if err != nil {
		app.errorLog.Printf("event=session_open_failed kind=%q error=%q", "unknown", err.Error())
		for _, inv := range invoices {
			if networkCheck(err) {
				if err := app.invoices.UpdatePendingInvoice(inv.Id); err != nil {
					app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
				}
			} else {
				app.handleInvoiceFailure(inv, "Błąd autoryzacji sesji.")
			}
		}
		return
	}
	app.infoLog.Printf("event=session_opened kind=%q session_ref=%q count=%d", "unknown", inSession.InSessionRef, len(invoices))
	var wg sync.WaitGroup
	for _, inv := range invoices {
		wg.Add(1)
		go func(inv *models.Invoice) {
			defer wg.Done()
			c <- struct{}{}
			defer func() {
				<-c
			}()
			app.confirmInvoice(inv, inSession.InSessionRef)
		}(inv)
	}
	wg.Wait()
	if err := app.ksefClient.CloseInSession(inSession); err != nil {
		app.errorLog.Printf("event=session_close_failed kind=%q session_ref=%q error=%q", "unknown", inSession.InSessionRef, err.Error())
		return
	}
	app.infoLog.Printf("event=session_closed kind=%q session_ref=%q", "unknown", inSession.InSessionRef)
}

func (app *application) processInvoice(inv *models.Invoice, inSession *ksef.InSession) {
	app.infoLog.Printf("event=invoice_send_started invoice_id=%d external_id=%q session_ref=%q", inv.Id, inv.ExternalId, inSession.InSessionRef)
	ref, err := app.ksefClient.SendInvoice([]byte(inv.RawXml), inSession)
	if err != nil {
		if errors.Is(err, ksef.INVALID_SESSION_ERR) {
			app.errorLog.Printf("event=invoice_send_failed invoice_id=%d external_id=%q reason=%q", inv.Id, inv.ExternalId, "invalid_session")
			if err := app.invoices.UpdatePendingInvoice(inv.Id); err != nil {
				app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
			}
			return
		}
		if networkCheck(err) {
			app.infoLog.Printf("event=invoice_paused_network invoice_id=%d external_id=%q", inv.Id, inv.ExternalId)
			if err := app.invoices.UpdatePendingInvoice(inv.Id); err != nil {
				app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
			}
			return
		}
		app.errorLog.Printf("event=invoice_send_failed invoice_id=%d external_id=%q error=%q", inv.Id, inv.ExternalId, err.Error())
		app.handleInvoiceFailure(inv, err.Error())
		return
	}
	inv.SubmissionReference = &ref
	app.infoLog.Printf("event=invoice_submission_created invoice_id=%d external_id=%q submission_reference=%q", inv.Id, inv.ExternalId, ref)
	app.confirmInvoice(inv, inSession.InSessionRef)
}

func (app *application) confirmInvoice(inv *models.Invoice, inSessionRef string) {
	if inv.SubmissionReference == nil {
		app.errorLog.Printf("event=invoice_confirmation_failed invoice_id=%d external_id=%q reason=%q", inv.Id, inv.ExternalId, "missing_submission_reference")
		app.handleInvoiceFailure(inv, "BRAK DANYCH O WYSYŁCE")
		return
	}
	app.infoLog.Printf("event=invoice_confirmation_started invoice_id=%d external_id=%q submission_reference=%q session_ref=%q", inv.Id, inv.ExternalId, *inv.SubmissionReference, inSessionRef)
	statusRes, err := app.ksefClient.WaitForSendingConfirmation(app.config.Ksef.ConfirmationMaxAttempts, inSessionRef, *inv.SubmissionReference)
	if err != nil {
		if networkCheck(err) {
			app.infoLog.Printf("event=invoice_confirmation_paused_network invoice_id=%d external_id=%q", inv.Id, inv.ExternalId)
			if err := app.invoices.UpdatePendingInvoice(inv.Id); err != nil {
				app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
			}
			return
		} else if errors.Is(err, ksef.UNKNOWN_STATE_ERR) {
			app.infoLog.Printf("event=invoice_marked_unknown invoice_id=%d external_id=%q submission_reference=%q", inv.Id, inv.ExternalId, *inv.SubmissionReference)
			if err := app.invoices.UpdateUnknownInvoice(inv.Id, *inv.SubmissionReference); err != nil {
				app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusUnknown, err.Error())
			}
			return
		}
		app.errorLog.Printf("event=invoice_confirmation_failed invoice_id=%d external_id=%q error=%q", inv.Id, inv.ExternalId, err.Error())
		app.handleInvoiceFailure(inv, err.Error())
		return
	}
	upoXmlData := ""
	if statusRes.UpoDownloadUrl != nil {
		upoBytes, err := app.ksefClient.DownloadUPO(*statusRes.UpoDownloadUrl)
		if err != nil {
			app.errorLog.Printf("event=upo_download_failed invoice_id=%d external_id=%q error=%q", inv.Id, inv.ExternalId, err.Error())
		} else {
			upoXmlData = string(upoBytes)
		}
	}
	if err = app.invoices.UpdateSentInvoice(inv.Id, *statusRes.KsefNumber, upoXmlData, *inv.SubmissionReference); err != nil {
		app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusSent, err.Error())
	} else {
		app.infoLog.Printf("event=invoice_sent invoice_id=%d external_id=%q ksef_id=%q submission_reference=%q", inv.Id, inv.ExternalId, *statusRes.KsefNumber, *inv.SubmissionReference)
	}
}

func (app *application) handleInvoiceFailure(inv *models.Invoice, errorTxt string) {
	if inv.AttemptCount >= app.config.User.Max_retries {
		if err := app.invoices.UpdateFailedInvoice(inv.Id, errorTxt); err != nil {
			app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusFailed, err.Error())
		}
		app.infoLog.Printf("event=invoice_marked_failed invoice_id=%d external_id=%q attempt_count=%d error=%q", inv.Id, inv.ExternalId, inv.AttemptCount, errorTxt)
	} else {
		if err := app.invoices.UpdateRetryInvoice(inv.Id, errorTxt); err != nil {
			app.errorLog.Printf("event=invoice_status_update_failed invoice_id=%d target_status=%q error=%q", inv.Id, models.StatusPending, err.Error())
		}
		app.infoLog.Printf("event=invoice_marked_retry invoice_id=%d external_id=%q next_attempt=%d error=%q", inv.Id, inv.ExternalId, inv.AttemptCount+1, errorTxt)
	}
}

func networkCheck(err error) bool {
	if err == nil {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	return false
}
