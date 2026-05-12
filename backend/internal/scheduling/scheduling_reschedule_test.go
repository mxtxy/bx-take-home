package scheduling

import (
	"context"
	"testing"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestSchedulingService_RescheduleJob_SuccessSameTechnician(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T00:00:00Z")

	job, events, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 1,
		StartsAt:     testutil.MustTime(t, "2026-05-12T04:00:00Z"),
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if job.TechnicianID != 1 || !job.StartsAt.Equal(testutil.MustTime(t, "2026-05-12T04:00:00Z")) || !job.EndsAt.Equal(testutil.MustTime(t, "2026-05-12T06:00:00Z")) {
		t.Fatalf("job = %#v", job)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications WHERE type = 'job_updated' AND recipient_user_id = 3`); got != 1 {
		t.Fatalf("target notifications = %d", got)
	}
	assertHasEvent(t, events, "jobs.changed", int64Ptr(3), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(1), nil)
}

func TestSchedulingService_RescheduleJob_SuccessDifferentTechnician(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T00:00:00Z")

	job, events, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 2,
		StartsAt:     testutil.MustTime(t, "2026-05-12T04:00:00Z"),
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if job.TechnicianID != 2 {
		t.Fatalf("job = %#v", job)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications WHERE type = 'job_updated'`); got != 2 {
		t.Fatalf("updated notification count = %d", got)
	}
	assertHasEvent(t, events, "notification.created", int64Ptr(3), nil)
	assertHasEvent(t, events, "notification.created", int64Ptr(4), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(3), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(4), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(1), nil)
}

func TestSchedulingService_RescheduleJob_ManagerCannotUpdateOtherManagersJob(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, events, err := service.RescheduleJob(context.Background(), testutil.Manager2(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 1,
		StartsAt:     testutil.MustTime(t, "2026-05-12T04:00:00Z"),
	})
	assertCode(t, err, domain.ErrorForbidden)
	if got := scalarString(t, database, `SELECT DATE_FORMAT(starts_at, '%Y-%m-%d %H') FROM jobs WHERE id = ?`, jobID); got != "2026-05-12 10" {
		t.Fatalf("starts_at changed: %s", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications`); got != 0 {
		t.Fatalf("notification count = %d", got)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v", events)
	}
}

func TestSchedulingService_RescheduleJob_TechnicianActorForbidden(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, _, err := service.RescheduleJob(context.Background(), testutil.Technician1(), domain.RescheduleJobInput{JobID: jobID, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T14:00:00Z")})
	assertCode(t, err, domain.ErrorForbidden)
}

func TestSchedulingService_RescheduleJob_InvalidInput(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	tests := []domain.RescheduleJobInput{
		{JobID: 0, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:00Z")},
		{JobID: 1, TechnicianID: 0, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:00Z")},
		{JobID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:01Z")},
	}
	for _, input := range tests {
		_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), input)
		assertCode(t, err, domain.ErrorInvalidInput)
	}
}

func TestSchedulingService_RescheduleJob_MissingJob(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: 999, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:00Z")})
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_RescheduleJob_MissingTargetTechnician(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: jobID, TechnicianID: 999, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:00Z")})
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_RescheduleJob_CompletedJobImmutable(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertCompletedJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z", "2026-05-12T15:00:00Z")
	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: jobID, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T04:00:00Z")})
	assertCode(t, err, domain.ErrorCompletedJobImmutable)
}

func TestSchedulingService_RescheduleJob_OverlapConflict(t *testing.T) {
	service, database := newTestSchedulingService(t)
	job1 := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-11T22:00:00Z")
	testutil.InsertScheduledJob(t, database, 2, 1, 1, "2026-05-12T00:00:00Z")

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: job1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T01:00:00Z")})
	assertCode(t, err, domain.ErrorScheduleConflict)
	if got := scalarString(t, database, `SELECT DATE_FORMAT(starts_at, '%Y-%m-%d %H') FROM jobs WHERE id = ?`, job1); got != "2026-05-11 22" {
		t.Fatalf("job changed: %s", got)
	}
}

func TestSchedulingService_RescheduleJob_ExcludesCurrentJobFromOverlapCheck(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T00:00:00Z")
	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: jobID, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T00:00:00Z")})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
}

func TestSchedulingService_RescheduleJob_TargetTechnicianBoundaryAllowed(t *testing.T) {
	service, database := newTestSchedulingService(t)
	job1 := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-11T22:00:00Z")
	testutil.InsertScheduledJob(t, database, 2, 1, 1, "2026-05-12T00:00:00Z")
	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{JobID: job1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T02:00:00Z")})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
}

func TestSchedulingService_RescheduleJob_RejectsUnavailableWindow(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T00:00:00Z")

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 1,
		StartsAt:     testutil.MustTime(t, "2026-05-12T08:00:00Z"),
	})
	assertCode(t, err, domain.ErrorTechnicianUnavailable)
	if got := scalarString(t, database, `SELECT DATE_FORMAT(starts_at, '%Y-%m-%d %H') FROM jobs WHERE id = ?`, jobID); got != "2026-05-12 00" {
		t.Fatalf("job changed: %s", got)
	}
}

func TestSchedulingService_RescheduleJob_RejectsCrossOrganizationActor(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager3(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 3,
		StartsAt:     testutil.MustTime(t, "2026-05-12T00:00:00Z"),
	})
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_RescheduleJob_WritesAuditLog(t *testing.T) {
	service, database := newTestSchedulingService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T00:00:00Z")

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: 2,
		StartsAt:     testutil.MustTime(t, "2026-05-12T04:00:00Z"),
	})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}

	var action string
	var previousTechnicianID, newTechnicianID int64
	var previousStartsAt, newStartsAt string
	if err := database.QueryRow(`
		SELECT action, previous_technician_id, new_technician_id,
		       DATE_FORMAT(previous_starts_at, '%Y-%m-%d %H:%i:%s'),
		       DATE_FORMAT(new_starts_at, '%Y-%m-%d %H:%i:%s')
		FROM schedule_audit_logs
		WHERE job_id = ? AND action = 'job_rescheduled'
	`, jobID).Scan(&action, &previousTechnicianID, &newTechnicianID, &previousStartsAt, &newStartsAt); err != nil {
		t.Fatalf("query audit log: %v", err)
	}
	if action != "job_rescheduled" || previousTechnicianID != 1 || newTechnicianID != 2 || previousStartsAt != "2026-05-12 00:00:00" || newStartsAt != "2026-05-12 04:00:00" {
		t.Fatalf("audit row action=%s previousTech=%d newTech=%d previousStart=%s newStart=%s", action, previousTechnicianID, newTechnicianID, previousStartsAt, newStartsAt)
	}
}
