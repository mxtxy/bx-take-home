package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/db"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

func FixedTime() time.Time {
	return time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
}

func TestDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true&multiStatements=true"
}

func OpenDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(TestDSN())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func PrepareDB(t *testing.T) *sql.DB {
	t.Helper()
	database := OpenDB(t)
	AcquireDatabaseLock(t, database)
	ApplySchema(t, database)
	ResetAndSeed(t, database)
	return database
}

func AcquireDatabaseLock(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()
	conn, err := database.Conn(ctx)
	if err != nil {
		t.Fatalf("open lock connection: %v", err)
	}
	var acquired int
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK('brix_scheduler_test_lock', 30)`).Scan(&acquired); err != nil {
		_ = conn.Close()
		t.Fatalf("acquire db lock: %v", err)
	}
	if acquired != 1 {
		_ = conn.Close()
		t.Fatalf("database test lock not acquired")
	}
	t.Cleanup(func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK('brix_scheduler_test_lock')`)
		_ = conn.Close()
	})
}

func ApplySchema(t *testing.T, database *sql.DB) {
	t.Helper()
	if err := db.ApplySQLFile(context.Background(), database, filepath.Join("..", "..", "migrations", "001_init.sql")); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func ResetAndSeed(t *testing.T, database *sql.DB) {
	t.Helper()
	if err := db.ResetTestDatabase(context.Background(), database); err != nil {
		t.Fatalf("reset db: %v", err)
	}
	if err := db.ApplySQLFile(context.Background(), database, filepath.Join("..", "..", "migrations", "002_seed.sql")); err != nil {
		t.Fatalf("seed db: %v", err)
	}
}

func Manager1() domain.Actor {
	managerID := int64(1)
	return domain.Actor{
		UserID:      1,
		Email:       "manager1@brix.test",
		DisplayName: "Sarah Manager",
		Role:        domain.RoleManager,
		ManagerID:   &managerID,
	}
}

func Manager2() domain.Actor {
	managerID := int64(2)
	return domain.Actor{
		UserID:      2,
		Email:       "manager2@brix.test",
		DisplayName: "Alex Manager",
		Role:        domain.RoleManager,
		ManagerID:   &managerID,
	}
}

func Technician1() domain.Actor {
	technicianID := int64(1)
	return domain.Actor{
		UserID:       3,
		Email:        "technician1@brix.test",
		DisplayName:  "Tom Technician",
		Role:         domain.RoleTechnician,
		TechnicianID: &technicianID,
	}
}

func Technician2() domain.Actor {
	technicianID := int64(2)
	return domain.Actor{
		UserID:       4,
		Email:        "technician2@brix.test",
		DisplayName:  "Priya Technician",
		Role:         domain.RoleTechnician,
		TechnicianID: &technicianID,
	}
}

func MustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func InsertScheduledJob(t *testing.T, database *sql.DB, quoteID, technicianID, managerID int64, startsAt string) int64 {
	t.Helper()
	start := MustTime(t, startsAt)
	end := start.Add(2 * time.Hour)
	result, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (?, ?, ?, ?, ?, 'scheduled')
	`, quoteID, technicianID, managerID, start, end)
	if err != nil {
		t.Fatalf("insert scheduled job: %v", err)
	}
	jobID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	if _, err := database.Exec(`UPDATE quotes SET status = 'scheduled' WHERE id = ?`, quoteID); err != nil {
		t.Fatalf("update quote status: %v", err)
	}
	return jobID
}

func InsertCompletedJob(t *testing.T, database *sql.DB, quoteID, technicianID, managerID int64, startsAt string, completedAt string) int64 {
	t.Helper()
	jobID := InsertScheduledJob(t, database, quoteID, technicianID, managerID, startsAt)
	completed := MustTime(t, completedAt)
	if _, err := database.Exec(`UPDATE jobs SET status = 'completed', completed_at = ? WHERE id = ?`, completed, jobID); err != nil {
		t.Fatalf("complete job: %v", err)
	}
	return jobID
}
