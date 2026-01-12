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
		time.Sleep(time.Second * 5)
		// logika wysyłki
		ok := true

		if ok {
			app.infoLog.Printf("Pomyślnie wysłano fakturę - ID: %d", inv.Id)
			app.invoices.UpdateSentInvoice(inv.Id, "ksefidmock")
		} else {
			if inv.AttemptCount >= app.config.User.Max_retries {
				app.infoLog.Printf("Osiągnięto maksymalną ilość prób wysyłki faktury o ID: %d", inv.Id)
				app.invoices.UpdateFailedInvoice(inv.Id, "kseferrmock")
			} else {
				app.infoLog.Printf("Ponawianie próby wysyłki faktury o ID: %d", inv.Id)
			}
		}
	}
}
