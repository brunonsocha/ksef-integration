package main

import (
	"context"
	"database/sql"
	"errors"
	"ksef-integration/internal/models"
	"net/url"
	"sync"
	"time"
)

func (app *application) startSender(ctx context.Context) {
	app.infoLog.Printf("Rozpoczynanie procesu...")
	// the loop does the check every 30 secs
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	taskLimit := 5 // will make it configurable with a recommended range after testing
	taskChan := make(chan struct{}, taskLimit)
	for {
		select {
		case <-ctx.Done():
			app.infoLog.Printf("Aplikacja zostanie zamknięta.")
			if app.ksefClient.InSessionRef != "" {
				if err := app.ksefClient.CloseInSession(); err != nil {
					app.errorLog.Printf("Błąd przy zamykaniu sesji interaktywnej: %v", err)
				} else {
					app.infoLog.Printf("Pomyślnie zamknięto sesję interaktywną.")
				}
			}
			return
		case <-ticker.C:
		invoices, err := app.invoices.GetPendingInvoicesConc(50) // will make it configurable as well
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				app.errorLog.Printf("Błąd przy pozyskiwaniu faktur: %v", err)
			}
			continue
		}
	
		app.infoLog.Printf("Odnaleziono %d faktur. Rozpoczynanie wysyłki.", len(invoices))

		if app.ksefClient.InSessionRef == "" {
			if err := app.ksefClient.OpenInSession(); err != nil {
				app.errorLog.Printf("Błąd przy otwieraniu sesji interaktywnej: %v", err)
				for _, inv := range invoices {
					if networkCheck(err) {
							app.invoices.UpdatePendingInvoice(inv.Id)
					} else {
						app.handleInvoiceFailure(inv, "Błąd autoryzacji sesji.")
					}

				}
				continue
			}
		}

		var wg sync.WaitGroup

		for _, inv := range invoices {
			wg.Add(1)
			go func(invoiceToProcess *models.Invoice) {
				defer wg.Done()
				taskChan <- struct{}{}
				defer func(){
					<- taskChan
				}()
				app.processInvoice(invoiceToProcess)
			}(inv)
		}
		wg.Wait()
		if app.ksefClient.InSessionRef != "" {
			if err := app.ksefClient.CloseInSession(); err != nil {
				app.errorLog.Printf("Błąd przy zamykaniu sesji: %v", err)
			} else {
				app.infoLog.Printf("Zamknięto sesję.")
			}
		}
	}
	}
}

func (app *application) processInvoice(inv *models.Invoice) {
	ref, err := app.ksefClient.SendInvoice([]byte(inv.RawXml))
	if err != nil {
		if networkCheck(err) {
			app.infoLog.Printf("Brak sieci. Wstrzymywanie faktury o ID: %d", inv.Id)
			app.invoices.UpdatePendingInvoice(inv.Id)
			return
		}
		app.errorLog.Printf("Błąd przy wysyłaniu faktury: %v", err)
		app.handleInvoiceFailure(inv, err.Error())
		return
	}
	statusRes, err := app.ksefClient.WaitForSendingConfirmation(15, app.ksefClient.InSessionRef, ref) // config for max attempts????
	if err != nil {
		if networkCheck(err) {
			app.infoLog.Printf("Brak sieci. Wstrzymywanie faktury o ID: %d", inv.Id)
			app.invoices.UpdatePendingInvoice(inv.Id)
			return
		}
		app.errorLog.Printf("Błąd przy potwierdzaniu statusu faktury: %v", err)
		app.handleInvoiceFailure(inv, err.Error())
		return
	}
	upoXmlData := ""
	if statusRes.UpoDownloadUrl != nil {
		upoBytes, err := app.ksefClient.DownloadUPO(*statusRes.UpoDownloadUrl)
		if err != nil {
			app.errorLog.Printf("Błąd przy pobieraniu UPO: %v", err)
		} else {
			upoXmlData = string(upoBytes)
		}
	}
	if err = app.invoices.UpdateSentInvoice(inv.Id, *statusRes.KsefNumber, upoXmlData); err != nil {
		app.errorLog.Printf("Błąd przy zapisie statusu wysłanej faktury: %v", err)
	} else {
		app.infoLog.Printf("Wysłano i zatwierdzono wysyłkę faktury o ID %d", inv.Id)
	}
}

func (app *application) handleInvoiceFailure(inv *models.Invoice, errorTxt string) {
	if inv.AttemptCount >= app.config.User.Max_retries {
		if err := app.invoices.UpdateFailedInvoice(inv.Id, errorTxt); err != nil {
			app.errorLog.Printf("Błąd przy zapisie statusu niewysłanej faktury: %v", err)
		}
		app.infoLog.Printf("Maksymalna ilość prób wysyłki faktury o ID %d", inv.Id)
	} else {
		if err := app.invoices.UpdateRetryInvoice(inv.Id, errorTxt); err != nil {
			app.errorLog.Printf("Błąd przy zapisie statusu faktury do ponownej wysyłki: %v", err)
		}
		app.infoLog.Printf("Ponawianie próby wysłki faktury o ID: %d", inv.Id)
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
