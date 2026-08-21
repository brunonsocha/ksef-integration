package config

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"io"
	"os"
	"strings"
)

type Config struct {
	Ksef struct {
		Nip                     string `yaml:"nip"`
		Url                     string `yaml:"url"`
		Token                   string `yaml:"token"`
		HttpTimeoutSec          int    `yaml:"http_timeout_sec"`
		AuthRetryDelaySec       int    `yaml:"auth_retry_delay_sec"`
		PollingDelaySec         int    `yaml:"polling_delay_sec"`
		ConfirmationMaxAttempts int    `yaml:"confirmation_max_attempts"`
		Form                    struct {
			System_code    string `yaml:"system_code"`
			Schema_version string `yaml:"schema_version"`
			Value          string `yaml:"value"`
		} `yaml:"form"`
	} `yaml:"ksef"`
	Sqlite struct {
		Db_path       string `yaml:"db_path"`
		BusyTimeoutMs int    `yaml:"busy_timeout_ms"`
	} `yaml:"sqlite"`
	Server struct {
		Port string `yaml:"port"`
	}
	Runonce struct {
		Xml_path string `yaml:"xml_path"`
	} `yaml:"runonce"`
	User struct {
		Max_retries int `yaml:"max_retries"`
	} `yaml:"user"`
	XSDPath            string `yaml:"xsd_path"`
	DashPageSize       int    `yaml:"dash_page_size"`
	PollingInterval    int    `yaml:"polling_interval"`
	ShutdownTimeoutSec int    `yaml:"shutdown_timeout_sec"`
	SenderBatchSize    int    `yaml:"sender_batch_size"`
	SenderWorkerLimit  int    `yaml:"sender_worker_limit"`
}

// supposedly, passing in filepaths is a codesmell
func Load(f io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("Niepoprawna składnia pliku .yaml - %v", err)
	}
	return &config, nil
}

func (c *Config) Validate() error {
	var errors []string
	if c.Ksef.Nip == "" {
		errors = append(errors, "Brakuje NIP.")
	}
	if c.Ksef.Url == "" {
		errors = append(errors, "Brakuje adresu URL KSeF.")
	}
	if c.Ksef.Token == "" {
		errors = append(errors, "Brakuje tokena KSeF - pozyskaj go z panelu KSeF.")
	}
	if c.Ksef.HttpTimeoutSec <= 0 {
		errors = append(errors, "Timeout klienta HTTP KSeF musi być większy niż 0 sekund.")
	}
	if c.Ksef.AuthRetryDelaySec <= 0 {
		errors = append(errors, "Opóźnienie ponowienia autoryzacji musi być większe niż 0 sekund.")
	}
	if c.Ksef.PollingDelaySec <= 0 {
		errors = append(errors, "Opóźnienie przy sprawdzaniu statusu faktury musi być większe niż 0 sekund.")
	}
	if c.Ksef.ConfirmationMaxAttempts <= 0 {
		errors = append(errors, "Maksymalna liczba prób potwierdzenia musi być większa niż 0.")
	}
	if c.Sqlite.Db_path == "" {
		errors = append(errors, "Brakuje ścieżki do bazy danych.")
	}
	if c.Sqlite.BusyTimeoutMs <= 0 {
		errors = append(errors, "Timeout oczekiwania SQLite musi być większy niż 0 milisekund.")
	}
	if c.Server.Port == "" {
		errors = append(errors, "Brakuje portu, na którym ma działać aplikacja.")
	}
	if c.XSDPath == "" {
		errors = append(errors, "Brakuje ścieżki do schematu XSD.")
	} else if _, err := os.Stat(c.XSDPath); err != nil {
		errors = append(errors, "Niepoprawna ścieżka do schematu XSD.")
	}
	if c.User.Max_retries < 0 {
		errors = append(errors, "Maksymalna ilość prób ponownych musi być minimum 0.")
	}
	if c.DashPageSize <= 0 {
		errors = append(errors, "Strony w panelu nie mogą mieścić zera faktur.")
	}
	if c.PollingInterval <= 0 {
		errors = append(errors, "Próby ponowne nie mogą być podejmowane co mniej niż 1 sekundę.")
	}
	if c.ShutdownTimeoutSec <= 0 {
		errors = append(errors, "Zamknięcie nie może występować w mniej niż 1 sekundę.")
	}
	if c.SenderBatchSize <= 0 {
		errors = append(errors, "Batch wysyłki musi być większy niż 0.")
	}
	if c.SenderWorkerLimit <= 0 {
		errors = append(errors, "Limit równoległych zadań wysyłki musi być większy niż 0.")
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}
	return nil
}
