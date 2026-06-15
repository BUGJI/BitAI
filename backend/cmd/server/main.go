package main

import (
	"log"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/db"
	bithttp "bitapi/backend/internal/http"
)

func main() {
	cfg := config.Load()
	conn, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(conn); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := db.Seed(conn, cfg); err != nil {
		log.Fatalf("seed database: %v", err)
	}
	router := bithttp.NewRouter(conn, cfg)
	log.Printf("%s backend listening on %s", cfg.AppName, cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
