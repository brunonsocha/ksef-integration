package main

import (
	"encoding/json"
	"ksef-integration/internal/models"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", app.createInvoice)
	return mux
}

func (app *application) createInvoice(w http.ResponseWriter, r *http.Request) {
	var inv models.Invoice
	err := json.NewDecoder(r.Body).Decode(&inv)
	if err != nil {
		app.errorLog.Printf("Bad JSON format", "error", err)
		http.Error(w, "Bad JSON format", http.StatusBadRequest)
		return
	}
	inv.Status = models.StatusPending
	id, err := app.invoices.InsertInvoice(&inv)
	if err != nil {
		app.errorLog.Printf("Database fail", "error", err)
		http.Error(w, "Couldn't insert the invoice into the database", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": inv.Status})
}
