package jobs

import (
	"context"
	"database/sql"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type QueryService struct {
	db *sql.DB
}

func NewQueryService(db *sql.DB) *QueryService {
	return &QueryService{db: db}
}

func (s *QueryService) ListJobs(ctx context.Context, actor domain.Actor) ([]domain.JobDTO, error) {
	var where string
	var arg int64
	switch actor.Role {
	case domain.RoleManager:
		if actor.ManagerID == nil {
			return nil, domain.Errorf(domain.ErrorUnauthorized)
		}
		where = `j.manager_id = ?`
		arg = *actor.ManagerID
	case domain.RoleTechnician:
		if actor.TechnicianID == nil {
			return nil, domain.Errorf(domain.ErrorUnauthorized)
		}
		where = `j.technician_id = ?`
		arg = *actor.TechnicianID
	default:
		return nil, domain.Errorf(domain.ErrorUnauthorized)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			j.id, j.quote_id, q.customer_name, q.description,
			j.technician_id, tech_user.display_name,
			j.manager_id, manager_user.display_name,
			j.starts_at, j.ends_at, j.status, j.completed_at,
			j.created_at, j.updated_at
		FROM jobs j
		JOIN quotes q ON q.id = j.quote_id
		JOIN technicians tech ON tech.id = j.technician_id
		JOIN users tech_user ON tech_user.id = tech.user_id
		JOIN managers manager ON manager.id = j.manager_id
		JOIN users manager_user ON manager_user.id = manager.user_id
		WHERE `+where+`
		ORDER BY j.starts_at ASC, j.id ASC
	`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.JobDTO, 0)
	for rows.Next() {
		var job domain.JobDTO
		var status string
		var completedAt sql.NullTime
		if err := rows.Scan(
			&job.ID,
			&job.QuoteID,
			&job.QuoteCustomerName,
			&job.QuoteDescription,
			&job.TechnicianID,
			&job.TechnicianName,
			&job.ManagerID,
			&job.ManagerName,
			&job.StartsAt,
			&job.EndsAt,
			&status,
			&completedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		job.Status = domain.JobStatus(status)
		if completedAt.Valid {
			value := completedAt.Time
			job.CompletedAt = &value
		}
		result = append(result, job)
	}
	return result, rows.Err()
}
