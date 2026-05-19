package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CreateOrganization inserts a new organization and returns it.
func (s *Store) CreateOrganization(ctx context.Context, name, email string, description, officeLocation *string) (*Organization, error) {
	org := &Organization{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO organizations (name, email, description, office_location)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, description, office_location, email, created_at, updated_at, deleted_at`,
		name, email, description, officeLocation,
	).Scan(&org.ID, &org.Name, &org.Description, &org.OfficeLocation, &org.Email, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// GetOrganization returns a single organization by ID.
func (s *Store) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	org := &Organization{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, office_location, email, created_at, updated_at, deleted_at
		 FROM organizations WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&org.ID, &org.Name, &org.Description, &org.OfficeLocation, &org.Email, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// ListOrganizations returns a paginated list of active organizations.
func (s *Store) ListOrganizations(ctx context.Context, limit, offset int) ([]Organization, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, office_location, email, created_at, updated_at, deleted_at
		 FROM organizations WHERE deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Description, &o.OfficeLocation, &o.Email, &o.CreatedAt, &o.UpdatedAt, &o.DeletedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// UpdateOrganization updates an existing organization and returns it.
func (s *Store) UpdateOrganization(ctx context.Context, id uuid.UUID, name, email string, description, officeLocation *string) (*Organization, error) {
	org := &Organization{}
	err := s.db.QueryRowContext(ctx,
		`UPDATE organizations
		 SET name = $2, email = $3, description = $4, office_location = $5, updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, name, description, office_location, email, created_at, updated_at, deleted_at`,
		id, name, email, description, officeLocation,
	).Scan(&org.ID, &org.Name, &org.Description, &org.OfficeLocation, &org.Email, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization soft-deletes an organization by setting deleted_at.
func (s *Store) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE organizations SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id,
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

// GetOrganizationByEmail returns a single organization matching the given email.
func (s *Store) GetOrganizationByEmail(ctx context.Context, email string) (*Organization, error) {
	org := &Organization{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, office_location, email, created_at, updated_at, deleted_at
		 FROM organizations WHERE email = $1 AND deleted_at IS NULL`, email,
	).Scan(&org.ID, &org.Name, &org.Description, &org.OfficeLocation, &org.Email, &org.CreatedAt, &org.UpdatedAt, &org.DeletedAt)
	if err != nil {
		return nil, err
	}
	return org, nil
}

// GetOrganizationSummary returns high-level dashboard stats for an org:
// total sites, total assets, and the latest rollup timestamp.
func (s *Store) GetOrganizationSummary(ctx context.Context, orgID uuid.UUID) (*OrgSummary, error) {
	summary := &OrgSummary{OrgID: orgID}
	err := s.db.QueryRowContext(ctx,
		`SELECT
			(SELECT COUNT(*) FROM sites WHERE org_id = $1 AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM assets WHERE org_id = $1 AND deleted_at IS NULL),
			(SELECT MAX(computed_at) FROM emission_rollup_monthly WHERE entity_type = 'org' AND entity_id = $1)`,
		orgID,
	).Scan(&summary.SiteCount, &summary.AssetCount, &summary.LastUpdate)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

// CountOrganizations returns the total number of active organizations (for pagination).
func (s *Store) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL`,
	).Scan(&count)
	return count, err
}

// GetOrganizationEmissionTrend returns emission rollups for an org over a time range at the given grain.
func (s *Store) GetOrganizationEmissionTrend(ctx context.Context, orgID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error) {
	table := rollupTableForGrain(grain)
	return s.queryRollupTrend(ctx, table, EntityTypeOrg, orgID, from, to)
}
