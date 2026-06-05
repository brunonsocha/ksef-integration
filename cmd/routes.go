package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"net/http"
	"strconv"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
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
	mux.HandleFunc("GET /ui/invoices", app.getDashboard)
	mux.HandleFunc("GET /ui/invoice", app.getInvoice)
	mux.HandleFunc("GET /ui/invoicetable", app.getDashboardInvoices)
	mux.HandleFunc("DELETE /deleteinvoice", app.deleteInvoice)
	return mux
}

func (app *application) createInvoice(w http.ResponseWriter, r *http.Request) {
	var inv ksef.InvoiceReceived
	// will save json for debugging
	bodyJson, err := io.ReadAll(r.Body)
	if err != nil {
		app.errorLog.Printf("Nie można było odczytać otrzymanego requesta: %v", err)
		http.Error(w, "Niepoprawny request", http.StatusInternalServerError)
		return
	}
	// wouldnt work with decoder
	if err := json.Unmarshal(bodyJson, &inv); err != nil {
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
	if err := app.xsdValidator.ValidateMem(xmlcontent, xsdvalidate.ValidErrDefault); err != nil {
		app.errorLog.Printf("Wystąpił błąd przy walidacji struktury otrzymanej faktury w postaci XML - %v.", err)
		http.Error(w, "Błąd przy walidacji wytworzonego XML.", http.StatusUnprocessableEntity)
		return
	}
	
	dbInv := &models.Invoice{
		ExternalId: inv.InvoiceNumber,
		RawJson: string(bodyJson),
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

func (app *application) getDashboard(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("status")
	// will make it configurable
	pageSize := 50
	pageRaw := r.URL.Query().Get("page")
	data, err := app.dashboardHelper(filter, pageRaw, pageSize) 
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") != "" {
		app.renderer.render(w, "main-page", data)
		return
	}
	app.renderer.render(w, "base", data)
}

func (app *application) getDashboardInvoices(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("status")
	// will make it configurable
	pageSize := 50
	pageRaw := r.URL.Query().Get("page")
	data, err := app.dashboardHelper(filter, pageRaw, pageSize) 
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		return
	}
	app.infoLog.Printf("Załadowano strona %d, filtr %s", data.Page, data.CurrentFilter)
	app.renderer.render(w, "invoice-table", data)
}

func (app *application) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
		return
	}
	invoice, err := app.invoices.GetInvoice(int64(id))
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd przy pozyskiwaniu faktury: %v", err), http.StatusNotFound)
		return
	}
	app.renderer.render(w, "invoice-container", invoice)
}

func (app *application) dashboardHelper(filter, pageRaw string, pageSize int) (dashboardData, error) {
	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		page = 1
	}
	prevPage := page-1
	if prevPage < 1 {
		prevPage = 1
	}
	invoices, err := app.invoices.GetAllInvoices(filter, page, pageSize+1)
	if err != nil {
		return dashboardData{}, err
	}
	more := false
	if len(invoices) > pageSize {
		more = true
		invoices = invoices[:pageSize]
	}
	data := dashboardData{
		Invoices: invoices,
		CurrentFilter: filter,
		Page: page,
		PrevPage: prevPage,
		NextPage: page+1,
		More: more,
	}
	return data, nil
}

func (app *application) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
		return
	}
	if err := app.invoices.DeleteInvoice(int64(id)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o id %d.", id), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
