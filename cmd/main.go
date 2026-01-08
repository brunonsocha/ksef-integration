package main

import (
	"ksef-integration/internal/config"
	"ksef-integration/internal/database"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

type application struct {
	infoLog *log.Logger
	errorLog *log.Logger
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
	infoLog.Printf("Połączono z bazą danych")
	defer db.Close()
}
