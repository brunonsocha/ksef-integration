package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Ksef struct {
		Nip string `yaml:"nip"`
		Url string `yaml:"url"`
		Token string `yaml:"token"`
		Public_key_path string `yaml:"public_key_path"`
		Form struct {
			System_code string `yaml:"system_code"`
			Schema_version string `yaml:"schema_version"`
			Value string `yaml:"value"`
		} `yaml:"form"`
	} `yaml:"ksef"`
	Sqlite struct {
		Db_path string `yaml:"db_path"`
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
}

func Load(path string) (*Config, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
