package main

import (
	"encoding/json"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", app.createInvoice)
	return mux
}

func (app *application) createInvoice(w http.ResponseWriter, r *http.Request) {
	var inv ksef.InvoiceReceived
	if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
		app.errorLog.Printf("Zły format JSON: %v", err)
		http.Error(w, "Zły format JSON", http.StatusBadRequest)
		return
	}
	if err := inv.ValidateInvoiceReceived(); err != nil {
		app.errorLog.Printf("Niepoprawne dane w otrzymanej fakturze: %v", err)
		http.Error(w, "Niepoprawne dane w otrzymanej fakturze", http.StatusUnprocessableEntity)
		return
	}
	xmlcontent, err := ksef.TransformToXML(&inv)
	if err != nil {
		app.errorLog.Printf("Wystąpił błąd przy transformacji w XML: %v", err)
		http.Error(w, "Błąd przy XML", http.StatusInternalServerError)
		return
	}
	
	dbInv := &models.Invoice{
		ExternalId: inv.InvoiceNumber,
		RawXml: string(xmlcontent),
	}
	id, err := app.invoices.InsertInvoice(dbInv)
	if err != nil {
		app.errorLog.Printf("Błąd bazy danych: %v", err)
		http.Error(w, "Nie wprowadzono faktury do bazy danych", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": dbInv.Status})
}
