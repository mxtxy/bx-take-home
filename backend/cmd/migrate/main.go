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

	if err := db.ApplySQLFile(context.Background(), database, db.MigrationPath("001_init.sql")); err != nil {
		log.Fatalf("apply migration: %v", err)
	}
	fmt.Println("migrations applied")
}
