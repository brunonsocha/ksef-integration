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
		app.errorLog.Printf("Nie można odczytać JSON", err)
	}


	var inv models.Invoice
	err = json.Unmarshal(jsonBody, &inv)
	if err != nil {
		app.errorLog.Printf("Zły format JSON: %v", err)
		http.Error(w, "Zły format JSON", http.StatusBadRequest)
		return
	}
	inv.RawXml = string(jsonBody)
	id, err := app.invoices.InsertInvoice(&inv)
	if err != nil {
		app.errorLog.Printf("Błąd bazy danych: %v", err)
		http.Error(w, "Nie wprowadzono faktury do bazy danych", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": inv.Status})
}
