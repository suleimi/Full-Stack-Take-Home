package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CreateSite inserts a new site under the given organization.
func (s *Store) CreateSite(ctx context.Context, orgID uuid.UUID, name string, location *string) (*Site, error) {
	site := &Site{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO sites (org_id, name, location)
		 VALUES ($1, $2, $3)
		 RETURNING id, org_id, name, location, created_at, updated_at, deleted_at`,
		orgID, name, location,
	).Scan(&site.ID, &site.OrgID, &site.Name, &site.Location, &site.CreatedAt, &site.UpdatedAt, &site.DeletedAt)
	if err != nil {
		return nil, err
	}
	return site, nil
}

// GetSite returns a single site by ID within the given organization.
func (s *Store) GetSite(ctx context.Context, orgID, siteID uuid.UUID) (*Site, error) {
	site := &Site{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, name, location, created_at, updated_at, deleted_at
		 FROM sites WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`, siteID, orgID,
	).Scan(&site.ID, &site.OrgID, &site.Name, &site.Location, &site.CreatedAt, &site.UpdatedAt, &site.DeletedAt)
	if err != nil {
		return nil, err
	}
	return site, nil
}

// ListSitesByOrg returns a paginated list of active sites for an organization.
func (s *Store) ListSitesByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]Site, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, name, location, created_at, updated_at, deleted_at
		 FROM sites WHERE org_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var st Site
		if err := rows.Scan(&st.ID, &st.OrgID, &st.Name, &st.Location, &st.CreatedAt, &st.UpdatedAt, &st.DeletedAt); err != nil {
			return nil, err
		}
		sites = append(sites, st)
	}
	return sites, rows.Err()
}

// UpdateSite updates an existing site and returns it.
func (s *Store) UpdateSite(ctx context.Context, orgID, siteID uuid.UUID, name string, location *string) (*Site, error) {
	site := &Site{}
	err := s.db.QueryRowContext(ctx,
		`UPDATE sites SET name = $3, location = $4, updated_at = now()
		 WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
		 RETURNING id, org_id, name, location, created_at, updated_at, deleted_at`,
		siteID, orgID, name, location,
	).Scan(&site.ID, &site.OrgID, &site.Name, &site.Location, &site.CreatedAt, &site.UpdatedAt, &site.DeletedAt)
	if err != nil {
		return nil, err
	}
	return site, nil
}

// DeleteSite soft-deletes a site by setting deleted_at.
func (s *Store) DeleteSite(ctx context.Context, orgID, siteID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE sites SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL`, siteID, orgID,
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

// CountSitesByOrg returns the total number of active sites for an organization.
func (s *Store) CountSitesByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sites WHERE org_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&count)
	return count, err
}

// GetSiteEmissionTrend returns emission rollups for a site over a time range at the given grain.
func (s *Store) GetSiteEmissionTrend(ctx context.Context, siteID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error) {
	table := rollupTableForGrain(grain)
	return s.queryRollupTrend(ctx, table, EntityTypeSite, siteID, from, to)
}
