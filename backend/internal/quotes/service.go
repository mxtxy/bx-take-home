package quotes

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListQuotes(ctx context.Context, actor domain.Actor, status *domain.QuoteStatus) ([]domain.QuoteDTO, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil || actor.OrganizationID <= 0 {
		return nil, domain.Errorf(domain.ErrorForbidden)
	}
	if status != nil && *status != domain.QuoteUnscheduled && *status != domain.QuoteScheduled {
		return nil, domain.Errorf(domain.ErrorInvalidInput)
	}

	query := `
		SELECT id, organization_id, customer_name, description, status, created_at, updated_at
		FROM quotes
		WHERE organization_id = ?
	`
	args := []any{actor.OrganizationID}
	if status != nil {
		query += ` AND status = ?`
		args = append(args, string(*status))
	}
	query += ` ORDER BY created_at ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.QuoteDTO, 0)
	for rows.Next() {
		var quote domain.QuoteDTO
		var statusValue string
		if err := rows.Scan(&quote.ID, &quote.OrganizationID, &quote.CustomerName, &quote.Description, &statusValue, &quote.CreatedAt, &quote.UpdatedAt); err != nil {
			return nil, err
		}
		quote.Status = domain.QuoteStatus(statusValue)
		result = append(result, quote)
	}
	return result, rows.Err()
}

func (s *Service) CreateQuote(ctx context.Context, actor domain.Actor, input domain.CreateQuoteInput) (domain.QuoteDTO, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil || actor.OrganizationID <= 0 {
		return domain.QuoteDTO{}, domain.Errorf(domain.ErrorForbidden)
	}
	if err := domain.ValidateCreateQuoteInput(input); err != nil {
		return domain.QuoteDTO{}, err
	}
	customerName := strings.TrimSpace(input.CustomerName)
	description := strings.TrimSpace(input.Description)
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO quotes (organization_id, customer_name, description, status)
		VALUES (?, ?, ?, 'unscheduled')
	`, actor.OrganizationID, customerName, description)
	if err != nil {
		return domain.QuoteDTO{}, err
	}
	quoteID, err := result.LastInsertId()
	if err != nil {
		return domain.QuoteDTO{}, err
	}
	var quote domain.QuoteDTO
	var statusValue string
	err = s.db.QueryRowContext(ctx, `
		SELECT id, organization_id, customer_name, description, status, created_at, updated_at
		FROM quotes
		WHERE id = ? AND organization_id = ?
	`, quoteID, actor.OrganizationID).Scan(&quote.ID, &quote.OrganizationID, &quote.CustomerName, &quote.Description, &statusValue, &quote.CreatedAt, &quote.UpdatedAt)
	if err != nil {
		return domain.QuoteDTO{}, err
	}
	quote.Status = domain.QuoteStatus(statusValue)
	return quote, nil
}
