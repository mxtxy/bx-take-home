package db

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func TestOpen_InvalidDSNReturnsError(t *testing.T) {
	database, err := Open("%")
	if err == nil {
		if database != nil {
			_ = database.Close()
		}
		t.Fatal("expected invalid DSN error")
	}
}

func TestOpen_PingFailureClosesDatabase(t *testing.T) {
	database, err := Open("brix:brix_password@tcp(127.0.0.1:1)/brix_scheduler?parseTime=true")
	if err == nil {
		_ = database.Close()
		t.Fatal("expected ping error")
	}
}

func TestDefaultDSN_UsesEnvironmentOverride(t *testing.T) {
	t.Setenv("DATABASE_DSN", "custom-dsn")

	if got := DefaultDSN(); got != "custom-dsn" {
		t.Fatalf("DefaultDSN() = %q", got)
	}
}

func TestDefaultDSN_UsesLocalDefault(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")

	if got := DefaultDSN(); got != "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true" {
		t.Fatalf("DefaultDSN() = %q", got)
	}
}

func TestMigrationDSN_UsesEnvironmentOverride(t *testing.T) {
	t.Setenv("DATABASE_DSN", "custom-dsn")

	if got := MigrationDSN(); got != "custom-dsn" {
		t.Fatalf("MigrationDSN() = %q", got)
	}
}

func TestMigrationDSN_UsesLocalDefaultWithMultiStatements(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")

	if got := MigrationDSN(); got != "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true&multiStatements=true" {
		t.Fatalf("MigrationDSN() = %q", got)
	}
}

func TestApplySQLFile_ReturnsReadError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()

	err = ApplySQLFile(context.Background(), database, filepath.Join(t.TempDir(), "missing.sql"))
	if err == nil {
		t.Fatal("expected read error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestApplySQLFile_IgnoresEmptyFile(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	path := filepath.Join(t.TempDir(), "empty.sql")
	if err := os.WriteFile(path, []byte(" \n\t "), 0o600); err != nil {
		t.Fatalf("write sql file: %v", err)
	}

	if err := ApplySQLFile(context.Background(), database, path); err != nil {
		t.Fatalf("ApplySQLFile: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMigrationPath_UsesEnvironmentDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001_init.sql"), []byte("SELECT 1"), 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	t.Setenv("MIGRATIONS_DIR", dir)

	if got := MigrationPath("001_init.sql"); got != filepath.Join(dir, "001_init.sql") {
		t.Fatalf("MigrationPath() = %q", got)
	}
}

func TestMigrationPath_UsesRepositoryMigrationDirectory(t *testing.T) {
	t.Setenv("MIGRATIONS_DIR", "")

	if got := MigrationPath("001_init.sql"); got != filepath.Join("..", "..", "migrations", "001_init.sql") {
		t.Fatalf("MigrationPath() = %q", got)
	}
}

func TestMigrationPath_FallsBackWhenFileIsMissing(t *testing.T) {
	t.Setenv("MIGRATIONS_DIR", t.TempDir())

	if got := MigrationPath("missing.sql"); got != filepath.Join("migrations", "missing.sql") {
		t.Fatalf("MigrationPath() = %q", got)
	}
}

func TestResetTestDatabase_ReturnsExecError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectExec("SET FOREIGN_KEY_CHECKS = 0").WillReturnError(errors.New("exec failed"))

	err = ResetTestDatabase(context.Background(), database)
	if err == nil {
		t.Fatal("expected reset error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMySQLErrorHelpers_ReturnFalseForNilAndNonMatchingErrors(t *testing.T) {
	if IsDuplicateKey(nil) {
		t.Fatal("nil should not be duplicate key")
	}
	if IsForeignKey(errors.New("plain error")) {
		t.Fatal("plain error should not be foreign key")
	}
	if IsCheckConstraint(&mysql.MySQLError{Number: 9999}) {
		t.Fatal("unmatched MySQL number should not be check constraint")
	}
}
