package scheduling

import (
	"context"
	"testing"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestSchedulingService_CompleteJob_Success(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	clockTime := testutil.MustTime(t, "2026-05-12T16:00:00Z")
	service.clock = func() time.Time { return clockTime }

	job, events, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: jobID})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if job.Status != domain.JobCompleted || job.CompletedAt == nil || !job.CompletedAt.Equal(clockTime) {
		t.Fatalf("job = %#v", job)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = 1 AND type = 'job_completed'`); got != 1 {
		t.Fatalf("manager notification count = %d", got)
	}
	assertHasEvent(t, events, "notification.created", int64Ptr(1), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(1), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(3), nil)
}

func TestSchedulingService_CompleteJob_ManagerActorForbidden(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, _, err := service.CompleteJob(context.Background(), testutil.Manager1(), domain.CompleteJobInput{JobID: jobID})
	assertCode(t, err, domain.ErrorForbidden)
	if got := scalarString(t, database, `SELECT status FROM jobs WHERE id = ?`, jobID); got != "scheduled" {
		t.Fatalf("status = %s", got)
	}
}

func TestSchedulingService_CompleteJob_WrongTechnicianForbidden(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, events, err := service.CompleteJob(context.Background(), testutil.Technician2(), domain.CompleteJobInput{JobID: jobID})
	assertCode(t, err, domain.ErrorForbidden)
	if got := scalarString(t, database, `SELECT status FROM jobs WHERE id = ?`, jobID); got != "scheduled" {
		t.Fatalf("status = %s", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications`); got != 0 {
		t.Fatalf("notification count = %d", got)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v", events)
	}
}

func TestSchedulingService_CompleteJob_RejectsCrossOrganizationActor(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician3(), domain.CompleteJobInput{JobID: jobID})
	assertCode(t, err, domain.ErrorNotFound)
	if got := scalarString(t, database, `SELECT status FROM jobs WHERE id = ?`, jobID); got != "scheduled" {
		t.Fatalf("status = %s", got)
	}
}

func TestSchedulingService_CompleteJob_InvalidInput(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 0})
	assertCode(t, err, domain.ErrorInvalidInput)
}

func TestSchedulingService_CompleteJob_MissingJob(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 999})
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_CompleteJob_AlreadyCompletedConflict(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertCompletedJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z", "2026-05-12T15:00:00Z")

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: jobID})
	assertCode(t, err, domain.ErrorJobAlreadyCompleted)
	if got := scalarString(t, database, `SELECT DATE_FORMAT(completed_at, '%Y-%m-%d %H') FROM jobs WHERE id = ?`, jobID); got != "2026-05-12 15" {
		t.Fatalf("completed_at changed: %s", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications`); got != 0 {
		t.Fatalf("notification count = %d", got)
	}
}

func TestSchedulingService_CompleteJob_WritesAuditLog(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	clockTime := testutil.MustTime(t, "2026-05-12T16:00:00Z")
	service.clock = func() time.Time { return clockTime }

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: jobID})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	var action, completedAt string
	if err := database.QueryRow(`
		SELECT action, DATE_FORMAT(new_completed_at, '%Y-%m-%d %H:%i:%s')
		FROM schedule_audit_logs
		WHERE job_id = ? AND action = 'job_completed'
	`, jobID).Scan(&action, &completedAt); err != nil {
		t.Fatalf("query audit log: %v", err)
	}
	if action != "job_completed" || completedAt != "2026-05-12 16:00:00" {
		t.Fatalf("audit row action=%s completedAt=%s", action, completedAt)
	}
}
