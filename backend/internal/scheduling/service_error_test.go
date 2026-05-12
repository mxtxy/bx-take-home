package scheduling

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestNewService_DefaultClockAndHelpers(t *testing.T) {
	service := NewService(nil, Options{})

	if service.clock().Location() != time.UTC {
		t.Fatalf("clock location = %v", service.clock().Location())
	}
	if nullableInt64(nil) != nil {
		t.Fatal("nil pointer should become nil SQL value")
	}
	if got := itoa(0); got != "0" {
		t.Fatalf("itoa(0) = %q", got)
	}
}

func TestAssignJob_ReturnsBeginError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
	if err == nil {
		t.Fatal("expected begin error")
	}
	assertSchedulingMock(t, mock)
}

func TestAssignJob_ReturnsInsertErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode domain.ErrorCode
	}{
		{"duplicate quote", &mysql.MySQLError{Number: 1062}, domain.ErrorQuoteAlreadyScheduled},
		{"generic insert", errors.New("insert failed"), domain.ErrorInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newMockSchedulingService(t)
			expectAssignBeforeInsert(mock)
			mock.ExpectExec("INSERT INTO jobs").WillReturnError(tt.err)

			_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
			if code := domain.CodeOf(err); code != tt.wantCode {
				t.Fatalf("code = %v, err = %v", code, err)
			}
			assertSchedulingMock(t, mock)
		})
	}
}

func TestAssignJob_ReturnsLastInsertIDError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectAssignBeforeInsert(mock)
	mock.ExpectExec("INSERT INTO jobs").WillReturnResult(sqlmock.NewErrorResult(errors.New("last id failed")))

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
	if err == nil {
		t.Fatal("expected last insert id error")
	}
	assertSchedulingMock(t, mock)
}

func TestAssignJob_ReturnsQuoteUpdateError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectAssignBeforeInsert(mock)
	mock.ExpectExec("INSERT INTO jobs").WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec("UPDATE quotes SET status").WillReturnError(errors.New("update failed"))

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
	if err == nil {
		t.Fatal("expected quote update error")
	}
	assertSchedulingMock(t, mock)
}

func TestAssignJob_ReturnsNotificationError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectAssignBeforeInsert(mock)
	mock.ExpectExec("INSERT INTO jobs").WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec("UPDATE quotes SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnError(errors.New("notification failed"))

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
	if err == nil {
		t.Fatal("expected notification error")
	}
	assertSchedulingMock(t, mock)
}

func TestAssignJob_ReturnsCommitError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectAssignBeforeInsert(mock)
	mock.ExpectExec("INSERT INTO jobs").WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec("UPDATE quotes SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnResult(sqlmock.NewResult(9, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	_, _, err := service.AssignJob(context.Background(), testutil.Manager1(), validAssignInput())
	if err == nil {
		t.Fatal("expected commit error")
	}
	assertSchedulingMock(t, mock)
}

func TestRescheduleJob_ReturnsBeginError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), validRescheduleInput())
	if err == nil {
		t.Fatal("expected begin error")
	}
	assertSchedulingMock(t, mock)
}

func TestRescheduleJob_ReturnsUpdateError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectRescheduleBeforeUpdate(mock, jobRowOptions{})
	mock.ExpectExec("UPDATE jobs").WillReturnError(errors.New("update failed"))

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), validRescheduleInput())
	if err == nil {
		t.Fatal("expected update error")
	}
	assertSchedulingMock(t, mock)
}

func TestRescheduleJob_ReturnsTargetNotificationError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectRescheduleBeforeUpdate(mock, jobRowOptions{})
	mock.ExpectExec("UPDATE jobs").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnError(errors.New("notification failed"))

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), validRescheduleInput())
	if err == nil {
		t.Fatal("expected notification error")
	}
	assertSchedulingMock(t, mock)
}

func TestRescheduleJob_ReturnsPreviousTechnicianNotificationError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectRescheduleBeforeUpdate(mock, jobRowOptions{technicianID: 1, technicianUserID: 3})
	mock.ExpectExec("UPDATE jobs").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnError(errors.New("previous notification failed"))

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), validRescheduleInput())
	if err == nil {
		t.Fatal("expected previous technician notification error")
	}
	assertSchedulingMock(t, mock)
}

func TestRescheduleJob_ReturnsCommitError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectRescheduleBeforeUpdate(mock, jobRowOptions{technicianID: 2, technicianUserID: 4})
	mock.ExpectExec("UPDATE jobs").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnResult(sqlmock.NewResult(11, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	_, _, err := service.RescheduleJob(context.Background(), testutil.Manager1(), validRescheduleInput())
	if err == nil {
		t.Fatal("expected commit error")
	}
	assertSchedulingMock(t, mock)
}

func TestCompleteJob_ReturnsBeginError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 1})
	if err == nil {
		t.Fatal("expected begin error")
	}
	assertSchedulingMock(t, mock)
}

func TestCompleteJob_ReturnsUpdateError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectCompleteBeforeUpdate(mock)
	mock.ExpectExec("UPDATE jobs SET status").WillReturnError(errors.New("update failed"))

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 1})
	if err == nil {
		t.Fatal("expected update error")
	}
	assertSchedulingMock(t, mock)
}

