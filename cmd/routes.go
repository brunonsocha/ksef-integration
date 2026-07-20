package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"net/http"
	"net/url"
	"strconv"
	"time"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

type dashboardData struct {
	Invoices      []*models.Invoice
	CurrentFilter string
	Page          int
	PrevPage      int
	NextPage      int
	More          bool
	Query         string
	EscapedQuery  string
	WebhookMaxRetries int
}

type invoiceStatusRes struct {
	Id int64 `json:"id"`
	ExternalId string `json:"external_id"`
	Status string `json:"status"`
	KsefId *string `json:"ksef_id"`
	KsefErr *string `json:"ksef_error"`
	SubmissionReference *string `json:"submission_reference"`
	AttemptCount int `json:"attempt_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	WebhookDelivered bool `json:"webhook_delivered"`
	WebhookErr *string `json:"webhook_error"`
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", app.createInvoice)
	fs := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fs))
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /ui/invoices", app.getDashboard)
	mux.HandleFunc("GET /ui/invoice", app.getDashboardInvoice)
	mux.HandleFunc("GET /ui/invoicetable", app.getDashboardInvoices)
	mux.HandleFunc("DELETE /deleteinvoice", app.deleteInvoice)
	mux.HandleFunc("DELETE /ui/deleteinvoice", app.deleteDashboardInvoice)
	mux.HandleFunc("GET /health/live", app.getHealthLive)
	mux.HandleFunc("GET /health/ready", app.getHealthReady)
	mux.HandleFunc("GET /invoice", app.getInvoiceStatus)
	mux.HandleFunc("POST /ui/retrywebhook", app.postDashboardRetryWebhook)
	return mux
}

func (app *application) createInvoice(w http.ResponseWriter, r *http.Request) {
	var inv ksef.InvoiceReceived
	action := "replaced"
	bodyJson, err := io.ReadAll(r.Body)
	if err != nil {
		app.errorLog.Printf("event=invoice_request_read_failed error=%q", err.Error())
		http.Error(w, "Niepoprawny request", http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(bodyJson, &inv); err != nil {
		app.errorLog.Printf("event=invoice_json_invalid error=%q", err.Error())
		http.Error(w, "Zły format JSON", http.StatusBadRequest)
		return
	}

	app.infoLog.Printf("event=invoice_received external_id=%q", inv.InvoiceNumber)
	if err := inv.ValidateInvoiceReceived(); err != nil {
		app.errorLog.Printf("event=invoice_validation_failed external_id=%q error=%q", inv.InvoiceNumber, err.Error())
		http.Error(w, "Niepoprawne dane w otrzymanej fakturze", http.StatusUnprocessableEntity)
		return
	}
	xmlcontent, err := ksef.TransformToXML(&inv)
	if err != nil {
		app.errorLog.Printf("event=invoice_xml_transform_failed external_id=%q error=%q", inv.InvoiceNumber, err.Error())
		http.Error(w, "Błąd przy XML", http.StatusInternalServerError)
		return
	}
	if err := app.xsdValidator.ValidateMem(xmlcontent, xsdvalidate.ValidErrDefault); err != nil {
		app.errorLog.Printf("event=invoice_xsd_validation_failed external_id=%q error=%q", inv.InvoiceNumber, err.Error())
		http.Error(w, "Błąd przy walidacji wytworzonego XML.", http.StatusUnprocessableEntity)
		return
	}
	var id int64
	dbInv := &models.Invoice{
		ExternalId:  inv.InvoiceNumber,
		RawJson:     string(bodyJson),
		RawXml:      string(xmlcontent),
		CallbackURL: inv.CallbackURL,
	}
	id, err = app.invoices.ReplaceInvoice(dbInv.ExternalId, dbInv.RawJson, dbInv.RawXml, dbInv.CallbackURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			id, err = app.invoices.InsertInvoice(dbInv)
			action = "created"
			if err != nil {
				app.errorLog.Printf("event=invoice_insert_failed external_id=%q error=%q", dbInv.ExternalId, err.Error())
				http.Error(w, "Nie wprowadzono faktury do bazy danych", http.StatusInternalServerError)
				return

			}
		} else {
			app.errorLog.Printf("event=invoice_replace_failed external_id=%q error=%q", dbInv.ExternalId, err.Error())
			http.Error(w, "Nie wprowadzono faktury do bazy danych", http.StatusInternalServerError)
			return
		}
	}
	app.infoLog.Printf("event=invoice_%s invoice_id=%d external_id=%q status=%q", action, id, inv.InvoiceNumber, models.StatusPending)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": models.StatusPending})
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
	pageRaw := r.URL.Query().Get("page")
	query := r.URL.Query().Get("query")
	data, err := app.dashboardHelper(filter, pageRaw, query, app.config.DashPageSize)
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
	pageRaw := r.URL.Query().Get("page")
	query := r.URL.Query().Get("query")
	data, err := app.dashboardHelper(filter, pageRaw, query, app.config.DashPageSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		return
	}
	app.infoLog.Printf("event=dashboard_page_loaded page=%d filter=%q count=%d", data.Page, data.CurrentFilter, len(data.Invoices))
	app.renderer.render(w, "invoice-table", data)
}

func (app *application) getDashboardInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
		return
	}
	invoice, err := app.invoices.GetInvoice(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd przy pozyskiwaniu faktury: %v", err), http.StatusNotFound)
		return
	}
	app.renderer.render(w, "invoice-container", invoice)
}

func (app *application) dashboardHelper(filter, pageRaw, query string, pageSize int) (dashboardData, error) {
	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		page = 1
	}
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	invoices, err := app.invoices.GetAllInvoices(filter, query, page, pageSize)
	if err != nil {
		return dashboardData{}, err
	}
	more := false
	if len(invoices) > pageSize {
		more = true
		invoices = invoices[:pageSize]
	}
	data := dashboardData{
		Invoices:      invoices,
		CurrentFilter: filter,
		Page:          page,
		PrevPage:      prevPage,
		NextPage:      page + 1,
		More:          more,
		Query:         query,
		EscapedQuery:  url.QueryEscape(query),
		WebhookMaxRetries: app.config.User.Max_retries,
	}
	return data, nil
}

func (app *application) deleteDashboardInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
		return
	}
	if err := app.invoices.DeleteInvoice(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o id %d.", id), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		}
		return
	}
	app.infoLog.Printf("event=invoice_deleted invoice_id=%d", id)
	w.Header().Set("HX-Trigger", "refreshInvoices")
	w.WriteHeader(http.StatusOK)
}

func (app *application) getHealthLive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (app *application) getHealthReady(w http.ResponseWriter, r *http.Request) {
	if app.xsdValidator == nil {
		http.Error(w, "Narzędzie do walidacji plików XML nie jest uruchomione.", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := app.invoices.DB.PingContext(ctx); err != nil {
		http.Error(w, "Wystąpił problem z połączeniem z bazą danych.", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (app *application) getInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	var inv *models.Invoice
	var err error
	var id int64
	// idk about this i feel like there's too much nesting here
	// torn between WET and DRY here, but i think i'd rather have one handler if the
	// only diff is having the param be a string or int and call a different function
	// from the models package.
	externalId := r.URL.Query().Get("external_id")
	if externalId == "" {
		id, err = strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
			return
		}
		inv, err = app.invoices.GetInvoice(id)
	} else {
		inv, err = app.invoices.GetInvoiceExternalId(externalId)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if externalId != "" {
				http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o zewnętrznym id %s.", externalId), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o id %d.", id), http.StatusNotFound)
			}
		} else {
			http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		}
		return
	}
	payload := invoiceStatusRes{
		Id: inv.Id,
		ExternalId: inv.ExternalId,
		Status: string(inv.Status),
		KsefId: inv.KsefId,
		KsefErr: inv.KsefErr,
		SubmissionReference: inv.SubmissionReference,
		AttemptCount: inv.AttemptCount,
		CreatedAt: inv.CreatedAt,
		UpdatedAt: inv.UpdatedAt,
		WebhookDelivered: inv.WebhookDelivered,
		WebhookErr: inv.WebhookErr,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		app.errorLog.Printf("event=invoice_status_encode_failed invoice_id=%d error=%v", inv.Id, err)
	}
}

func (app *application) postDashboardRetryWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusBadRequest)
		return
	}
	if err := app.invoices.ResetWebhookAttemptCount(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o id %d.", id), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("HX-Trigger", "refreshInvoices")
	w.WriteHeader(http.StatusOK)
}

func (app *application) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	externalId := r.URL.Query().Get("external_id")
	if externalId == "" {
		http.Error(w, fmt.Sprintf("Niepoprawne id."), http.StatusBadRequest)
		return
	}
	if err := app.invoices.DeleteInvoiceExternalId(externalId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, fmt.Sprintf("Nie można było znaleźć faktury o zewnętrznym id: %s.", externalId), http.StatusNotFound)
		} else { 
			http.Error(w, fmt.Sprintf("Wystąpił błąd: %v", err), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}
