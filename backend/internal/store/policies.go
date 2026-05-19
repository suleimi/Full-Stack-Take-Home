package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CreateEmissionPolicy inserts a new emission policy for the given entity.
// effective_from is truncated to the period boundary so the policy governs
// the current ongoing bucket (e.g. creating a daily policy at 10:23 takes
// effect from 00:00 that day, not the next day).
func (s *Store) CreateEmissionPolicy(ctx context.Context, orgID, entityID uuid.UUID, entityType EntityType, period Period, emissionLimit float64, unit EmissionUnit, effectiveFrom time.Time) (*EmissionPolicy, error) {
	effectiveFrom = truncateToPeriod(effectiveFrom, period)

	p := &EmissionPolicy{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO emission_policies (org_id, entity_type, entity_id, emission_limit, unit, period, effective_from)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, org_id, entity_type, entity_id, emission_limit, unit, period, effective_from, effective_to, created_at`,
		orgID, entityType, entityID, emissionLimit, unit, period, effectiveFrom,
	).Scan(&p.ID, &p.OrgID, &p.EntityType, &p.EntityID, &p.EmissionLimit, &p.Unit, &p.Period, &p.EffectiveFrom, &p.EffectiveTo, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetEmissionPolicy returns a single emission policy by ID.
func (s *Store) GetEmissionPolicy(ctx context.Context, id uuid.UUID) (*EmissionPolicy, error) {
	p := &EmissionPolicy{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, entity_type, entity_id, emission_limit, unit, period, effective_from, effective_to, created_at
		 FROM emission_policies WHERE id = $1`, id,
	).Scan(&p.ID, &p.OrgID, &p.EntityType, &p.EntityID, &p.EmissionLimit, &p.Unit, &p.Period, &p.EffectiveFrom, &p.EffectiveTo, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListPoliciesByOrg returns all policies for an org, optionally filtered to only active ones.
func (s *Store) ListPoliciesByOrg(ctx context.Context, orgID uuid.UUID, activeOnly bool, limit, offset int) ([]EmissionPolicy, error) {
	query := `SELECT id, org_id, entity_type, entity_id, emission_limit, unit, period, effective_from, effective_to, created_at
		 FROM emission_policies WHERE org_id = $1`
	if activeOnly {
		query += ` AND (effective_to IS NULL OR effective_to > now())`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPolicies(rows)
}

// ListPoliciesByEntity returns policies for a specific entity (org/site/asset).
func (s *Store) ListPoliciesByEntity(ctx context.Context, entityType EntityType, entityID uuid.UUID, activeOnly bool) ([]EmissionPolicy, error) {
	query := `SELECT id, org_id, entity_type, entity_id, emission_limit, unit, period, effective_from, effective_to, created_at
		 FROM emission_policies WHERE entity_type = $1 AND entity_id = $2`
	if activeOnly {
		query += ` AND (effective_to IS NULL OR effective_to > now())`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanPolicies(rows)
}

// RetireEmissionPolicy sets effective_to on a policy so it stops governing future periods.
func (s *Store) RetireEmissionPolicy(ctx context.Context, id uuid.UUID, effectiveTo time.Time) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE emission_policies SET effective_to = $2
		 WHERE id = $1 AND effective_to IS NULL`, id, effectiveTo,
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

// CountPoliciesByOrg returns the total number of policies for an org.
func (s *Store) CountPoliciesByOrg(ctx context.Context, orgID uuid.UUID, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM emission_policies WHERE org_id = $1`
	if activeOnly {
		query += ` AND (effective_to IS NULL OR effective_to > now())`
	}
	var count int
	err := s.db.QueryRowContext(ctx, query, orgID).Scan(&count)
	return count, err
}

// truncateToPeriod floors a timestamp to the start of the given period,
// mirroring the date_trunc that PostgreSQL uses for rollup bucket_start.
func truncateToPeriod(t time.Time, p Period) time.Time {
	switch p {
	case PeriodHour:
		return t.Truncate(time.Hour)
	case PeriodDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case PeriodMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

func scanPolicies(rows *sql.Rows) ([]EmissionPolicy, error) {
	var policies []EmissionPolicy
	for rows.Next() {
		var p EmissionPolicy
		if err := rows.Scan(&p.ID, &p.OrgID, &p.EntityType, &p.EntityID, &p.EmissionLimit, &p.Unit, &p.Period, &p.EffectiveFrom, &p.EffectiveTo, &p.CreatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}
