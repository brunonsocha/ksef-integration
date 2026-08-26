package main

import (
	"context"
	"crypto/tls"
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"ksef-integration/internal/ksef"
	"ksef-integration/internal/models"
	xsdinvoices "ksef-integration/xsd"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

type application struct {
	infoLog           *log.Logger
	errorLog          *log.Logger
	invoices          *models.InvoiceModel
	config            *config.Config
	ksefClient        *ksef.Client
	renderer          *renderer
	xsdValidator      *xsdinvoices.Validator
	httpClient        *http.Client
	appApiKey         string
	dashboardUsername string
	dashboardPassword string
}

func main() {
	infoLog := log.New(os.Stdout, "[INFO]\t", log.Ltime)
	errorLog := log.New(os.Stderr, "[ERROR]\t", log.Ltime)
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
	xsdValidator, err := xsdinvoices.New(context.Background())
	if err != nil {
		errorLog.Fatal(err)
	}
	infoLog.Printf("event=xsd_validator_ready")
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
		renderer:          newRenderer(errorLog),
		xsdValidator:      xsdValidator,
		httpClient:        &client,
		appApiKey:         appApiKey,
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
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	go func() {
		scheme := "http"
		serve := srv.ListenAndServe
		if cfg.Server.TLSCertPath != "" && cfg.Server.TLSKeyPath != "" {
			scheme = "https"
			serve = func() error {
				return srv.ListenAndServeTLS(cfg.Server.TLSCertPath, cfg.Server.TLSKeyPath)
			}
		}
		app.infoLog.Printf("event=server_started scheme=%q addr=%q", scheme, srv.Addr)
		if err := serve(); err != nil && err != http.ErrServerClosed {
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
