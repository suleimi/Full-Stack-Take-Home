package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// rollupTableForGrain maps a grain string to the corresponding rollup table.
func rollupTableForGrain(grain Grain) string {
	switch grain {
	case GrainHour:
		return "emission_rollup_hourly"
	case GrainDay:
		return "emission_rollup_daily"
	case GrainMonth:
		return "emission_rollup_monthly"
	default:
		return "emission_rollup_daily"
	}
}

// queryRollupTrend is the shared implementation for fetching emission trends.
func (s *Store) queryRollupTrend(ctx context.Context, table string, entityType EntityType, entityID uuid.UUID, from, to time.Time) ([]EmissionRollup, error) {
	query := fmt.Sprintf(
		`SELECT entity_type, entity_id, bucket_start, total_emission,
		        reading_count, min_reading, max_reading, unit, computed_at
		 FROM %s
		 WHERE entity_type = $1 AND entity_id = $2
		   AND bucket_start >= $3 AND bucket_start < $4
		 ORDER BY bucket_start`, table)

	rows, err := s.db.QueryContext(ctx, query, entityType, entityID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRollups(rows)
}

// GetEmissionTotalToDate returns the lifetime emission total for a given entity.
func (s *Store) GetEmissionTotalToDate(ctx context.Context, entityType EntityType, entityID uuid.UUID) (*EmissionTotalToDate, error) {
	t := &EmissionTotalToDate{}
	err := s.db.QueryRowContext(ctx,
		`SELECT entity_type, entity_id, total_to_date, reading_count, as_of
		 FROM emission_total_to_date
		 WHERE entity_type = $1 AND entity_id = $2`, entityType, entityID,
	).Scan(&t.EntityType, &t.EntityID, &t.TotalToDate, &t.ReadingCount, &t.AsOf)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListRollupsByOrg returns rollup rows for all entities belonging to an org at a given grain.
// Useful for dashboards showing all sites or assets side by side.
func (s *Store) ListRollupsByOrg(ctx context.Context, orgID uuid.UUID, entityType EntityType, grain Grain, from, to time.Time) ([]EmissionRollup, error) {
	table := rollupTableForGrain(grain)

	var query string
	switch entityType {
	case EntityTypeSite:
		query = fmt.Sprintf(
			`SELECT r.entity_type, r.entity_id, r.bucket_start, r.total_emission,
			        r.reading_count, r.min_reading, r.max_reading, r.unit, r.computed_at
			 FROM %s r
			 JOIN sites s ON s.id = r.entity_id
			 WHERE r.entity_type = 'site' AND s.org_id = $1
			   AND r.bucket_start >= $2 AND r.bucket_start < $3
			 ORDER BY r.bucket_start`, table)
	case EntityTypeAsset:
		query = fmt.Sprintf(
			`SELECT r.entity_type, r.entity_id, r.bucket_start, r.total_emission,
			        r.reading_count, r.min_reading, r.max_reading, r.unit, r.computed_at
			 FROM %s r
			 JOIN assets a ON a.id = r.entity_id
			 WHERE r.entity_type = 'asset' AND a.org_id = $1
			   AND r.bucket_start >= $2 AND r.bucket_start < $3
			 ORDER BY r.bucket_start`, table)
	default:
		query = fmt.Sprintf(
			`SELECT entity_type, entity_id, bucket_start, total_emission,
			        reading_count, min_reading, max_reading, unit, computed_at
			 FROM %s
			 WHERE entity_type = 'org' AND entity_id = $1
			   AND bucket_start >= $2 AND bucket_start < $3
			 ORDER BY bucket_start`, table)
	}

	rows, err := s.db.QueryContext(ctx, query, orgID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRollups(rows)
}

// RefreshEmissions calls the refresh_emissions stored procedure.
func (s *Store) RefreshEmissions(ctx context.Context, window string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("CALL refresh_emissions(INTERVAL '%s')", window))
	return err
}

func scanRollups(rows *sql.Rows) ([]EmissionRollup, error) {
	var rollups []EmissionRollup
	for rows.Next() {
		var r EmissionRollup
		if err := rows.Scan(
			&r.EntityType, &r.EntityID, &r.BucketStart, &r.TotalEmission,
			&r.ReadingCount, &r.MinReading, &r.MaxReading, &r.Unit, &r.ComputedAt,
		); err != nil {
			return nil, err
		}
		rollups = append(rollups, r)
	}
	return rollups, rows.Err()
}
