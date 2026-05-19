package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// ListOpenViolationsByOrg returns unacknowledged violations for an org, newest first.
func (s *Store) ListOpenViolationsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]EmissionViolation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, entity_type, entity_id, policy_id, period,
		        limit_value, unit, period_start, measured_value, overage,
		        detected_at, last_evaluated_at, acknowledged_at
		 FROM emission_violations
		 WHERE org_id = $1 AND acknowledged_at IS NULL
		 ORDER BY detected_at DESC
		 LIMIT $2 OFFSET $3`, orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanViolations(rows)
}

// ListViolationsByOrg returns all violations (open and acknowledged) for an org.
func (s *Store) ListViolationsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]EmissionViolation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, entity_type, entity_id, policy_id, period,
		        limit_value, unit, period_start, measured_value, overage,
		        detected_at, last_evaluated_at, acknowledged_at
		 FROM emission_violations
		 WHERE org_id = $1
		 ORDER BY detected_at DESC
		 LIMIT $2 OFFSET $3`, orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanViolations(rows)
}

// ListViolationsByEntity returns the full violation history for a specific entity.
func (s *Store) ListViolationsByEntity(ctx context.Context, entityType EntityType, entityID uuid.UUID, limit, offset int) ([]EmissionViolation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, entity_type, entity_id, policy_id, period,
		        limit_value, unit, period_start, measured_value, overage,
		        detected_at, last_evaluated_at, acknowledged_at
		 FROM emission_violations
		 WHERE entity_type = $1 AND entity_id = $2
		 ORDER BY detected_at DESC
		 LIMIT $3 OFFSET $4`, entityType, entityID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanViolations(rows)
}

// GetViolation returns a single violation by ID.
func (s *Store) GetViolation(ctx context.Context, id uuid.UUID) (*EmissionViolation, error) {
	v := &EmissionViolation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, entity_type, entity_id, policy_id, period,
		        limit_value, unit, period_start, measured_value, overage,
		        detected_at, last_evaluated_at, acknowledged_at
		 FROM emission_violations WHERE id = $1`, id,
	).Scan(&v.ID, &v.OrgID, &v.EntityType, &v.EntityID, &v.PolicyID, &v.Period,
		&v.LimitValue, &v.Unit, &v.PeriodStart, &v.MeasuredValue, &v.Overage,
		&v.DetectedAt, &v.LastEvaluatedAt, &v.AcknowledgedAt)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// AcknowledgeViolation marks a violation as acknowledged.
func (s *Store) AcknowledgeViolation(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE emission_violations SET acknowledged_at = now()
		 WHERE id = $1 AND acknowledged_at IS NULL`, id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CountOpenViolationsByOrg returns the number of unacknowledged violations for an org.
func (s *Store) CountOpenViolationsByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM emission_violations
		 WHERE org_id = $1 AND acknowledged_at IS NULL`, orgID,
	).Scan(&count)
	return count, err
}

// CountViolationsByOrg returns the total number of violations for an org.
func (s *Store) CountViolationsByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM emission_violations WHERE org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func scanViolations(rows *sql.Rows) ([]EmissionViolation, error) {
	var violations []EmissionViolation
	for rows.Next() {
		var v EmissionViolation
		if err := rows.Scan(
			&v.ID, &v.OrgID, &v.EntityType, &v.EntityID, &v.PolicyID, &v.Period,
			&v.LimitValue, &v.Unit, &v.PeriodStart, &v.MeasuredValue, &v.Overage,
			&v.DetectedAt, &v.LastEvaluatedAt, &v.AcknowledgedAt,
		); err != nil {
			return nil, err
		}
		violations = append(violations, v)
	}
	return violations, rows.Err()
}
