package main

import (
	"log"

	"github.com/exf1kzz/go-labs-avito/internal/config"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("configuration loaded: http_addr=%s", config.HTTP.Address)
}
