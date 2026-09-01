package config

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"io"
	"strings"
)

type Config struct {
	Ksef struct {
		Nip                     string `yaml:"nip"`
		Url                     string `yaml:"url"`
		HttpTimeoutSec          int    `yaml:"http_timeout_sec"`
		AuthRetryDelaySec       int    `yaml:"auth_retry_delay_sec"`
		PollingDelaySec         int    `yaml:"polling_delay_sec"`
		ConfirmationMaxAttempts int    `yaml:"confirmation_max_attempts"`
	} `yaml:"ksef"`
	Sqlite struct {
		Db_path       string `yaml:"db_path"`
		BusyTimeoutMs int    `yaml:"busy_timeout_ms"`
	} `yaml:"sqlite"`
	Server struct {
		Port        string `yaml:"port"`
		TLSCertPath string `yaml:"tls_cert_path"`
		TLSKeyPath  string `yaml:"tls_key_path"`
	}
	User struct {
		Max_retries int `yaml:"max_retries"`
	} `yaml:"user"`
	DashPageSize       int `yaml:"dash_page_size"`
	PollingInterval    int `yaml:"polling_interval"`
	ShutdownTimeoutSec int `yaml:"shutdown_timeout_sec"`
	SenderBatchSize    int `yaml:"sender_batch_size"`
	SenderWorkerLimit  int `yaml:"sender_worker_limit"`
}

// supposedly, passing in filepaths is a codesmell
func Load(f io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("configuration contains invalid YAML: %w", err)
	}
	return &config, nil
}

func (c *Config) Validate() error {
	var errors []string
	if c.Ksef.Nip == "" {
		errors = append(errors, "KSeF NIP is required")
	}
	if c.Ksef.Url == "" {
		errors = append(errors, "KSeF URL is required")
	}
	if c.Ksef.HttpTimeoutSec <= 0 {
		errors = append(errors, "KSeF HTTP timeout must be greater than 0 seconds")
	}
	if c.Ksef.AuthRetryDelaySec <= 0 {
		errors = append(errors, "authentication retry delay must be greater than 0 seconds")
	}
	if c.Ksef.PollingDelaySec <= 0 {
		errors = append(errors, "invoice-status polling delay must be greater than 0 seconds")
	}
	if c.Ksef.ConfirmationMaxAttempts <= 0 {
		errors = append(errors, "confirmation attempt limit must be greater than 0")
	}
	if c.Sqlite.Db_path == "" {
		errors = append(errors, "database path is required")
	}
	if c.Sqlite.BusyTimeoutMs <= 0 {
		errors = append(errors, "SQLite busy timeout must be greater than 0 milliseconds")
	}
	if c.Server.Port == "" {
		errors = append(errors, "server port is required")
	}
	if c.User.Max_retries < 0 {
		errors = append(errors, "retry limit must be at least 0")
	}
	if c.DashPageSize <= 0 {
		errors = append(errors, "dashboard page size must be greater than 0")
	}
	if c.PollingInterval <= 0 {
		errors = append(errors, "retry polling interval must be at least 1 second")
	}
	if c.ShutdownTimeoutSec <= 0 {
		errors = append(errors, "shutdown timeout must be at least 1 second")
	}
	if c.SenderBatchSize <= 0 {
		errors = append(errors, "sender batch size must be greater than 0")
	}
	if c.SenderWorkerLimit <= 0 {
		errors = append(errors, "sender worker limit must be greater than 0")
	}
	// i think this is a XOR?
	if (c.Server.TLSCertPath == "") != (c.Server.TLSKeyPath == "") {
		errors = append(errors, "TLS certificate and key paths need to be configured")
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}
	return nil
}
