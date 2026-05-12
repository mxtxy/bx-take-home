package notifications

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func newTestNotificationService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	database := testutil.PrepareDB(t)
	return NewService(database, Options{Clock: func() time.Time {
		return testutil.MustTime(t, "2026-05-12T16:00:00Z")
	}}), database
}

func TestNewService_DefaultClock(t *testing.T) {
	service := NewService(nil, Options{})

	if service.clock().Location() != time.UTC {
		t.Fatalf("clock location = %v", service.clock().Location())
	}
}

func TestNotificationService_ListNotifications_ReturnsOnlyActorNotifications(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	insertNotificationRow(t, database, 1, jobID, "job_completed", "manager notification", "2026-05-12 10:00:00")
	insertNotificationRow(t, database, 3, jobID, "job_assigned", "technician notification", "2026-05-12 11:00:00")

	notifications, err := service.ListNotifications(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notifications) != 1 || notifications[0].Message != "manager notification" {
		t.Fatalf("notifications = %#v", notifications)
	}
	if notifications[0].OrganizationID != 1 {
		t.Fatalf("organization id = %d", notifications[0].OrganizationID)
	}
}

func TestNotificationService_UnreadCountCountsOnlyActorUnreadNotifications(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	readID := insertNotificationRow(t, database, 1, jobID, "job_completed", "read", "2026-05-12 10:00:00")
	insertNotificationRow(t, database, 1, jobID, "job_completed", "unread", "2026-05-12 11:00:00")
	insertNotificationRow(t, database, 3, jobID, "job_assigned", "other user", "2026-05-12 12:00:00")
	if _, err := database.Exec(`UPDATE notifications SET read_at = '2026-05-12 15:00:00' WHERE id = ?`, readID); err != nil {
		t.Fatalf("set read_at: %v", err)
	}

	count, err := service.UnreadCount(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count = %d", count)
	}
}

