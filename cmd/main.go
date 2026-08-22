package main

import (
	"context"
	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type application struct {
	infoLog      *log.Logger
	errorLog     *log.Logger
	invoices     *models.InvoiceModel
	config       *config.Config
	ksefClient   *ksef.Client
	renderer     *renderer
	xsdValidator *xsdvalidate.XsdHandler
	httpClient   *http.Client
	appApiKey    string
	dashboardUsername string
	dashboardPassword string
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
	}
	infoLog.Printf("event=config_loaded")
	if err := cfg.Validate(); err != nil {
		errorLog.Fatalf("event=config_validation_failed error=%q", err.Error())
	}
	infoLog.Printf("event=config_validated")
	ksefToken := strings.TrimSpace(os.Getenv("KSEF_TOKEN"))
	if ksefToken == "" {
		errorLog.Fatal("event=environment_validation_failed variable=\"KSEF_TOKEN\" reason=\"missing\"")
	}
	appApiKey := strings.TrimSpace(os.Getenv("APP_API_KEY"))
	if appApiKey == "" {
		errorLog.Fatal("event=environment_validation_failed variable=\"APP_API_KEY\" reason=\"missing\"")
	}
	dashboardUsername := strings.TrimSpace(os.Getenv("DASHBOARD_USERNAME"))
	if dashboardUsername == "" {
		errorLog.Fatal("event=environment_validation_failed variable=\"DASHBOARD_USERNAME\" reason=\"missing\"")
	}

	dashboardPassword := strings.TrimSpace(os.Getenv("DASHBOARD_PASSWORD"))
	if dashboardPassword == "" {
		errorLog.Fatal("event=environment_validation_failed variable=\"DASHBOARD_PASSWORD\" reason=\"missing\"")
	}
	infoLog.Printf("event=environment_validated")
	db, err := database.Connect(cfg.Sqlite.Db_path, cfg.Sqlite.BusyTimeoutMs)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()
	infoLog.Printf("event=db_connected db_path=%q", cfg.Sqlite.Db_path)
	if err = database.Setup(db); err != nil {
		errorLog.Fatal(err)
	}
	infoLog.Printf("event=db_ready")
	if err := xsdvalidate.Init(); err != nil {
		errorLog.Fatal(err)
	}
	defer xsdvalidate.Cleanup()
	xsdValidator, err := xsdvalidate.NewXsdHandlerUrl(cfg.XSDPath, xsdvalidate.ParsErrDefault)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer xsdValidator.Free()
	infoLog.Printf("event=xsd_validator_ready xsd_path=%q", cfg.XSDPath)
	client := http.Client{
		Timeout: time.Duration(cfg.Ksef.HttpTimeoutSec) * time.Second,
	}
	app := &application{
		infoLog:  infoLog,
		errorLog: errorLog,
		invoices: &models.InvoiceModel{
			DB: db,
		},
		config: cfg,
		ksefClient: ksef.NewClient(
			cfg.Ksef.Nip,
			ksefToken,
			cfg.Ksef.Url,
			cfg.Ksef.HttpTimeoutSec,
			cfg.Ksef.AuthRetryDelaySec,
			cfg.Ksef.PollingDelaySec,
		),
		renderer:     newRenderer(),
		xsdValidator: xsdValidator,
		httpClient:   &client,
		appApiKey: appApiKey,
		dashboardUsername: dashboardUsername,
		dashboardPassword: dashboardPassword,
	}
	if err := app.invoices.RecoverProcessingInvoices(); err != nil {
		app.errorLog.Fatalf("event=invoice_recovery_failed error=%q", err.Error())
	}
	app.infoLog.Printf("event=invoice_recovery_finished")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdownC := make(chan struct{}, 1)
	go app.startSender(ctx, shutdownC)
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: app.routes(),
	}
	go func() {
		app.infoLog.Printf("event=server_started addr=%q", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.errorLog.Fatalf("event=server_failed error=%q", err.Error())
		}
	}()
	<-ctx.Done()
	app.infoLog.Printf("event=shutdown_started")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(app.config.ShutdownTimeoutSec)*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.errorLog.Fatalf("event=shutdown_failed error=%q", err.Error())
	}
	<-shutdownC
	app.infoLog.Printf("event=shutdown_finished")
}
