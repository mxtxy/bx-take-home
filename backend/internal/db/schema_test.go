package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_DSN"); dsn != "" {
		return dsn
	}
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}
	return "brix:brix_password@tcp(127.0.0.1:3307)/brix_scheduler?parseTime=true&multiStatements=true"
}

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	database, err := Open(testDSN())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	acquireTestLock(t, database)

	migrationsDir := filepath.Join("..", "..", "migrations")
	if err := ApplySQLFile(ctx, database, filepath.Join(migrationsDir, "001_init.sql")); err != nil {
		t.Fatalf("apply init migration: %v", err)
	}
	resetAndSeed(t, database)
	return database
}

func acquireTestLock(t *testing.T, database *sql.DB) {
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

func resetAndSeed(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()
	if err := ResetTestDatabase(ctx, database); err != nil {
		t.Fatalf("reset db: %v", err)
	}
	if err := ApplySQLFile(ctx, database, filepath.Join("..", "..", "migrations", "002_seed.sql")); err != nil {
		t.Fatalf("apply seed migration: %v", err)
	}
}

func TestSchema_RequiredTablesExist(t *testing.T) {
	database := openMigratedDB(t)
	rows, err := database.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
	`)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		found[name] = true
	}
	for _, tableName := range []string{"users", "managers", "technicians", "quotes", "jobs", "notifications"} {
		if !found[tableName] {
			t.Fatalf("expected table %s to exist; found %#v", tableName, found)
		}
	}
}

func TestSchema_UsersEmailUnique(t *testing.T) {
	database := openMigratedDB(t)
	insert := `INSERT INTO users (email, password_hash, display_name, role) VALUES (?, 'hash', 'Duplicate', 'manager')`
	if _, err := database.Exec(insert, "duplicate@brix.test"); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err := database.Exec(insert, "duplicate@brix.test")
	if !IsDuplicateKey(err) {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestSchema_ManagersUserUnique(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`INSERT INTO managers (user_id) VALUES (1)`)
	if !IsDuplicateKey(err) {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestSchema_TechniciansUserUnique(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`INSERT INTO technicians (user_id) VALUES (3)`)
	if !IsDuplicateKey(err) {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestSchema_JobsQuoteUnique(t *testing.T) {
	database := openMigratedDB(t)
	insertJob(t, database, 1, 1, 1, "2026-05-12 10:00:00", "2026-05-12 12:00:00")
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (1, 2, 1, '2026-05-12 14:00:00', '2026-05-12 16:00:00', 'scheduled')
	`)
	if !IsDuplicateKey(err) {
		t.Fatalf("expected duplicate key error, got %v", err)
	}
}

func TestSchema_JobMustReferenceExistingQuote(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (999, 1, 1, '2026-05-12 10:00:00', '2026-05-12 12:00:00', 'scheduled')
	`)
	if !IsForeignKey(err) {
		t.Fatalf("expected foreign key error, got %v", err)
	}
}

func TestSchema_JobMustReferenceExistingTechnician(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (1, 999, 1, '2026-05-12 10:00:00', '2026-05-12 12:00:00', 'scheduled')
	`)
	if !IsForeignKey(err) {
		t.Fatalf("expected foreign key error, got %v", err)
	}
}

func TestSchema_JobMustReferenceExistingManager(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (1, 1, 999, '2026-05-12 10:00:00', '2026-05-12 12:00:00', 'scheduled')
	`)
	if !IsForeignKey(err) {
		t.Fatalf("expected foreign key error, got %v", err)
	}
}

func TestSchema_JobWindowMustBePositive(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (1, 1, 1, '2026-05-12 12:00:00', '2026-05-12 10:00:00', 'scheduled')
	`)
	if !IsCheckConstraint(err) {
		t.Fatalf("expected check constraint error, got %v", err)
	}
}

func TestSchema_JobWindowMustBeTwoHours(t *testing.T) {
	database := openMigratedDB(t)
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (1, 1, 1, '2026-05-12 10:00:00', '2026-05-12 11:00:00', 'scheduled')
	`)
	if !IsCheckConstraint(err) {
		t.Fatalf("expected check constraint error, got %v", err)
	}
}

func TestSeed_SeededUsersExist(t *testing.T) {
	database := openMigratedDB(t)
	rows, err := database.Query(`SELECT email, password_hash FROM users ORDER BY id`)
	if err != nil {
		t.Fatalf("query users: %v", err)
	}
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var email, hash string
		if err := rows.Scan(&email, &hash); err != nil {
			t.Fatalf("scan user: %v", err)
		}
		got[email] = hash
	}
	for _, email := range []string{"manager1@brix.test", "manager2@brix.test", "technician1@brix.test", "technician2@brix.test"} {
		if got[email] == "" {
			t.Fatalf("expected seeded user %s with non-empty hash, got %#v", email, got)
		}
	}
}

func TestSeed_SeededQuotesExist(t *testing.T) {
	database := openMigratedDB(t)
	rows, err := database.Query(`SELECT status, COUNT(*) FROM quotes GROUP BY status`)
	if err != nil {
		t.Fatalf("query quotes: %v", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			t.Fatalf("scan quote count: %v", err)
		}
		counts[status] = count
	}
	if counts["unscheduled"] != 5 || len(counts) != 1 {
		t.Fatalf("expected 5 unscheduled quotes only, got %#v", counts)
	}
}

func TestSeed_PreservesExistingQuoteStatusWhenReapplied(t *testing.T) {
	database := openMigratedDB(t)
	insertJob(t, database, 1, 1, 1, "2026-05-12 10:00:00", "2026-05-12 12:00:00")
	if _, err := database.Exec(`UPDATE quotes SET status = 'scheduled' WHERE id = 1`); err != nil {
		t.Fatalf("mark quote scheduled: %v", err)
	}

	if err := ApplySQLFile(context.Background(), database, filepath.Join("..", "..", "migrations", "002_seed.sql")); err != nil {
		t.Fatalf("reapply seed migration: %v", err)
	}

	var status string
	if err := database.QueryRow(`SELECT status FROM quotes WHERE id = 1`).Scan(&status); err != nil {
		t.Fatalf("query quote status: %v", err)
	}
	if status != "scheduled" {
		t.Fatalf("quote status = %s, want scheduled", status)
	}
	var jobCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM jobs WHERE quote_id = 1`).Scan(&jobCount); err != nil {
		t.Fatalf("query jobs: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("job count = %d, want 1", jobCount)
	}
}

func insertJob(t *testing.T, database *sql.DB, quoteID, technicianID, managerID int64, startsAt, endsAt string) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (?, ?, ?, ?, ?, 'scheduled')
	`, quoteID, technicianID, managerID, startsAt, endsAt)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
}

func TestTimeConstantParses(t *testing.T) {
	if _, err := time.Parse(time.RFC3339, "2026-05-12T00:00:00Z"); err != nil {
		t.Fatalf("fixed test time must parse: %v", err)
	}
	if errors.Is(sql.ErrNoRows, nil) {
		t.Fatal("sanity check failed")
	}
}
