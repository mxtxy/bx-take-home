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
	if actor.Role != domain.RoleManager || actor.ManagerID == nil || actor.OrganizationID <= 0 {
		return nil, domain.Errorf(domain.ErrorForbidden)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, u.id, u.display_name, u.email
		FROM technicians t
		JOIN users u ON u.id = t.user_id
		WHERE u.organization_id = ?
		ORDER BY u.display_name ASC, t.id ASC
	`, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.TechnicianDTO, 0)
	indexByID := map[int64]int{}
	for rows.Next() {
		var technician domain.TechnicianDTO
		if err := rows.Scan(&technician.ID, &technician.UserID, &technician.DisplayName, &technician.Email); err != nil {
			return nil, err
		}
		technician.Availability = []domain.AvailabilityRuleDTO{}
		indexByID[technician.ID] = len(result)
		result = append(result, technician)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	availabilityRows, err := s.db.QueryContext(ctx, `
		SELECT ar.technician_id, ar.weekday, TIME_FORMAT(ar.starts_at, '%H:%i'), TIME_FORMAT(ar.ends_at, '%H:%i')
		FROM technician_availability_rules ar
		JOIN technicians t ON t.id = ar.technician_id
		JOIN users u ON u.id = t.user_id
		WHERE ar.organization_id = ?
		  AND u.organization_id = ?
		ORDER BY ar.technician_id ASC, ar.weekday ASC, ar.starts_at ASC
	`, actor.OrganizationID, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defer availabilityRows.Close()

	for availabilityRows.Next() {
		var technicianID int64
		var rule domain.AvailabilityRuleDTO
		if err := availabilityRows.Scan(&technicianID, &rule.Weekday, &rule.StartsAt, &rule.EndsAt); err != nil {
			return nil, err
		}
		if index, ok := indexByID[technicianID]; ok {
			result[index].Availability = append(result[index].Availability, rule)
		}
	}
	return result, availabilityRows.Err()
}
