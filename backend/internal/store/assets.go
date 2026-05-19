package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CreateAsset inserts a new asset under the given organization and site.
func (s *Store) CreateAsset(ctx context.Context, orgID, siteID uuid.UUID, name string, assetTypeID *uuid.UUID) (*Asset, error) {
	asset := &Asset{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO assets (org_id, site_id, asset_type_id, name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, site_id, asset_type_id, name, created_at, updated_at, deleted_at`,
		orgID, siteID, assetTypeID, name,
	).Scan(&asset.ID, &asset.OrgID, &asset.SiteID, &asset.AssetTypeID, &asset.Name, &asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// GetAsset returns a single asset by ID within the given organization and site.
func (s *Store) GetAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID) (*Asset, error) {
	asset := &Asset{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, site_id, asset_type_id, name, created_at, updated_at, deleted_at
		 FROM assets WHERE id = $1 AND org_id = $2 AND site_id = $3 AND deleted_at IS NULL`,
		assetID, orgID, siteID,
	).Scan(&asset.ID, &asset.OrgID, &asset.SiteID, &asset.AssetTypeID, &asset.Name, &asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// ListAssetsBySite returns a paginated list of active assets for a given site.
func (s *Store) ListAssetsBySite(ctx context.Context, orgID, siteID uuid.UUID, limit, offset int) ([]Asset, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, site_id, asset_type_id, name, created_at, updated_at, deleted_at
		 FROM assets WHERE org_id = $1 AND site_id = $2 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`, orgID, siteID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.OrgID, &a.SiteID, &a.AssetTypeID, &a.Name, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// ListAssetsByOrg returns a paginated list of active assets for an organization.
func (s *Store) ListAssetsByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]Asset, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, site_id, asset_type_id, name, created_at, updated_at, deleted_at
		 FROM assets WHERE org_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.OrgID, &a.SiteID, &a.AssetTypeID, &a.Name, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// UpdateAsset updates an existing asset and returns it.
func (s *Store) UpdateAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID, name string, assetTypeID *uuid.UUID) (*Asset, error) {
	asset := &Asset{}
	err := s.db.QueryRowContext(ctx,
		`UPDATE assets SET name = $4, asset_type_id = $5, updated_at = now()
		 WHERE id = $1 AND org_id = $2 AND site_id = $3 AND deleted_at IS NULL
		 RETURNING id, org_id, site_id, asset_type_id, name, created_at, updated_at, deleted_at`,
		assetID, orgID, siteID, name, assetTypeID,
	).Scan(&asset.ID, &asset.OrgID, &asset.SiteID, &asset.AssetTypeID, &asset.Name, &asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt)
	if err != nil {
		return nil, err
	}
	return asset, nil
}

// DeleteAsset soft-deletes an asset by setting deleted_at.
func (s *Store) DeleteAsset(ctx context.Context, orgID, siteID, assetID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE assets SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND org_id = $2 AND site_id = $3 AND deleted_at IS NULL`, assetID, orgID, siteID,
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

// CountAssetsBySite returns the total number of active assets for a given site.
func (s *Store) CountAssetsBySite(ctx context.Context, orgID, siteID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM assets WHERE org_id = $1 AND site_id = $2 AND deleted_at IS NULL`, orgID, siteID,
	).Scan(&count)
	return count, err
}

// CountAssetsByOrg returns the total number of active assets for an organization.
func (s *Store) CountAssetsByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM assets WHERE org_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&count)
	return count, err
}

// GetAssetEmissionTrend returns emission rollups for an asset over a time range at the given grain.
func (s *Store) GetAssetEmissionTrend(ctx context.Context, assetID uuid.UUID, grain Grain, from, to time.Time) ([]EmissionRollup, error) {
	table := rollupTableForGrain(grain)
	return s.queryRollupTrend(ctx, table, EntityTypeAsset, assetID, from, to)
}
