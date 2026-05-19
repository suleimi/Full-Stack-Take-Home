package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ErrDuplicateBatch is returned when a client-supplied batch_id already exists.
var ErrDuplicateBatch = errors.New("duplicate batch_id: readings already ingested")

// claimBatch attempts to insert a row into ingested_batches. The PRIMARY KEY
// on batch_id means a concurrent transaction doing the same INSERT will block
// until the first one commits or rolls back, eliminating the TOCTOU race that
// existed with the old SELECT EXISTS approach.
//
// Returns ErrDuplicateBatch if the batch_id was already claimed.
func claimBatch(ctx context.Context, tx *sql.Tx, batchID, orgID, siteID uuid.UUID) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO ingested_batches (batch_id, org_id, site_id) VALUES ($1, $2, $3)`,
		batchID, orgID, siteID,
	)
	if err != nil {
		// 23505 = unique_violation — another transaction already committed this batch_id.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrDuplicateBatch
		}
		return err
	}
	return nil
}

// InsertReadings batch-inserts emission readings within a transaction.
// The client supplies the batch_id for idempotency. If a batch with that ID
// already exists, ErrDuplicateBatch is returned and no data is written.
func (s *Store) InsertReadings(ctx context.Context, orgID, siteID, batchID uuid.UUID, inputs []ReadingInput) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := claimBatch(ctx, tx, batchID, orgID, siteID); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO asset_emission_readings
		    (org_id, site_id, asset_id, recorder_device_id, batch_id, reading, unit, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range inputs {
		unit := r.Unit
		if unit == "" {
			unit = "kg_co2e"
		}
		_, err := stmt.ExecContext(ctx, orgID, siteID, r.AssetID, r.RecorderDeviceID, batchID, r.Reading, unit, r.RecordedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertReadingsTx batch-inserts readings using an existing transaction (for combining with job enqueue).
// The client supplies the batch_id for idempotency. If a batch with that ID
// already exists, ErrDuplicateBatch is returned and no data is written.
func (s *Store) InsertReadingsTx(ctx context.Context, tx *sql.Tx, orgID, siteID, batchID uuid.UUID, inputs []ReadingInput) error {
	if err := claimBatch(ctx, tx, batchID, orgID, siteID); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO asset_emission_readings
		    (org_id, site_id, asset_id, recorder_device_id, batch_id, reading, unit, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range inputs {
		unit := r.Unit
		if unit == "" {
			unit = "kg_co2e"
		}
		_, err := stmt.ExecContext(ctx, orgID, siteID, r.AssetID, r.RecorderDeviceID, batchID, r.Reading, unit, r.RecordedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListReadingsByAsset returns recent readings for a specific asset.
func (s *Store) ListReadingsByAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, site_id, asset_id, recorder_device_id, batch_id,
		        reading, unit, recorded_at, created_at
		 FROM asset_emission_readings
		 WHERE org_id = $1 AND site_id = $2 AND asset_id = $3
		   AND recorded_at >= $4 AND recorded_at < $5
		 ORDER BY recorded_at DESC
		 LIMIT $6 OFFSET $7`,
		orgID, siteID, assetID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReadings(rows)
}

// ListReadingsBySite returns recent readings for all assets at a site.
func (s *Store) ListReadingsBySite(ctx context.Context, orgID, siteID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, site_id, asset_id, recorder_device_id, batch_id,
		        reading, unit, recorded_at, created_at
		 FROM asset_emission_readings
		 WHERE org_id = $1 AND site_id = $2
		   AND recorded_at >= $3 AND recorded_at < $4
		 ORDER BY recorded_at DESC
		 LIMIT $5 OFFSET $6`,
		orgID, siteID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReadings(rows)
}

// ListReadingsByDevice returns readings produced by a specific recorder device.
func (s *Store) ListReadingsByDevice(ctx context.Context, deviceID uuid.UUID, from, to time.Time, limit, offset int) ([]AssetEmissionReading, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, site_id, asset_id, recorder_device_id, batch_id,
		        reading, unit, recorded_at, created_at
		 FROM asset_emission_readings
		 WHERE recorder_device_id = $1
		   AND recorded_at >= $2 AND recorded_at < $3
		 ORDER BY recorded_at DESC
		 LIMIT $4 OFFSET $5`,
		deviceID, from, to, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReadings(rows)
}

// CountReadingsByAsset returns the total reading count for an asset in a time range.
func (s *Store) CountReadingsByAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, from, to time.Time) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM asset_emission_readings
		 WHERE org_id = $1 AND site_id = $2 AND asset_id = $3
		   AND recorded_at >= $4 AND recorded_at < $5`,
		orgID, siteID, assetID, from, to,
	).Scan(&count)
	return count, err
}

func scanReadings(rows *sql.Rows) ([]AssetEmissionReading, error) {
	var readings []AssetEmissionReading
	for rows.Next() {
		var r AssetEmissionReading
		if err := rows.Scan(
			&r.ID, &r.OrgID, &r.SiteID, &r.AssetID, &r.RecorderDeviceID, &r.BatchID,
			&r.Reading, &r.Unit, &r.RecordedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		readings = append(readings, r)
	}
	return readings, rows.Err()
}
