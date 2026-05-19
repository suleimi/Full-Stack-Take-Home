package store

import (
	"context"

	"github.com/google/uuid"
)

// CreateAssetType inserts a new asset type and returns it.
func (s *Store) CreateAssetType(ctx context.Context, name string, description *string) (*AssetType, error) {
	at := &AssetType{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO asset_types (name, description)
		 VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		name, description,
	).Scan(&at.ID, &at.Name, &at.Description, &at.CreatedAt, &at.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return at, nil
}

// GetAssetType returns a single asset type by ID.
func (s *Store) GetAssetType(ctx context.Context, id uuid.UUID) (*AssetType, error) {
	at := &AssetType{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM asset_types WHERE id = $1`, id,
	).Scan(&at.ID, &at.Name, &at.Description, &at.CreatedAt, &at.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return at, nil
}

// ListAssetTypes returns all asset types ordered by name.
func (s *Store) ListAssetTypes(ctx context.Context) ([]AssetType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM asset_types ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []AssetType
	for rows.Next() {
		var at AssetType
		if err := rows.Scan(&at.ID, &at.Name, &at.Description, &at.CreatedAt, &at.UpdatedAt); err != nil {
			return nil, err
		}
		types = append(types, at)
	}
	return types, rows.Err()
}
