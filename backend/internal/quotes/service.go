package quotes

import (
	"context"
	"database/sql"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListQuotes(ctx context.Context, actor domain.Actor, status *domain.QuoteStatus) ([]domain.QuoteDTO, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil {
		return nil, domain.Errorf(domain.ErrorForbidden)
	}
	if status != nil && *status != domain.QuoteUnscheduled && *status != domain.QuoteScheduled {
		return nil, domain.Errorf(domain.ErrorInvalidInput)
	}

	query := `
		SELECT id, customer_name, description, status, created_at, updated_at
		FROM quotes
	`
	args := []any{}
	if status != nil {
		query += ` WHERE status = ?`
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
		if err := rows.Scan(&quote.ID, &quote.CustomerName, &quote.Description, &statusValue, &quote.CreatedAt, &quote.UpdatedAt); err != nil {
			return nil, err
		}
		quote.Status = domain.QuoteStatus(statusValue)
		result = append(result, quote)
	}
	return result, rows.Err()
}
