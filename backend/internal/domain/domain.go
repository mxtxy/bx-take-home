package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Role string

const (
	RoleManager    Role = "manager"
	RoleTechnician Role = "technician"
)

type QuoteStatus string

const (
	QuoteUnscheduled QuoteStatus = "unscheduled"
	QuoteScheduled   QuoteStatus = "scheduled"
)

type JobStatus string

const (
	JobScheduled JobStatus = "scheduled"
	JobCompleted JobStatus = "completed"
)

type NotificationType string

const (
	NotificationJobAssigned  NotificationType = "job_assigned"
	NotificationJobUpdated   NotificationType = "job_updated"
	NotificationJobCompleted NotificationType = "job_completed"
)

type ErrorCode string

const (
	ErrorInvalidInput          ErrorCode = "invalid_input"
	ErrorUnauthorized          ErrorCode = "unauthorized"
	ErrorForbidden             ErrorCode = "forbidden"
	ErrorNotFound              ErrorCode = "not_found"
	ErrorInvalidCredentials    ErrorCode = "invalid_credentials"
	ErrorQuoteAlreadyScheduled ErrorCode = "quote_already_scheduled"
	ErrorScheduleConflict      ErrorCode = "schedule_conflict"
	ErrorTechnicianUnavailable ErrorCode = "technician_unavailable"
	ErrorCompletedJobImmutable ErrorCode = "completed_job_immutable"
	ErrorJobAlreadyCompleted   ErrorCode = "job_already_completed"
	ErrorInternal              ErrorCode = "internal_error"
)

type ServiceError struct {
	Code    ErrorCode
	Message string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code ErrorCode, message string) error {
	return &ServiceError{Code: code, Message: message}
}

func ErrorMessage(code ErrorCode) string {
	switch code {
	case ErrorInvalidInput:
		return "Invalid input."
	case ErrorUnauthorized:
		return "Authentication is required."
	case ErrorForbidden:
		return "You are not allowed to perform this action."
	case ErrorNotFound:
		return "Resource not found."
	case ErrorInvalidCredentials:
		return "Invalid email or password."
	case ErrorQuoteAlreadyScheduled:
		return "Quote has already been scheduled."
	case ErrorScheduleConflict:
		return "Technician already has a job in that time window."
	case ErrorTechnicianUnavailable:
		return "Technician is not available in that time window."
	case ErrorCompletedJobImmutable:
		return "Completed jobs cannot be rescheduled."
	case ErrorJobAlreadyCompleted:
		return "Job has already been completed."
	default:
		return "Internal server error."
	}
}

func Errorf(code ErrorCode) error {
	return NewError(code, ErrorMessage(code))
}

func CodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.Code
	}
	return ErrorInternal
}

type Actor struct {
	UserID         int64  `json:"id"`
	OrganizationID int64  `json:"organizationId"`
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	Role           Role   `json:"role"`
	ManagerID      *int64 `json:"managerId"`
	TechnicianID   *int64 `json:"technicianId"`
}

type CreateQuoteInput struct {
	CustomerName string
	Description  string
}

type AssignJobInput struct {
	QuoteID      int64
	TechnicianID int64
	StartsAt     time.Time
}

type RescheduleJobInput struct {
	JobID        int64
	TechnicianID int64
	StartsAt     time.Time
}

type CompleteJobInput struct {
	JobID int64
}

type TimeWindow struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type QuoteDTO struct {
	ID             int64       `json:"id"`
	OrganizationID int64       `json:"organizationId"`
	CustomerName   string      `json:"customerName"`
	Description    string      `json:"description"`
	Status         QuoteStatus `json:"status"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

type TechnicianDTO struct {
	ID           int64                 `json:"id"`
	UserID       int64                 `json:"userId"`
	DisplayName  string                `json:"displayName"`
	Email        string                `json:"email"`
	Availability []AvailabilityRuleDTO `json:"availability"`
}

type AvailabilityRuleDTO struct {
	Weekday  int    `json:"weekday"`
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

type JobDTO struct {
	ID                int64      `json:"id"`
	OrganizationID    int64      `json:"organizationId"`
	QuoteID           int64      `json:"quoteId"`
	QuoteCustomerName string     `json:"quoteCustomerName,omitempty"`
	QuoteDescription  string     `json:"quoteDescription,omitempty"`
	TechnicianID      int64      `json:"technicianId"`
	TechnicianName    string     `json:"technicianName,omitempty"`
	ManagerID         int64      `json:"managerId"`
	ManagerName       string     `json:"managerName,omitempty"`
	StartsAt          time.Time  `json:"startsAt"`
	EndsAt            time.Time  `json:"endsAt"`
	Status            JobStatus  `json:"status"`
	CompletedAt       *time.Time `json:"completedAt"`
	CreatedAt         time.Time  `json:"createdAt,omitempty"`
	UpdatedAt         time.Time  `json:"updatedAt,omitempty"`
}

type NotificationDTO struct {
	ID             int64            `json:"id"`
	OrganizationID int64            `json:"organizationId"`
	Type           NotificationType `json:"type"`
	Message        string           `json:"message"`
	JobID          *int64           `json:"jobId"`
	ReadAt         *time.Time       `json:"readAt"`
	CreatedAt      time.Time        `json:"createdAt"`
}

type DomainEvent struct {
	Type           string
	OrganizationID int64
	TargetUserID   *int64
	TargetRole     *Role
	JobID          *int64
	QuoteID        *int64
	NotificationID *int64
}

func CalculateWindow(startsAt time.Time) TimeWindow {
	start := startsAt.UTC()
	return TimeWindow{
		StartsAt: start,
		EndsAt:   start.Add(2 * time.Hour),
	}
}

func ValidateStartsAt(startsAt time.Time) error {
	if startsAt.Second() != 0 || startsAt.Nanosecond() != 0 {
		return Errorf(ErrorInvalidInput)
	}
	return nil
}

func ValidateCreateQuoteInput(input CreateQuoteInput) error {
	if strings.TrimSpace(input.CustomerName) == "" || strings.TrimSpace(input.Description) == "" {
		return Errorf(ErrorInvalidInput)
	}
	return nil
}

func Overlaps(existing TimeWindow, requested TimeWindow) bool {
	return existing.StartsAt.Before(requested.EndsAt) && existing.EndsAt.After(requested.StartsAt)
}

func ValidateAssignJobInput(input AssignJobInput) error {
	if input.QuoteID <= 0 || input.TechnicianID <= 0 {
		return Errorf(ErrorInvalidInput)
	}
	return ValidateStartsAt(input.StartsAt)
}

func ValidateRescheduleJobInput(input RescheduleJobInput) error {
	if input.JobID <= 0 || input.TechnicianID <= 0 {
		return Errorf(ErrorInvalidInput)
	}
	return ValidateStartsAt(input.StartsAt)
}

func ValidateCompleteJobInput(input CompleteJobInput) error {
	if input.JobID <= 0 {
		return Errorf(ErrorInvalidInput)
	}
	return nil
}
