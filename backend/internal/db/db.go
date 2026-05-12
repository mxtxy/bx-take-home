package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
)

func Open(dsn string) (*sql.DB, error) {
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(10)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func DefaultDSN() string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true"
}

func MigrationDSN() string {
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true&multiStatements=true"
}

func ApplySQLFile(ctx context.Context, database *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sqlText := strings.TrimSpace(string(content))
	if sqlText == "" {
		return nil
	}
	_, err = database.ExecContext(ctx, sqlText)
	return err
}

func MigrationPath(filename string) string {
	for _, dir := range []string{
		os.Getenv("MIGRATIONS_DIR"),
		"migrations",
		filepath.Join("..", "..", "migrations"),
		"/app/migrations",
	} {
		if dir == "" {
			continue
		}
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return filepath.Join("migrations", filename)
}

func ResetTestDatabase(ctx context.Context, database *sql.DB) error {
	statements := []string{
		"SET FOREIGN_KEY_CHECKS = 0",
		"TRUNCATE TABLE schedule_audit_logs",
		"TRUNCATE TABLE notifications",
		"TRUNCATE TABLE sessions",
		"TRUNCATE TABLE jobs",
		"TRUNCATE TABLE quotes",
		"TRUNCATE TABLE technician_availability_rules",
		"TRUNCATE TABLE technicians",
		"TRUNCATE TABLE managers",
		"TRUNCATE TABLE users",
		"TRUNCATE TABLE organizations",
		"SET FOREIGN_KEY_CHECKS = 1",
	}
	for _, statement := range statements {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func IsDuplicateKey(err error) bool {
	return hasMySQLError(err, 1062)
}

func IsForeignKey(err error) bool {
	return hasMySQLError(err, 1451, 1452)
}

func IsCheckConstraint(err error) bool {
	return hasMySQLError(err, 3819)
}

func hasMySQLError(err error, numbers ...uint16) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	for _, number := range numbers {
		if mysqlErr.Number == number {
			return true
		}
	}
	return false
}
