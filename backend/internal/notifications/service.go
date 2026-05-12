package notifications

import (
	"context"
	"database/sql"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type Options struct {
	Clock func() time.Time
}

type Service struct {
	db    *sql.DB
	clock func() time.Time
}

func NewService(database *sql.DB, options Options) *Service {
	if options.Clock == nil {
		options.Clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{db: database, clock: options.Clock}
}

func (s *Service) ListNotifications(ctx context.Context, actor domain.Actor) ([]domain.NotificationDTO, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, message, job_id, read_at, created_at
		FROM notifications
		WHERE recipient_user_id = ?
		ORDER BY created_at DESC, id DESC
	`, actor.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.NotificationDTO, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, notification)
	}
	return result, rows.Err()
}

func (s *Service) MarkRead(ctx context.Context, actor domain.Actor, notificationID int64) (domain.NotificationDTO, error) {
	if notificationID <= 0 {
		return domain.NotificationDTO{}, domain.Errorf(domain.ErrorInvalidInput)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.NotificationDTO{}, err
	}
	defer tx.Rollback()

	var recipientUserID int64
	var notification domain.NotificationDTO
	var typeValue string
	var jobID sql.NullInt64
	var readAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, recipient_user_id, type, message, job_id, read_at, created_at
		FROM notifications
		WHERE id = ?
		FOR UPDATE
	`, notificationID).Scan(
		&notification.ID,
		&recipientUserID,
		&typeValue,
		&notification.Message,
		&jobID,
		&readAt,
		&notification.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return domain.NotificationDTO{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return domain.NotificationDTO{}, err
	}
	if recipientUserID != actor.UserID {
		return domain.NotificationDTO{}, domain.Errorf(domain.ErrorForbidden)
	}
	notification.Type = domain.NotificationType(typeValue)
	if jobID.Valid {
		value := jobID.Int64
		notification.JobID = &value
	}
	if readAt.Valid {
		value := readAt.Time
		notification.ReadAt = &value
	} else {
		now := s.clock().UTC()
		if _, err := tx.ExecContext(ctx, `UPDATE notifications SET read_at = ? WHERE id = ?`, now, notificationID); err != nil {
			return domain.NotificationDTO{}, err
		}
		notification.ReadAt = &now
	}
	if err := tx.Commit(); err != nil {
		return domain.NotificationDTO{}, err
	}
	return notification, nil
}

type notificationScanner interface {
	Scan(dest ...any) error
}

func scanNotification(scanner notificationScanner) (domain.NotificationDTO, error) {
	var notification domain.NotificationDTO
	var typeValue string
	var jobID sql.NullInt64
	var readAt sql.NullTime
	if err := scanner.Scan(&notification.ID, &typeValue, &notification.Message, &jobID, &readAt, &notification.CreatedAt); err != nil {
		return domain.NotificationDTO{}, err
	}
	notification.Type = domain.NotificationType(typeValue)
	if jobID.Valid {
		value := jobID.Int64
		notification.JobID = &value
	}
	if readAt.Valid {
		value := readAt.Time
		notification.ReadAt = &value
	}
	return notification, nil
}
