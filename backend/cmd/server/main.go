package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/zhibo/backend/internal/api"
	"github.com/zhibo/backend/internal/config"
	"github.com/zhibo/backend/internal/infra/mysql"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	cfg := config.Load()

	db, err := mysql.Open(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer db.Close()

	router := api.NewRouter(cfg, db)

	addr := ":" + cfg.Port
	log.Printf("zhibo API listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}
