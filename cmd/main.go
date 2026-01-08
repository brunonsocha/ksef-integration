package main

import (
	"fmt"
	"ksef-integration/internal/config"
	"log"
)

// we'll set a config reader up, read from it, create a schema in db,

type application struct {

}

func main() {	
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(cfg.Ksef.Nip)
}
