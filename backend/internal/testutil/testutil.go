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
	DropSchema(t, database)
	if err := db.ApplySQLFile(context.Background(), database, filepath.Join("..", "..", "migrations", "001_init.sql")); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}

func DropSchema(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec(`
		SET FOREIGN_KEY_CHECKS = 0;
		DROP TABLE IF EXISTS schedule_audit_logs;
		DROP TABLE IF EXISTS notifications;
		DROP TABLE IF EXISTS sessions;
		DROP TABLE IF EXISTS jobs;
		DROP TABLE IF EXISTS quotes;
		DROP TABLE IF EXISTS technician_availability_rules;
		DROP TABLE IF EXISTS technicians;
		DROP TABLE IF EXISTS managers;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS organizations;
		SET FOREIGN_KEY_CHECKS = 1;
	`); err != nil {
		t.Fatalf("drop schema: %v", err)
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
		UserID:         1,
		OrganizationID: 1,
		Email:          "manager1@brix.test",
		DisplayName:    "Sarah Manager",
		Role:           domain.RoleManager,
		ManagerID:      &managerID,
	}
}

func Manager2() domain.Actor {
	managerID := int64(2)
	return domain.Actor{
		UserID:         2,
		OrganizationID: 1,
		Email:          "manager2@brix.test",
		DisplayName:    "Alex Manager",
		Role:           domain.RoleManager,
		ManagerID:      &managerID,
	}
}

func Manager3() domain.Actor {
	managerID := int64(3)
	return domain.Actor{
		UserID:         5,
		OrganizationID: 2,
		Email:          "manager3@other.test",
		DisplayName:    "Other Org Manager",
		Role:           domain.RoleManager,
		ManagerID:      &managerID,
	}
}

func Technician1() domain.Actor {
	technicianID := int64(1)
	return domain.Actor{
		UserID:         3,
		OrganizationID: 1,
		Email:          "technician1@brix.test",
		DisplayName:    "Tom Technician",
		Role:           domain.RoleTechnician,
		TechnicianID:   &technicianID,
	}
}

func Technician2() domain.Actor {
	technicianID := int64(2)
	return domain.Actor{
		UserID:         4,
		OrganizationID: 1,
		Email:          "technician2@brix.test",
		DisplayName:    "Priya Technician",
		Role:           domain.RoleTechnician,
		TechnicianID:   &technicianID,
	}
}

func Technician3() domain.Actor {
	technicianID := int64(3)
	return domain.Actor{
		UserID:         6,
		OrganizationID: 2,
		Email:          "technician3@other.test",
		DisplayName:    "Other Org Tech",
		Role:           domain.RoleTechnician,
		TechnicianID:   &technicianID,
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
	var organizationID int64
	if err := database.QueryRow(`SELECT organization_id FROM quotes WHERE id = ?`, quoteID).Scan(&organizationID); err != nil {
		t.Fatalf("query quote organization: %v", err)
	}
	result, err := database.Exec(`
		INSERT INTO jobs (organization_id, quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (?, ?, ?, ?, ?, ?, 'scheduled')
	`, organizationID, quoteID, technicianID, managerID, start, end)
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

func InsertOtherOrganizationFixture(t *testing.T, database *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO organizations (id, name, slug, created_at) VALUES (2, 'Other Organization', 'other', '2026-05-11 00:00:00.000000') ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		`INSERT INTO users (id, organization_id, email, password_hash, display_name, role, created_at, updated_at) VALUES
			(5, 2, 'manager3@other.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Other Org Manager', 'manager', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
			(6, 2, 'technician3@other.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Other Org Tech', 'technician', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000')
		 ON DUPLICATE KEY UPDATE organization_id = VALUES(organization_id), display_name = VALUES(display_name), role = VALUES(role)`,
		`INSERT INTO managers (id, user_id, created_at) VALUES (3, 5, '2026-05-11 00:00:00.000000') ON DUPLICATE KEY UPDATE user_id = VALUES(user_id)`,
		`INSERT INTO technicians (id, user_id, created_at) VALUES (3, 6, '2026-05-11 00:00:00.000000') ON DUPLICATE KEY UPDATE user_id = VALUES(user_id)`,
		`INSERT INTO technician_availability_rules (organization_id, technician_id, weekday, starts_at, ends_at) VALUES
			(2, 3, 1, '08:00:00', '18:00:00'),
			(2, 3, 2, '08:00:00', '18:00:00'),
			(2, 3, 3, '08:00:00', '18:00:00'),
			(2, 3, 4, '08:00:00', '18:00:00'),
			(2, 3, 5, '08:00:00', '18:00:00')
		 ON DUPLICATE KEY UPDATE starts_at = VALUES(starts_at), ends_at = VALUES(ends_at)`,
		`INSERT INTO quotes (id, organization_id, customer_name, description, status, created_at, updated_at) VALUES
			(6, 2, 'Other Org HVAC', 'Repair rooftop unit', 'unscheduled', '2026-05-11 00:05:00.000000', '2026-05-11 00:05:00.000000')
		 ON DUPLICATE KEY UPDATE customer_name = VALUES(customer_name), description = VALUES(description), updated_at = VALUES(updated_at)`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("insert other organization fixture: %v", err)
		}
	}
}

func CountRows(t *testing.T, database *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := database.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}
