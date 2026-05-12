package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mxtxy/bx-take-home/backend/internal/db"
)

func main() {
	database, err := db.Open(db.MigrationDSN())
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := db.ApplySQLFile(context.Background(), database, db.MigrationPath("002_seed.sql")); err != nil {
		log.Fatalf("apply seed: %v", err)
	}
	fmt.Println("seed applied")
}