func TestNotificationService_UnreadCount_ReturnsQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("count failed"))
	service := NewService(database, Options{})

	_, err = service.UnreadCount(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected count error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_ListNotifications_SortedNewestFirst(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	insertNotificationRow(t, database, 1, jobID, "job_completed", "A", "2026-05-12 10:00:00")
	insertNotificationRow(t, database, 1, jobID, "job_completed", "B", "2026-05-12 11:00:00")
	insertNotificationRow(t, database, 1, jobID, "job_completed", "C", "2026-05-12 11:00:00")

	notifications, err := service.ListNotifications(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	got := []string{notifications[0].Message, notifications[1].Message, notifications[2].Message}
	want := []string{"C", "B", "A"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestNotificationService_ListNotifications_EmptyResultReturnsEmptySlice(t *testing.T) {
	service, _ := newTestNotificationService(t)

	notifications, err := service.ListNotifications(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if notifications == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(notifications) != 0 {
		t.Fatalf("len = %d", len(notifications))
	}
}

func TestNotificationService_ListNotifications_IncludesReadAt(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	notificationID := insertNotificationRow(t, database, 1, jobID, "job_completed", "manager notification", "2026-05-12 10:00:00")
	if _, err := database.Exec(`UPDATE notifications SET read_at = '2026-05-12 15:00:00' WHERE id = ?`, notificationID); err != nil {
		t.Fatalf("set read_at: %v", err)
	}

	notifications, err := service.ListNotifications(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notifications) != 1 || notifications[0].ReadAt == nil {
		t.Fatalf("notifications = %#v", notifications)
	}
}

func TestNotificationService_ListNotifications_ReturnsQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT id, organization_id").WillReturnError(errors.New("query failed"))
	service := NewService(database, Options{})

	_, err = service.ListNotifications(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected query error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_ListNotifications_ReturnsScanError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	rows := sqlmock.NewRows([]string{"id", "organization_id", "type", "message", "job_id", "read_at", "created_at"}).
		AddRow("bad-id", int64(1), "job_assigned", "message", nil, nil, time.Now())
	mock.ExpectQuery("SELECT id, organization_id").WillReturnRows(rows)
	service := NewService(database, Options{})

	_, err = service.ListNotifications(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_MarkRead_Success(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	notificationID := insertNotificationRow(t, database, 1, jobID, "job_completed", "manager notification", "2026-05-12 10:00:00")

	notification, err := service.MarkRead(context.Background(), testutil.Manager1(), notificationID)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if notification.ReadAt == nil || !notification.ReadAt.Equal(testutil.MustTime(t, "2026-05-12T16:00:00Z")) {
		t.Fatalf("readAt = %#v", notification.ReadAt)
	}
}

func TestNotificationService_MarkRead_AlreadyReadIsIdempotent(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	notificationID := insertNotificationRow(t, database, 1, jobID, "job_completed", "manager notification", "2026-05-12 10:00:00")
	if _, err := database.Exec(`UPDATE notifications SET read_at = '2026-05-12 15:00:00' WHERE id = ?`, notificationID); err != nil {
		t.Fatalf("set read_at: %v", err)
	}

	notification, err := service.MarkRead(context.Background(), testutil.Manager1(), notificationID)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if notification.ReadAt == nil || !notification.ReadAt.Equal(testutil.MustTime(t, "2026-05-12T15:00:00Z")) {
		t.Fatalf("readAt = %#v", notification.ReadAt)
	}
}

func TestNotificationService_MarkRead_WrongUserForbidden(t *testing.T) {
	service, database := newTestNotificationService(t)
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	notificationID := insertNotificationRow(t, database, 1, jobID, "job_completed", "manager notification", "2026-05-12 10:00:00")

	_, err := service.MarkRead(context.Background(), testutil.Technician1(), notificationID)
	if code := domain.CodeOf(err); code != domain.ErrorForbidden {
		t.Fatalf("code = %v, err = %v", code, err)
	}
	if got := nullableTimeString(t, database, notificationID); got != "" {
		t.Fatalf("read_at changed: %s", got)
	}
}

func TestNotificationService_MarkRead_MissingNotification(t *testing.T) {
	service, _ := newTestNotificationService(t)
	_, err := service.MarkRead(context.Background(), testutil.Manager1(), 999)
	if code := domain.CodeOf(err); code != domain.ErrorNotFound {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestNotificationService_MarkRead_InvalidID(t *testing.T) {
	service, _ := newTestNotificationService(t)
	_, err := service.MarkRead(context.Background(), testutil.Manager1(), 0)
	if code := domain.CodeOf(err); code != domain.ErrorInvalidInput {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestNotificationService_MarkRead_ReturnsBeginError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
	service := NewService(database, Options{})

	_, err = service.MarkRead(context.Background(), testutil.Manager1(), 1)
	if err == nil {
		t.Fatal("expected begin error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_MarkRead_ReturnsSelectError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, organization_id").WithArgs(int64(1)).WillReturnError(errors.New("select failed"))
	service := NewService(database, Options{})

	_, err = service.MarkRead(context.Background(), testutil.Manager1(), 1)
	if err == nil {
		t.Fatal("expected select error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_MarkRead_ReturnsUpdateError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, organization_id").WithArgs(int64(1)).WillReturnRows(notificationRows(false))
	mock.ExpectExec("UPDATE notifications SET read_at").WillReturnError(errors.New("update failed"))
	service := NewService(database, Options{Clock: testutil.FixedTime})

	_, err = service.MarkRead(context.Background(), testutil.Manager1(), 1)
	if err == nil {
		t.Fatal("expected update error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestNotificationService_MarkRead_ReturnsCommitError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, organization_id").WithArgs(int64(1)).WillReturnRows(notificationRows(false))
	mock.ExpectExec("UPDATE notifications SET read_at").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	service := NewService(database, Options{Clock: testutil.FixedTime})

	_, err = service.MarkRead(context.Background(), testutil.Manager1(), 1)
	if err == nil {
		t.Fatal("expected commit error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func insertNotificationRow(t *testing.T, database *sql.DB, recipientUserID int64, jobID int64, notificationType string, message string, createdAt string) int64 {
	t.Helper()
	result, err := database.Exec(`
		INSERT INTO notifications (organization_id, recipient_user_id, actor_user_id, job_id, type, message, created_at)
		VALUES (1, ?, 1, ?, ?, ?, ?)
	`, recipientUserID, jobID, notificationType, message, createdAt)
	if err != nil {
		t.Fatalf("insert notification: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func nullableTimeString(t *testing.T, database *sql.DB, notificationID int64) string {
	t.Helper()
	var value sql.NullString
	if err := database.QueryRow(`SELECT DATE_FORMAT(read_at, '%Y-%m-%d %H:%i:%s') FROM notifications WHERE id = ?`, notificationID).Scan(&value); err != nil {
		t.Fatalf("read read_at: %v", err)
	}
	if !value.Valid {
		return ""
	}
	return value.String
}

func notificationRows(read bool) *sqlmock.Rows {
	var readAt any
	if read {
		readAt = testutil.FixedTime()
	}
	return sqlmock.NewRows([]string{"id", "organization_id", "recipient_user_id", "type", "message", "job_id", "read_at", "created_at"}).
		AddRow(int64(1), int64(1), int64(1), "job_completed", "message", int64(1), readAt, testutil.FixedTime())
}
