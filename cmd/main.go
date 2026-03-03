package main

import (
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

type application struct {
	infoLog *log.Logger
	errorLog *log.Logger
	invoices *models.InvoiceModel
	config *config.Config
	ksefClient *ksef.Client
}

func main() {	
	infoLog := log.New(os.Stdout, "[INFO]\t", log.Ltime)
	errorLog := log.New(os.Stderr, "[BŁĄD]\t", log.Ltime)
	f, err := os.Open("config.yaml")
	if err != nil {
		errorLog.Fatal(err)
		return
	}
	defer f.Close()
	cfg, err := config.Load(f)
	if err != nil {
		errorLog.Fatal(err)
		return
	}
	infoLog.Printf("Wczytano plik konfiguracyjny")
	ksefClient := ksef.NewClient(cfg.Ksef.Nip, cfg.Ksef.Token, cfg.Ksef.Public_key_path, cfg.Ksef.Url)

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
		config: cfg,
		ksefClient: ksefClient,
	}
	go app.startSender()
	srv := &http.Server{
		Addr: ":" + cfg.Server.Port,
		Handler: app.routes(),
	}
	infoLog.Printf("Start serwera na porcie %s", srv.Addr)
	srv.ListenAndServe()
}
