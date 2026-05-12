package scheduling

import (
	"context"
	"database/sql"
	"time"

	dbutil "github.com/mxtxy/bx-take-home/backend/internal/db"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type Options struct {
	Clock func() time.Time
}

type Service struct {
	db    *sql.DB
	clock func() time.Time
}

type quoteRow struct {
	ID           int64
	Status       domain.QuoteStatus
	CustomerName string
	Description  string
}

type technicianRow struct {
	ID          int64
	UserID      int64
	DisplayName string
}

type jobRow struct {
	ID                int64
	QuoteID           int64
	QuoteCustomerName string
	QuoteDescription  string
	TechnicianID      int64
	TechnicianUserID  int64
	TechnicianName    string
	ManagerID         int64
	ManagerUserID     int64
	ManagerName       string
	StartsAt          time.Time
	EndsAt            time.Time
	Status            domain.JobStatus
	CompletedAt       *time.Time
}

func NewService(database *sql.DB, options Options) *Service {
	if options.Clock == nil {
		options.Clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{db: database, clock: options.Clock}
}

func (s *Service) AssignJob(ctx context.Context, actor domain.Actor, input domain.AssignJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if err := domain.ValidateAssignJobInput(input); err != nil {
		return domain.JobDTO{}, nil, err
	}
	window := domain.CalculateWindow(input.StartsAt)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	defer tx.Rollback()

	quote, err := selectQuoteForUpdate(ctx, tx, input.QuoteID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if quote.Status != domain.QuoteUnscheduled {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorQuoteAlreadyScheduled)
	}

	technician, err := selectTechnicianForUpdate(ctx, tx, input.TechnicianID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureNoOverlap(ctx, tx, input.TechnicianID, window.StartsAt, window.EndsAt, nil); err != nil {
		return domain.JobDTO{}, nil, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (?, ?, ?, ?, ?, 'scheduled')
	`, input.QuoteID, input.TechnicianID, *actor.ManagerID, window.StartsAt, window.EndsAt)
	if err != nil {
		if dbutil.IsDuplicateKey(err) {
			return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorQuoteAlreadyScheduled)
		}
		return domain.JobDTO{}, nil, err
	}
	jobID, err := result.LastInsertId()
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quotes SET status = 'scheduled' WHERE id = ?`, input.QuoteID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationID, err := insertNotification(ctx, tx, technician.UserID, &actor.UserID, &jobID, domain.NotificationJobAssigned, "You have been assigned job #"+itoa(jobID)+" for "+quote.CustomerName+".")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:           jobID,
		QuoteID:      input.QuoteID,
		TechnicianID: input.TechnicianID,
		ManagerID:    *actor.ManagerID,
		StartsAt:     window.StartsAt,
		EndsAt:       window.EndsAt,
		Status:       domain.JobScheduled,
	}
	events := []domain.DomainEvent{
		notificationEvent(technician.UserID, jobID, notificationID),
		jobsChangedEvent(technician.UserID, jobID),
		quotesChangedEvent(input.QuoteID),
		jobsChangedEvent(actor.UserID, jobID),
	}
	return job, events, nil
}

func (s *Service) RescheduleJob(ctx context.Context, actor domain.Actor, input domain.RescheduleJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if err := domain.ValidateRescheduleJobInput(input); err != nil {
		return domain.JobDTO{}, nil, err
	}
	window := domain.CalculateWindow(input.StartsAt)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	defer tx.Rollback()

	current, err := selectJobForUpdate(ctx, tx, input.JobID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if current.ManagerID != *actor.ManagerID {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if current.Status == domain.JobCompleted {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorCompletedJobImmutable)
	}
	targetTechnician, err := selectTechnicianForUpdate(ctx, tx, input.TechnicianID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureNoOverlap(ctx, tx, input.TechnicianID, window.StartsAt, window.EndsAt, &input.JobID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET technician_id = ?, starts_at = ?, ends_at = ?
		WHERE id = ?
	`, input.TechnicianID, window.StartsAt, window.EndsAt, input.JobID); err != nil {
		return domain.JobDTO{}, nil, err
	}

	var notificationIDs []int64
	targetNotificationID, err := insertNotification(ctx, tx, targetTechnician.UserID, &actor.UserID, &input.JobID, domain.NotificationJobUpdated, "Job #"+itoa(input.JobID)+" has been updated.")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationIDs = append(notificationIDs, targetNotificationID)
	if current.TechnicianID != input.TechnicianID {
		previousNotificationID, err := insertNotification(ctx, tx, current.TechnicianUserID, &actor.UserID, &input.JobID, domain.NotificationJobUpdated, "Job #"+itoa(input.JobID)+" has been reassigned.")
		if err != nil {
			return domain.JobDTO{}, nil, err
		}
		notificationIDs = append(notificationIDs, previousNotificationID)
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:           input.JobID,
		QuoteID:      current.QuoteID,
		TechnicianID: input.TechnicianID,
		ManagerID:    current.ManagerID,
		StartsAt:     window.StartsAt,
		EndsAt:       window.EndsAt,
		Status:       domain.JobScheduled,
	}
	events := []domain.DomainEvent{
		notificationEvent(targetTechnician.UserID, input.JobID, notificationIDs[0]),
		jobsChangedEvent(targetTechnician.UserID, input.JobID),
	}
	if current.TechnicianID != input.TechnicianID {
		events = append(events,
			notificationEvent(current.TechnicianUserID, input.JobID, notificationIDs[1]),
			jobsChangedEvent(current.TechnicianUserID, input.JobID),
		)
	}
	events = append(events, jobsChangedEvent(actor.UserID, input.JobID))
	return job, events, nil
}

func (s *Service) CompleteJob(ctx context.Context, actor domain.Actor, input domain.CompleteJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleTechnician || actor.TechnicianID == nil {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if err := domain.ValidateCompleteJobInput(input); err != nil {
		return domain.JobDTO{}, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	defer tx.Rollback()

	current, err := selectJobForUpdate(ctx, tx, input.JobID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if current.TechnicianID != *actor.TechnicianID {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if current.Status == domain.JobCompleted {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorJobAlreadyCompleted)
	}
	completedAt := s.clock().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status = 'completed', completed_at = ? WHERE id = ?`, completedAt, input.JobID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationID, err := insertNotification(ctx, tx, current.ManagerUserID, &actor.UserID, &input.JobID, domain.NotificationJobCompleted, "Job #"+itoa(input.JobID)+" for "+current.QuoteCustomerName+" was completed.")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:           input.JobID,
		QuoteID:      current.QuoteID,
		TechnicianID: current.TechnicianID,
		ManagerID:    current.ManagerID,
		StartsAt:     current.StartsAt,
		EndsAt:       current.EndsAt,
		Status:       domain.JobCompleted,
		CompletedAt:  &completedAt,
	}
	events := []domain.DomainEvent{
		notificationEvent(current.ManagerUserID, input.JobID, notificationID),
		jobsChangedEvent(current.ManagerUserID, input.JobID),
		jobsChangedEvent(actor.UserID, input.JobID),
	}
	return job, events, nil
}

func selectQuoteForUpdate(ctx context.Context, tx *sql.Tx, quoteID int64) (quoteRow, error) {
	var quote quoteRow
	var status string
	err := tx.QueryRowContext(ctx, `
		SELECT id, customer_name, description, status
		FROM quotes
		WHERE id = ?
		FOR UPDATE
	`, quoteID).Scan(&quote.ID, &quote.CustomerName, &quote.Description, &status)
	if err == sql.ErrNoRows {
		return quoteRow{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return quoteRow{}, err
	}
	quote.Status = domain.QuoteStatus(status)
	return quote, nil
}

func selectTechnicianForUpdate(ctx context.Context, tx *sql.Tx, technicianID int64) (technicianRow, error) {
	var technician technicianRow
	err := tx.QueryRowContext(ctx, `
		SELECT t.id, t.user_id, u.display_name
		FROM technicians t
		JOIN users u ON u.id = t.user_id
		WHERE t.id = ?
		FOR UPDATE
	`, technicianID).Scan(&technician.ID, &technician.UserID, &technician.DisplayName)
	if err == sql.ErrNoRows {
		return technicianRow{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return technicianRow{}, err
	}
	return technician, nil
}

func selectJobForUpdate(ctx context.Context, tx *sql.Tx, jobID int64) (jobRow, error) {
	var job jobRow
	var status string
	var completedAt sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT
			j.id, j.quote_id, q.customer_name, q.description,
			j.technician_id, tech.user_id, tech_user.display_name,
			j.manager_id, manager.user_id, manager_user.display_name,
			j.starts_at, j.ends_at, j.status, j.completed_at
		FROM jobs j
		JOIN quotes q ON q.id = j.quote_id
		JOIN technicians tech ON tech.id = j.technician_id
		JOIN users tech_user ON tech_user.id = tech.user_id
		JOIN managers manager ON manager.id = j.manager_id
		JOIN users manager_user ON manager_user.id = manager.user_id
		WHERE j.id = ?
		FOR UPDATE
	`, jobID).Scan(
		&job.ID,
		&job.QuoteID,
		&job.QuoteCustomerName,
		&job.QuoteDescription,
		&job.TechnicianID,
		&job.TechnicianUserID,
		&job.TechnicianName,
		&job.ManagerID,
		&job.ManagerUserID,
		&job.ManagerName,
		&job.StartsAt,
		&job.EndsAt,
		&status,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return jobRow{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return jobRow{}, err
	}
	job.Status = domain.JobStatus(status)
	if completedAt.Valid {
		value := completedAt.Time
		job.CompletedAt = &value
	}
	return job, nil
}

func ensureNoOverlap(ctx context.Context, tx *sql.Tx, technicianID int64, startsAt time.Time, endsAt time.Time, excludeJobID *int64) error {
	query := `
		SELECT COUNT(*)
		FROM jobs
		WHERE technician_id = ?
		  AND starts_at < ?
		  AND ends_at > ?
	`
	args := []any{technicianID, endsAt, startsAt}
	if excludeJobID != nil {
		query += ` AND id <> ?`
		args = append(args, *excludeJobID)
	}
	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return domain.Errorf(domain.ErrorScheduleConflict)
	}
	return nil
}

func insertNotification(ctx context.Context, tx *sql.Tx, recipientUserID int64, actorUserID *int64, jobID *int64, notificationType domain.NotificationType, message string) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO notifications (recipient_user_id, actor_user_id, job_id, type, message)
		VALUES (?, ?, ?, ?, ?)
	`, recipientUserID, nullableInt64(actorUserID), nullableInt64(jobID), string(notificationType), message)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func notificationEvent(userID int64, jobID int64, notificationID int64) domain.DomainEvent {
	return domain.DomainEvent{Type: "notification.created", TargetUserID: int64Ptr(userID), JobID: int64Ptr(jobID), NotificationID: int64Ptr(notificationID)}
}

func jobsChangedEvent(userID int64, jobID int64) domain.DomainEvent {
	return domain.DomainEvent{Type: "jobs.changed", TargetUserID: int64Ptr(userID), JobID: int64Ptr(jobID)}
}

func quotesChangedEvent(quoteID int64) domain.DomainEvent {
	role := domain.RoleManager
	return domain.DomainEvent{Type: "quotes.changed", TargetRole: &role, QuoteID: int64Ptr(quoteID)}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	n := value
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
