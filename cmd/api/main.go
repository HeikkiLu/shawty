package main

import (
	"log"
	"urlshortener/urlshortener/internal/config"
	"urlshortener/urlshortener/internal/db"
	"urlshortener/urlshortener/internal/http"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal(err)
	}

	pg, err := db.Open(cfg)

	if err != nil {
		log.Fatal(err)
	}

	defer pg.Close()

	engine := http.NewServer(cfg, pg)

	if err := engine.Run(cfg.BindAddr()); err != nil {
		log.Fatal(err)
	}
}
