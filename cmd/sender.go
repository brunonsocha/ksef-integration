package main

import (
	"database/sql"
	"errors"
	"time"
)

func (app *application) startSender() {
	app.infoLog.Printf("Rozpoczynanie procesu...")
	for {
		inv, err := app.invoices.GetPendingInvoice()
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				app.infoLog.Printf("Brak faktur do wysyłki.")
				time.Sleep(time.Minute)
				continue
			}
			app.errorLog.Printf("Wystąpił błąd: %v", err)
			time.Sleep(time.Minute)
			continue
		}
		app.infoLog.Printf("Odnaleziono fakturę - ID: %d", inv.Id)
		ok := false

		if ok {
			err := app.invoices.UpdateSentInvoice(inv.Id, "ksefidmock")
			if err != nil {
				app.errorLog.Printf("Nie można znaleźć faktury: %v", err)
				continue
			}
			app.infoLog.Printf("Pomyślnie wysłano fakturę - ID: %d", inv.Id)
		} else {
			if inv.AttemptCount >= app.config.User.Max_retries {
				if err := app.invoices.UpdateFailedInvoice(inv.Id, "kseferrmock"); err != nil {
					app.errorLog.Printf("Nie można znaleźć faktury: %v", err)
					continue
				}
				app.infoLog.Printf("Osiągnięto maksymalną ilość prób wysyłki faktury o ID: %d", inv.Id)
			} else {
				app.infoLog.Printf("Ponawianie próby wysyłki faktury o ID: %d", inv.Id)
				if err := app.invoices.UpdateRetryInvoice(inv.Id, "kseferrmock"); err != nil {
					app.errorLog.Printf("Nie można znaleźć faktury: %v", err)
				}
			}
		}
		time.Sleep(time.Minute)
	}
}
