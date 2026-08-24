package main

import (
	"encoding/json"
	"net/http"
)

func (app *application) writeRes(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		app.errorLog.Printf("event=json_response_encode_failed http_status=%d error=%q", status, err)
	}
}

func (app *application) writeErrorRes(w http.ResponseWriter, status int, code, message string) {
	app.writeRes(w, status, errorRes{
		Status:  "error",
		Code:    code,
		Message: message,
	})
}
