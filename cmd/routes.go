package main

import (
	"encoding/json"
	"fmt"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"net/http"
	"strconv"
)

type dashboardData struct {
	Invoices []*models.Invoice
	CurrentFilter string
	Page int
	PrevPage int
	NextPage int
	More bool
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", app.createInvoice)
	fs := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fs))
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /ui/invoices", app.getDashboardInvoices)
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

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui/invoices", http.StatusSeeOther)	
}

func (app *application) getDashboardInvoices(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("status")
	// will make it configurable
	pageSize := 50
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	prevPage := page-1
	if prevPage < 1 {
		prevPage = 1
	}
	invoices, err := app.invoices.GetAllInvoices(filter, page, pageSize+1)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		return
	}
	more := false
	if len(invoices) > pageSize {
		more = true
		invoices = invoices[:50]
	}
	data := dashboardData{
		Invoices: invoices,
		CurrentFilter: filter,
		Page: page,
		PrevPage: prevPage,
		NextPage: page+1,
		More: more,
	}
	app.infoLog.Printf("Załadowano strona %d, filtr %s", data.Page, data.CurrentFilter)
	if r.Header.Get("HX-Request") != "" {
		app.renderer.render(w, "main-page", data)
		return
	}
	app.renderer.render(w, "base", data)
}
