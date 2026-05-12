package scheduling

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func newTestSchedulingService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database := testutil.PrepareDB(t)
	return NewService(database, Options{Clock: testutil.FixedTime}), database
}

func TestSchedulingService_AssignJob_Success(t *testing.T) {
	service, database := newTestSchedulingService(t)
	input := domain.AssignJobInput{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	job, events, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	if err != nil {
		t.Fatalf("assign job: %v", err)
	}
	if job.QuoteID != 1 || job.TechnicianID != 1 || job.ManagerID != 1 || job.Status != domain.JobScheduled || job.CompletedAt != nil {
		t.Fatalf("job = %#v", job)
	}
	if !job.StartsAt.Equal(input.StartsAt) || !job.EndsAt.Equal(testutil.MustTime(t, "2026-05-12T12:00:00Z")) {
		t.Fatalf("window = %s-%s", job.StartsAt, job.EndsAt)
	}
	if countRows(t, database, `SELECT COUNT(*) FROM jobs`) != 1 {
		t.Fatal("expected one job")
	}
	if got := scalarString(t, database, `SELECT status FROM quotes WHERE id = 1`); got != "scheduled" {
		t.Fatalf("quote status = %s", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = 3 AND type = 'job_assigned'`); got != 1 {
		t.Fatalf("notification count = %d", got)
	}
	assertHasEvent(t, events, "notification.created", int64Ptr(3), nil)
	assertHasEvent(t, events, "jobs.changed", int64Ptr(3), nil)
	assertHasEvent(t, events, "quotes.changed", nil, rolePtr(domain.RoleManager))
	assertHasEvent(t, events, "jobs.changed", int64Ptr(1), nil)
}

func TestSchedulingService_AssignJob_TechnicianActorForbidden(t *testing.T) {
	service, database := newTestSchedulingService(t)
	input := domain.AssignJobInput{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, events, err := service.AssignJob(context.Background(), testutil.Technician1(), input)
	assertCode(t, err, domain.ErrorForbidden)
	assertNoMutation(t, database)
	if len(events) != 0 {
		t.Fatalf("events = %#v", events)
	}
}

func TestSchedulingService_AssignJob_MalformedManagerActorForbidden(t *testing.T) {
	service, database := newTestSchedulingService(t)
	actor := testutil.Manager1()
	actor.ManagerID = nil
	input := domain.AssignJobInput{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), actor, input)
	assertCode(t, err, domain.ErrorForbidden)
	assertNoMutation(t, database)
}

func TestSchedulingService_AssignJob_InvalidInput(t *testing.T) {
	tests := []domain.AssignJobInput{
		{QuoteID: 0, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")},
		{QuoteID: 1, TechnicianID: 0, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")},
		{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:01Z")},
	}
	for _, input := range tests {
		t.Run("", func(t *testing.T) {
			service, database := newTestSchedulingService(t)
			_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
			assertCode(t, err, domain.ErrorInvalidInput)
			assertNoMutation(t, database)
		})
	}
}

func TestSchedulingService_AssignJob_MissingQuote(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	input := domain.AssignJobInput{QuoteID: 999, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_AssignJob_MissingTechnician(t *testing.T) {
	service, _ := newTestSchedulingService(t)
	input := domain.AssignJobInput{QuoteID: 1, TechnicianID: 999, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	assertCode(t, err, domain.ErrorNotFound)
}

func TestSchedulingService_AssignJob_QuoteAlreadyScheduled(t *testing.T) {
	service, database := newTestSchedulingService(t)
	input := domain.AssignJobInput{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}
	if _, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input); err != nil {
		t.Fatalf("first assign: %v", err)
	}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	assertCode(t, err, domain.ErrorQuoteAlreadyScheduled)
	if got := countRows(t, database, `SELECT COUNT(*) FROM jobs WHERE quote_id = 1`); got != 1 {
		t.Fatalf("jobs for quote = %d", got)
	}
}

func TestSchedulingService_AssignJob_OverlapConflict(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	input := domain.AssignJobInput{QuoteID: 2, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T11:00:00Z")}

	_, events, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	assertCode(t, err, domain.ErrorScheduleConflict)
	if got := countRows(t, database, `SELECT COUNT(*) FROM jobs`); got != 1 {
		t.Fatalf("job count = %d", got)
	}
	if got := scalarString(t, database, `SELECT status FROM quotes WHERE id = 2`); got != "unscheduled" {
		t.Fatalf("quote 2 status = %s", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications`); got != 0 {
		t.Fatalf("notification count = %d", got)
	}
	if len(events) != 0 {
		t.Fatalf("events = %#v", events)
	}
}

func TestSchedulingService_AssignJob_ExactOverlapConflict(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	input := domain.AssignJobInput{QuoteID: 2, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	assertCode(t, err, domain.ErrorScheduleConflict)
}

func TestSchedulingService_AssignJob_BoundaryBeforeAllowed(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	input := domain.AssignJobInput{QuoteID: 2, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T08:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
}

func TestSchedulingService_AssignJob_BoundaryAfterAllowed(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	input := domain.AssignJobInput{QuoteID: 2, TechnicianID: 1, StartsAt: testutil.MustTime(t, "2026-05-12T12:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
}

func TestSchedulingService_AssignJob_DifferentTechnicianSameWindowAllowed(t *testing.T) {
	service, database := newTestSchedulingService(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	input := domain.AssignJobInput{QuoteID: 2, TechnicianID: 2, StartsAt: testutil.MustTime(t, "2026-05-12T10:00:00Z")}

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), input)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
}

func TestSchedulingService_AssignJob_ConcurrentOverlapExactlyOneSuccess(t *testing.T) {
	service, database := newTestSchedulingService(t)
	start := make(chan struct{})
	results := make(chan domain.ErrorCode, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	run := func(actor domain.Actor, quoteID int64, startsAt string) {
		defer wg.Done()
		<-start
		_, _, err := service.AssignJob(context.Background(), actor, domain.AssignJobInput{
			QuoteID:      quoteID,
			TechnicianID: 1,
			StartsAt:     testutil.MustTime(t, startsAt),
		})
		results <- domain.CodeOf(err)
	}
	go run(testutil.Manager1(), 1, "2026-05-12T10:00:00Z")
	go run(testutil.Manager2(), 2, "2026-05-12T11:00:00Z")
	close(start)
	wg.Wait()
	close(results)

	assertResultCounts(t, results, map[domain.ErrorCode]int{"": 1, domain.ErrorScheduleConflict: 1})
	if got := countRows(t, database, `
		SELECT COUNT(*) FROM jobs
		WHERE technician_id = 1 AND starts_at < '2026-05-12 13:00:00' AND ends_at > '2026-05-12 10:00:00'
	`); got != 1 {
		t.Fatalf("overlapping jobs = %d", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications WHERE recipient_user_id = 3`); got != 1 {
		t.Fatalf("notification count = %d", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM quotes WHERE status = 'scheduled'`); got != 1 {
		t.Fatalf("scheduled quotes = %d", got)
	}
}

func TestSchedulingService_AssignJob_ConcurrentSameQuoteExactlyOneSuccess(t *testing.T) {
	service, database := newTestSchedulingService(t)
	start := make(chan struct{})
	results := make(chan domain.ErrorCode, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	run := func(actor domain.Actor, technicianID int64) {
		defer wg.Done()
		<-start
		_, _, err := service.AssignJob(context.Background(), actor, domain.AssignJobInput{
			QuoteID:      1,
			TechnicianID: technicianID,
			StartsAt:     testutil.MustTime(t, "2026-05-12T10:00:00Z"),
		})
		results <- domain.CodeOf(err)
	}
	go run(testutil.Manager1(), 1)
	go run(testutil.Manager2(), 2)
	close(start)
	wg.Wait()
	close(results)

	assertResultCounts(t, results, map[domain.ErrorCode]int{"": 1, domain.ErrorQuoteAlreadyScheduled: 1})
	if got := countRows(t, database, `SELECT COUNT(*) FROM jobs WHERE quote_id = 1`); got != 1 {
		t.Fatalf("jobs for quote 1 = %d", got)
	}
}

func assertCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	if code := domain.CodeOf(err); code != want {
		t.Fatalf("code = %v, want %v, err = %v", code, want, err)
	}
}

func assertNoMutation(t *testing.T, database *sql.DB) {
	t.Helper()
	if got := countRows(t, database, `SELECT COUNT(*) FROM jobs`); got != 0 {
		t.Fatalf("job count = %d", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM notifications`); got != 0 {
		t.Fatalf("notification count = %d", got)
	}
	if got := countRows(t, database, `SELECT COUNT(*) FROM quotes WHERE status = 'scheduled'`); got != 0 {
		t.Fatalf("scheduled quote count = %d", got)
	}
}

func countRows(t *testing.T, database *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := database.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}

func scalarString(t *testing.T, database *sql.DB, query string, args ...any) string {
	t.Helper()
	var value string
	if err := database.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("scalar string: %v", err)
	}
	return value
}

func assertHasEvent(t *testing.T, events []domain.DomainEvent, eventType string, userID *int64, role *domain.Role) {
	t.Helper()
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		if userID != nil && (event.TargetUserID == nil || *event.TargetUserID != *userID) {
			continue
		}
		if role != nil && (event.TargetRole == nil || *event.TargetRole != *role) {
			continue
		}
		if userID == nil && role == nil {
			return
		}
		if userID != nil || role != nil {
			return
		}
	}
	t.Fatalf("missing event type=%s user=%v role=%v in %#v", eventType, userID, role, events)
}

func rolePtr(value domain.Role) *domain.Role {
	return &value
}

func assertResultCounts(t *testing.T, results <-chan domain.ErrorCode, want map[domain.ErrorCode]int) {
	t.Helper()
	got := map[domain.ErrorCode]int{}
	for code := range results {
		got[code]++
	}
	for code, count := range want {
		if got[code] != count {
			t.Fatalf("result counts = %#v, want %#v", got, want)
		}
	}
}