func TestCompleteJob_ReturnsNotificationError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectCompleteBeforeUpdate(mock)
	mock.ExpectExec("UPDATE jobs SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnError(errors.New("notification failed"))

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 1})
	if err == nil {
		t.Fatal("expected notification error")
	}
	assertSchedulingMock(t, mock)
}

func TestCompleteJob_ReturnsCommitError(t *testing.T) {
	service, mock := newMockSchedulingService(t)
	expectCompleteBeforeUpdate(mock)
	mock.ExpectExec("UPDATE jobs SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notifications").WillReturnResult(sqlmock.NewResult(13, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	_, _, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: 1})
	if err == nil {
		t.Fatal("expected commit error")
	}
	assertSchedulingMock(t, mock)
}

func TestSelectHelpers_ReturnGenericErrors(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *sql.Tx) error
	}{
		{
			name: "quote",
			run: func(ctx context.Context, tx *sql.Tx) error {
				_, err := selectQuoteForUpdate(ctx, tx, 1)
				return err
			},
		},
		{
			name: "technician",
			run: func(ctx context.Context, tx *sql.Tx) error {
				_, err := selectTechnicianForUpdate(ctx, tx, 1)
				return err
			},
		},
		{
			name: "job",
			run: func(ctx context.Context, tx *sql.Tx) error {
				_, err := selectJobForUpdate(ctx, tx, 1)
				return err
			},
		},
		{
			name: "overlap",
			run: func(ctx context.Context, tx *sql.Tx) error {
				return ensureNoOverlap(ctx, tx, 1, testutil.FixedTime(), testutil.FixedTime().Add(2*time.Hour), nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, mock, tx := beginMockTx(t)
			defer database.Close()
			mock.ExpectQuery("SELECT").WillReturnError(errors.New("select failed"))

			if err := tt.run(context.Background(), tx); err == nil {
				t.Fatal("expected select error")
			}
			assertSchedulingMock(t, mock)
		})
	}
}

func validAssignInput() domain.AssignJobInput {
	return domain.AssignJobInput{QuoteID: 1, TechnicianID: 1, StartsAt: testutil.FixedTime()}
}

func validRescheduleInput() domain.RescheduleJobInput {
	return domain.RescheduleJobInput{JobID: 1, TechnicianID: 2, StartsAt: testutil.FixedTime().Add(2 * time.Hour)}
}

func newMockSchedulingService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return NewService(database, Options{Clock: testutil.FixedTime}), mock
}

func beginMockTx(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *sql.Tx) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	mock.ExpectBegin()
	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	return database, mock, tx
}

func assertSchedulingMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func expectAssignBeforeInsert(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	expectQuote(mock, domain.QuoteUnscheduled)
	expectTechnician(mock, 1, 3)
	expectNoOverlap(mock)
}

type jobRowOptions struct {
	technicianID     int64
	technicianUserID int64
}

func expectRescheduleBeforeUpdate(mock sqlmock.Sqlmock, options jobRowOptions) {
	if options.technicianID == 0 {
		options.technicianID = 1
	}
	if options.technicianUserID == 0 {
		options.technicianUserID = 3
	}
	mock.ExpectBegin()
	expectJob(mock, domain.JobScheduled, options.technicianID, options.technicianUserID)
	expectTechnician(mock, 2, 4)
	expectNoOverlap(mock)
}

func expectCompleteBeforeUpdate(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	expectJob(mock, domain.JobScheduled, 1, 3)
}

func expectQuote(mock sqlmock.Sqlmock, status domain.QuoteStatus) {
	rows := sqlmock.NewRows([]string{"id", "customer_name", "description", "status"}).
		AddRow(int64(1), "Acme Plumbing", "Replace tap", string(status))
	mock.ExpectQuery("SELECT id, customer_name").WithArgs(int64(1)).WillReturnRows(rows)
}

func expectTechnician(mock sqlmock.Sqlmock, technicianID int64, userID int64) {
	rows := sqlmock.NewRows([]string{"id", "user_id", "display_name"}).
		AddRow(technicianID, userID, "Tom Technician")
	mock.ExpectQuery("SELECT t.id, t.user_id").WithArgs(technicianID).WillReturnRows(rows)
}

func expectJob(mock sqlmock.Sqlmock, status domain.JobStatus, technicianID int64, technicianUserID int64) {
	rows := sqlmock.NewRows([]string{
		"id", "quote_id", "customer_name", "description",
		"technician_id", "technician_user_id", "technician_name",
		"manager_id", "manager_user_id", "manager_name",
		"starts_at", "ends_at", "status", "completed_at",
	}).AddRow(
		int64(1), int64(1), "Acme Plumbing", "Replace tap",
		technicianID, technicianUserID, "Tom Technician",
		int64(1), int64(1), "Sarah Manager",
		testutil.FixedTime(), testutil.FixedTime().Add(2*time.Hour), string(status), nil,
	)
	mock.ExpectQuery("SELECT").WithArgs(int64(1)).WillReturnRows(rows)
}

func expectNoOverlap(mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(rows)
}
