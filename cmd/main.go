package main

import (
	"context"
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
	_ "modernc.org/sqlite"
)

type application struct {
	infoLog *log.Logger
	errorLog *log.Logger
	invoices *models.InvoiceModel
	config *config.Config
	ksefClient *ksef.Client
	renderer *renderer
	xsdValidator *xsdvalidate.XsdHandler
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
	if err := xsdvalidate.Init(); err != nil {
		errorLog.Fatal(err)
	}
	defer xsdvalidate.Cleanup()
	xsdValidator, err := xsdvalidate.NewXsdHandlerUrl(cfg.XSDPath, xsdvalidate.ParsErrDefault)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer xsdValidator.Free()
	infoLog.Printf("Załadowano narzędzie do walidacji XML.")
	app := &application{
		infoLog: infoLog,
		errorLog: errorLog,
		invoices: &models.InvoiceModel{
			DB: db,
		},
		config: cfg,
		ksefClient: ksef.NewClient(cfg.Ksef.Nip, cfg.Ksef.Token, cfg.Ksef.Public_key_path, cfg.Ksef.Url),
		renderer: newRenderer(),
		xsdValidator: xsdValidator,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.startSender(ctx)
	srv := &http.Server{
		Addr: ":" + cfg.Server.Port,
		Handler: app.routes(),
	}
	go func(){
		app.infoLog.Printf("Start serwera na porcie %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.errorLog.Fatalf("Błąd aplikacji: %v", err)
		}
	}()
	<- ctx.Done()
	app.infoLog.Printf("Zatrzymywanie aplikacji...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.errorLog.Fatalf("Błąd przy zamykaniu aplikacji: %v", err)
	}
	app.infoLog.Printf("Zatrzymano aplikację.")
}
