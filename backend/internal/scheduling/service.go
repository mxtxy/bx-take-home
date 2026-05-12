package scheduling

import (
	"context"
	"database/sql"
	"time"
	_ "time/tzdata"

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

var scheduleLocation = mustScheduleLocation("Australia/Sydney")

type quoteRow struct {
	ID             int64
	OrganizationID int64
	Status         domain.QuoteStatus
	CustomerName   string
	Description    string
}

type technicianRow struct {
	ID             int64
	OrganizationID int64
	UserID         int64
	DisplayName    string
}

type jobRow struct {
	ID                int64
	OrganizationID    int64
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

type availabilityRuleRow struct {
	Weekday  int
	StartsAt string
	EndsAt   string
}

func NewService(database *sql.DB, options Options) *Service {
	if options.Clock == nil {
		options.Clock = func() time.Time { return time.Now().UTC() }
	}
	return &Service{db: database, clock: options.Clock}
}

func mustScheduleLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return location
}

func (s *Service) AssignJob(ctx context.Context, actor domain.Actor, input domain.AssignJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil || actor.OrganizationID <= 0 {
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

	quote, err := selectQuoteForUpdate(ctx, tx, actor.OrganizationID, input.QuoteID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if quote.Status != domain.QuoteUnscheduled {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorQuoteAlreadyScheduled)
	}

	technician, err := selectTechnicianForUpdate(ctx, tx, actor.OrganizationID, input.TechnicianID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureTechnicianAvailable(ctx, tx, actor.OrganizationID, input.TechnicianID, window.StartsAt, window.EndsAt); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureNoOverlap(ctx, tx, actor.OrganizationID, input.TechnicianID, window.StartsAt, window.EndsAt, nil); err != nil {
		return domain.JobDTO{}, nil, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (organization_id, quote_id, technician_id, manager_id, starts_at, ends_at, status)
		VALUES (?, ?, ?, ?, ?, ?, 'scheduled')
	`, actor.OrganizationID, input.QuoteID, input.TechnicianID, *actor.ManagerID, window.StartsAt, window.EndsAt)
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
	if _, err := tx.ExecContext(ctx, `UPDATE quotes SET status = 'scheduled' WHERE id = ? AND organization_id = ?`, input.QuoteID, actor.OrganizationID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := insertAuditLog(ctx, tx, auditLogInput{
		OrganizationID:  actor.OrganizationID,
		ActorUserID:     actor.UserID,
		Action:          "job_assigned",
		JobID:           jobID,
		QuoteID:         input.QuoteID,
		NewTechnicianID: &input.TechnicianID,
		NewStartsAt:     &window.StartsAt,
		NewEndsAt:       &window.EndsAt,
	}); err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationID, err := insertNotification(ctx, tx, actor.OrganizationID, technician.UserID, &actor.UserID, &jobID, domain.NotificationJobAssigned, "You have been assigned job #"+itoa(jobID)+" for "+quote.CustomerName+".")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:             jobID,
		OrganizationID: actor.OrganizationID,
		QuoteID:        input.QuoteID,
		TechnicianID:   input.TechnicianID,
		ManagerID:      *actor.ManagerID,
		StartsAt:       window.StartsAt,
		EndsAt:         window.EndsAt,
		Status:         domain.JobScheduled,
	}
	events := []domain.DomainEvent{
		notificationEvent(actor.OrganizationID, technician.UserID, jobID, notificationID),
		jobsChangedEvent(actor.OrganizationID, technician.UserID, jobID),
		quotesChangedEvent(actor.OrganizationID, input.QuoteID),
		jobsChangedEvent(actor.OrganizationID, actor.UserID, jobID),
	}
	return job, events, nil
}

func (s *Service) RescheduleJob(ctx context.Context, actor domain.Actor, input domain.RescheduleJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil || actor.OrganizationID <= 0 {
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

	current, err := selectJobForUpdate(ctx, tx, actor.OrganizationID, input.JobID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if current.ManagerID != *actor.ManagerID {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorForbidden)
	}
	if current.Status == domain.JobCompleted {
		return domain.JobDTO{}, nil, domain.Errorf(domain.ErrorCompletedJobImmutable)
	}
	targetTechnician, err := selectTechnicianForUpdate(ctx, tx, actor.OrganizationID, input.TechnicianID)
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureTechnicianAvailable(ctx, tx, actor.OrganizationID, input.TechnicianID, window.StartsAt, window.EndsAt); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := ensureNoOverlap(ctx, tx, actor.OrganizationID, input.TechnicianID, window.StartsAt, window.EndsAt, &input.JobID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET technician_id = ?, starts_at = ?, ends_at = ?
		WHERE id = ? AND organization_id = ?
	`, input.TechnicianID, window.StartsAt, window.EndsAt, input.JobID, actor.OrganizationID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := insertAuditLog(ctx, tx, auditLogInput{
		OrganizationID:       actor.OrganizationID,
		ActorUserID:          actor.UserID,
		Action:               "job_rescheduled",
		JobID:                input.JobID,
		QuoteID:              current.QuoteID,
		PreviousTechnicianID: &current.TechnicianID,
		NewTechnicianID:      &input.TechnicianID,
		PreviousStartsAt:     &current.StartsAt,
		PreviousEndsAt:       &current.EndsAt,
		NewStartsAt:          &window.StartsAt,
		NewEndsAt:            &window.EndsAt,
	}); err != nil {
		return domain.JobDTO{}, nil, err
	}

	var notificationIDs []int64
	targetNotificationID, err := insertNotification(ctx, tx, actor.OrganizationID, targetTechnician.UserID, &actor.UserID, &input.JobID, domain.NotificationJobUpdated, "Job #"+itoa(input.JobID)+" has been updated.")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationIDs = append(notificationIDs, targetNotificationID)
	if current.TechnicianID != input.TechnicianID {
		previousNotificationID, err := insertNotification(ctx, tx, actor.OrganizationID, current.TechnicianUserID, &actor.UserID, &input.JobID, domain.NotificationJobUpdated, "Job #"+itoa(input.JobID)+" has been reassigned.")
		if err != nil {
			return domain.JobDTO{}, nil, err
		}
		notificationIDs = append(notificationIDs, previousNotificationID)
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:             input.JobID,
		OrganizationID: actor.OrganizationID,
		QuoteID:        current.QuoteID,
		TechnicianID:   input.TechnicianID,
		ManagerID:      current.ManagerID,
		StartsAt:       window.StartsAt,
		EndsAt:         window.EndsAt,
		Status:         domain.JobScheduled,
	}
	events := []domain.DomainEvent{
		notificationEvent(actor.OrganizationID, targetTechnician.UserID, input.JobID, notificationIDs[0]),
		jobsChangedEvent(actor.OrganizationID, targetTechnician.UserID, input.JobID),
	}
	if current.TechnicianID != input.TechnicianID {
		events = append(events,
			notificationEvent(actor.OrganizationID, current.TechnicianUserID, input.JobID, notificationIDs[1]),
			jobsChangedEvent(actor.OrganizationID, current.TechnicianUserID, input.JobID),
		)
	}
	events = append(events, jobsChangedEvent(actor.OrganizationID, actor.UserID, input.JobID))
	return job, events, nil
}

func (s *Service) CompleteJob(ctx context.Context, actor domain.Actor, input domain.CompleteJobInput) (domain.JobDTO, []domain.DomainEvent, error) {
	if actor.Role != domain.RoleTechnician || actor.TechnicianID == nil || actor.OrganizationID <= 0 {
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

	current, err := selectJobForUpdate(ctx, tx, actor.OrganizationID, input.JobID)
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
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET status = 'completed', completed_at = ? WHERE id = ? AND organization_id = ?`, completedAt, input.JobID, actor.OrganizationID); err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := insertAuditLog(ctx, tx, auditLogInput{
		OrganizationID:       actor.OrganizationID,
		ActorUserID:          actor.UserID,
		Action:               "job_completed",
		JobID:                input.JobID,
		QuoteID:              current.QuoteID,
		PreviousTechnicianID: &current.TechnicianID,
		NewTechnicianID:      &current.TechnicianID,
		PreviousStartsAt:     &current.StartsAt,
		PreviousEndsAt:       &current.EndsAt,
		NewStartsAt:          &current.StartsAt,
		NewEndsAt:            &current.EndsAt,
		NewCompletedAt:       &completedAt,
	}); err != nil {
		return domain.JobDTO{}, nil, err
	}
	notificationID, err := insertNotification(ctx, tx, actor.OrganizationID, current.ManagerUserID, &actor.UserID, &input.JobID, domain.NotificationJobCompleted, "Job #"+itoa(input.JobID)+" for "+current.QuoteCustomerName+" was completed.")
	if err != nil {
		return domain.JobDTO{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return domain.JobDTO{}, nil, err
	}

	job := domain.JobDTO{
		ID:             input.JobID,
		OrganizationID: actor.OrganizationID,
		QuoteID:        current.QuoteID,
		TechnicianID:   current.TechnicianID,
		ManagerID:      current.ManagerID,
		StartsAt:       current.StartsAt,
		EndsAt:         current.EndsAt,
		Status:         domain.JobCompleted,
		CompletedAt:    &completedAt,
	}
	events := []domain.DomainEvent{
		notificationEvent(actor.OrganizationID, current.ManagerUserID, input.JobID, notificationID),
		jobsChangedEvent(actor.OrganizationID, current.ManagerUserID, input.JobID),
		jobsChangedEvent(actor.OrganizationID, actor.UserID, input.JobID),
	}
	return job, events, nil
}

func selectQuoteForUpdate(ctx context.Context, tx *sql.Tx, organizationID int64, quoteID int64) (quoteRow, error) {
	var quote quoteRow
	var status string
	err := tx.QueryRowContext(ctx, `
		SELECT id, organization_id, customer_name, description, status
		FROM quotes
		WHERE id = ? AND organization_id = ?
		FOR UPDATE
	`, quoteID, organizationID).Scan(&quote.ID, &quote.OrganizationID, &quote.CustomerName, &quote.Description, &status)
	if err == sql.ErrNoRows {
		return quoteRow{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return quoteRow{}, err
	}
	quote.Status = domain.QuoteStatus(status)
	return quote, nil
}

func selectTechnicianForUpdate(ctx context.Context, tx *sql.Tx, organizationID int64, technicianID int64) (technicianRow, error) {
	var technician technicianRow
	err := tx.QueryRowContext(ctx, `
		SELECT t.id, u.organization_id, t.user_id, u.display_name
		FROM technicians t
		JOIN users u ON u.id = t.user_id
		WHERE t.id = ? AND u.organization_id = ?
		FOR UPDATE
	`, technicianID, organizationID).Scan(&technician.ID, &technician.OrganizationID, &technician.UserID, &technician.DisplayName)
	if err == sql.ErrNoRows {
		return technicianRow{}, domain.Errorf(domain.ErrorNotFound)
	}
	if err != nil {
		return technicianRow{}, err
	}
	return technician, nil
}

func selectJobForUpdate(ctx context.Context, tx *sql.Tx, organizationID int64, jobID int64) (jobRow, error) {
	var job jobRow
	var status string
	var completedAt sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT
			j.id, j.organization_id, j.quote_id, q.customer_name, q.description,
			j.technician_id, tech.user_id, tech_user.display_name,
			j.manager_id, manager.user_id, manager_user.display_name,
			j.starts_at, j.ends_at, j.status, j.completed_at
		FROM jobs j
		JOIN quotes q ON q.id = j.quote_id
		JOIN technicians tech ON tech.id = j.technician_id
		JOIN users tech_user ON tech_user.id = tech.user_id
		JOIN managers manager ON manager.id = j.manager_id
		JOIN users manager_user ON manager_user.id = manager.user_id
		WHERE j.id = ? AND j.organization_id = ?
		FOR UPDATE
	`, jobID, organizationID).Scan(
		&job.ID,
		&job.OrganizationID,
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

func ensureTechnicianAvailable(ctx context.Context, tx *sql.Tx, organizationID int64, technicianID int64, startsAt time.Time, endsAt time.Time) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT weekday, TIME_FORMAT(starts_at, '%H:%i'), TIME_FORMAT(ends_at, '%H:%i')
		FROM technician_availability_rules
		WHERE organization_id = ?
		  AND technician_id = ?
	`, organizationID, technicianID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rule availabilityRuleRow
		if err := rows.Scan(&rule.Weekday, &rule.StartsAt, &rule.EndsAt); err != nil {
			return err
		}
		if windowInsideAvailability(rule, startsAt, endsAt) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return domain.Errorf(domain.ErrorTechnicianUnavailable)
}

func windowInsideAvailability(rule availabilityRuleRow, startsAt time.Time, endsAt time.Time) bool {
	localStart := startsAt.In(scheduleLocation)
	localEnd := endsAt.In(scheduleLocation)
	if !sameLocalDate(localStart, localEnd) {
		return false
	}
	startTime := localClock(localStart)
	endTime := localClock(localEnd)
	return rule.Weekday == mondayBasedWeekday(localStart) && rule.StartsAt <= startTime && rule.EndsAt >= endTime
}

func sameLocalDate(left time.Time, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func mondayBasedWeekday(value time.Time) int {
	return (int(value.Weekday()) + 6) % 7
}

func localClock(value time.Time) string {
	return value.Format("15:04")
}

func ensureNoOverlap(ctx context.Context, tx *sql.Tx, organizationID int64, technicianID int64, startsAt time.Time, endsAt time.Time, excludeJobID *int64) error {
	query := `
		SELECT COUNT(*)
		FROM jobs
		WHERE organization_id = ?
		  AND technician_id = ?
		  AND starts_at < ?
		  AND ends_at > ?
	`
	args := []any{organizationID, technicianID, endsAt, startsAt}
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

type auditLogInput struct {
	OrganizationID       int64
	ActorUserID          int64
	Action               string
	JobID                int64
	QuoteID              int64
	PreviousTechnicianID *int64
	NewTechnicianID      *int64
	PreviousStartsAt     *time.Time
	PreviousEndsAt       *time.Time
	NewStartsAt          *time.Time
	NewEndsAt            *time.Time
	NewCompletedAt       *time.Time
}

func insertAuditLog(ctx context.Context, tx *sql.Tx, input auditLogInput) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO schedule_audit_logs (
			organization_id, actor_user_id, action, job_id, quote_id,
			previous_technician_id, new_technician_id,
			previous_starts_at, previous_ends_at,
			new_starts_at, new_ends_at, new_completed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		input.OrganizationID,
		input.ActorUserID,
		input.Action,
		input.JobID,
		input.QuoteID,
		nullableInt64(input.PreviousTechnicianID),
		nullableInt64(input.NewTechnicianID),
		nullableTime(input.PreviousStartsAt),
		nullableTime(input.PreviousEndsAt),
		nullableTime(input.NewStartsAt),
		nullableTime(input.NewEndsAt),
		nullableTime(input.NewCompletedAt),
	)
	return err
}

func insertNotification(ctx context.Context, tx *sql.Tx, organizationID int64, recipientUserID int64, actorUserID *int64, jobID *int64, notificationType domain.NotificationType, message string) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO notifications (organization_id, recipient_user_id, actor_user_id, job_id, type, message)
		VALUES (?, ?, ?, ?, ?, ?)
	`, organizationID, recipientUserID, nullableInt64(actorUserID), nullableInt64(jobID), string(notificationType), message)
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

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func notificationEvent(organizationID int64, userID int64, jobID int64, notificationID int64) domain.DomainEvent {
	return domain.DomainEvent{Type: "notification.created", OrganizationID: organizationID, TargetUserID: int64Ptr(userID), JobID: int64Ptr(jobID), NotificationID: int64Ptr(notificationID)}
}

func jobsChangedEvent(organizationID int64, userID int64, jobID int64) domain.DomainEvent {
	return domain.DomainEvent{Type: "jobs.changed", OrganizationID: organizationID, TargetUserID: int64Ptr(userID), JobID: int64Ptr(jobID)}
}

func quotesChangedEvent(organizationID int64, quoteID int64) domain.DomainEvent {
	role := domain.RoleManager
	return domain.DomainEvent{Type: "quotes.changed", OrganizationID: organizationID, TargetRole: &role, QuoteID: int64Ptr(quoteID)}
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
