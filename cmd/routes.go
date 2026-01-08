package main

import (
	"encoding/json"
	"io"
	"ksef-integration/internal/models"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", app.createInvoice)
	return mux
}

func (app *application) createInvoice(w http.ResponseWriter, r *http.Request) {
	jsonBody, err := io.ReadAll(r.Body)
	if err != nil {
		app.errorLog.Printf("Can't read the JSON body: %v", err)
	}


	var inv models.Invoice
	err = json.Unmarshal(jsonBody, &inv)
	if err != nil {
		app.errorLog.Printf("Bad JSON format: %v", err)
		http.Error(w, "Bad JSON format", http.StatusBadRequest)
		return
	}
	inv.RawJson = string(jsonBody)
	id, err := app.invoices.InsertInvoice(&inv)
	if err != nil {
		app.errorLog.Printf("Database fail: %v", err)
		http.Error(w, "Couldn't insert the invoice into the database", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": inv.Status})
}
