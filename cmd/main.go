package main

import (
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"ksef-integration/internal/models"
	"log"
	"os"
	"net/http"

	_ "modernc.org/sqlite"
)

type application struct {
	infoLog *log.Logger
	errorLog *log.Logger
	invoices *models.InvoiceModel
	config *config.Config
}

func main() {	
	infoLog := log.New(os.Stdout, "[INFO]\t", log.Ltime)
	errorLog := log.New(os.Stderr, "[BŁĄD]\t", log.Ltime)
	cfg, err := config.Load("config.yaml")
	if err != nil {
		errorLog.Fatal(err)
		return
	}
	infoLog.Printf("Wczytano plik konfiguracyjny")
	db, err := database.Connect(cfg.Sqlite.Db_path)
	if err != nil {
		errorLog.Fatal(err)
		return
	}
	defer db.Close()
	infoLog.Printf("Połączono z bazą danych")
	if err = database.Setup(db); err != nil {
		errorLog.Fatal(err)
	}
	infoLog.Printf("Załadowano bazę danych")
	app := &application{
		infoLog: infoLog,
		errorLog: errorLog,
		invoices: &models.InvoiceModel{
			DB: db,
		},
	}

	srv := &http.Server{
		Addr: ":" + cfg.Server.Port,
		Handler: app.routes(),
	}
	infoLog.Printf("Start serwera na porcie %s", srv.Addr)
	srv.ListenAndServe()
}
