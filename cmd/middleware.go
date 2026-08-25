package main

import (
	"net/http"
	"strings"
)

func (app *application) requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		provided, found := strings.CutPrefix(authorization, "Bearer ")
		if !found || provided != app.appApiKey {
			app.errorLog.Printf("event=api_authentication_failed method=%q path=%q", r.Method, r.URL.Path)
			w.Header().Set("WWW-Authenticate", "Bearer")
			app.writeErrorRes(w, http.StatusUnauthorized, "unauthorized", "an API key is required.")
			return
		}
		next(w, r)
	}
}

func (app *application) requireDashboardAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != app.dashboardUsername || password != app.dashboardPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="KSeF dashboard"`)
			http.Error(w, "brak autoryzacji.", http.StatusUnauthorized)
			app.errorLog.Printf("event=dashboard_authentication_failed method=%q path=%q remote_addr=%q", r.Method, r.URL.Path, r.RemoteAddr)
			return
		}
		next(w, r)
	}
}
