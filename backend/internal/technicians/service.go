package technicians

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

func (s *Service) ListTechnicians(ctx context.Context, actor domain.Actor) ([]domain.TechnicianDTO, error) {
	if actor.Role != domain.RoleManager || actor.ManagerID == nil {
		return nil, domain.Errorf(domain.ErrorForbidden)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, u.id, u.display_name, u.email
		FROM technicians t
		JOIN users u ON u.id = t.user_id
		ORDER BY u.display_name ASC, t.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.TechnicianDTO, 0)
	for rows.Next() {
		var technician domain.TechnicianDTO
		if err := rows.Scan(&technician.ID, &technician.UserID, &technician.DisplayName, &technician.Email); err != nil {
			return nil, err
		}
		result = append(result, technician)
	}
	return result, rows.Err()
}
