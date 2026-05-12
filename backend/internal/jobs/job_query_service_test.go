package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestJobQueryService_ListJobs_ManagerSeesOwnJobsOnly(t *testing.T) {
	database := testutil.PrepareDB(t)
	job1 := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	_ = testutil.InsertScheduledJob(t, database, 2, 2, 2, "2026-05-12T12:00:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != job1 {
		t.Fatalf("jobs = %#v", jobs)
	}
	if jobs[0].OrganizationID != 1 {
		t.Fatalf("organization id = %d", jobs[0].OrganizationID)
	}
}

func TestJobQueryService_ListJobs_FiltersByOrganization(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	_ = testutil.InsertScheduledJob(t, database, 6, 3, 3, "2026-05-12T10:00:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != jobID {
		t.Fatalf("manager1 jobs = %#v", jobs)
	}
}

func TestJobQueryService_ListJobs_TechnicianSeesAssignedJobsOnly(t *testing.T) {
	database := testutil.PrepareDB(t)
	job1 := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	_ = testutil.InsertScheduledJob(t, database, 2, 2, 1, "2026-05-12T12:00:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Technician1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != job1 {
		t.Fatalf("jobs = %#v", jobs)
	}
}

func TestJobQueryService_ListJobs_SortedByStartsAtThenID(t *testing.T) {
	database := testutil.PrepareDB(t)
	jobA := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T14:00:00Z")
	jobB := testutil.InsertScheduledJob(t, database, 2, 1, 1, "2026-05-12T10:00:00Z")
	jobC := testutil.InsertScheduledJob(t, database, 3, 1, 1, "2026-05-12T10:00:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	got := []int64{jobs[0].ID, jobs[1].ID, jobs[2].ID}
	want := []int64{jobB, jobC, jobA}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestJobQueryService_ListJobs_IncludesQuoteTechnicianManagerNames(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	job := jobs[0]
	if job.QuoteCustomerName != "Acme Plumbing" || job.QuoteDescription == "" || job.TechnicianName != "Tom Technician" || job.ManagerName != "Sarah Manager" {
		t.Fatalf("joined fields missing: %#v", job)
	}
}

func TestJobQueryService_ListJobs_IncludesCompletedAt(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertCompletedJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z", "2026-05-12T12:30:00Z")
	service := NewQueryService(database)

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].CompletedAt == nil {
		t.Fatalf("completed job missing completedAt: %#v", jobs)
	}
}

func TestJobQueryService_ListJobs_EmptyResultReturnsEmptySlice(t *testing.T) {
	service := NewQueryService(testutil.PrepareDB(t))

	jobs, err := service.ListJobs(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if jobs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(jobs) != 0 {
		t.Fatalf("len = %d", len(jobs))
	}
}

func TestJobQueryService_ListJobs_RejectsMalformedActor(t *testing.T) {
	service := NewQueryService(testutil.PrepareDB(t))

	_, err := service.ListJobs(context.Background(), domain.Actor{UserID: 1, Role: domain.RoleManager})
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("manager code = %v, err = %v", code, err)
	}
	_, err = service.ListJobs(context.Background(), domain.Actor{UserID: 3, Role: domain.RoleTechnician})
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("technician code = %v, err = %v", code, err)
	}
	_, err = service.ListJobs(context.Background(), domain.Actor{UserID: 5, Role: "dispatcher"})
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("unknown role code = %v, err = %v", code, err)
	}
}

func TestJobQueryService_ListJobs_ReturnsQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("query failed"))
	service := NewQueryService(database)

	_, err = service.ListJobs(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected query error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestJobQueryService_ListJobs_ReturnsScanError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	rows := sqlmock.NewRows([]string{
		"id", "organization_id", "quote_id", "customer_name", "description",
		"technician_id", "technician_name",
		"manager_id", "manager_name",
		"starts_at", "ends_at", "status", "completed_at",
		"created_at", "updated_at",
	}).AddRow(
		"bad-id", int64(1), int64(1), "Acme Plumbing", "Replace tap",
		int64(1), "Tom Technician",
		int64(1), "Sarah Manager",
		time.Now(), time.Now(), "scheduled", nil,
		time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	service := NewQueryService(database)

	_, err = service.ListJobs(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
