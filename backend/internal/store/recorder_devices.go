package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// CreateRecorderDevice inserts a new recorder device of the given type.
// For field_device types, orgID should be nil (global). For sensor/satellite, orgID is required.
func (s *Store) CreateRecorderDevice(ctx context.Context, deviceType DeviceType, orgID *uuid.UUID, deviceMetadata *string) (*RecorderDevice, error) {
	rd := &RecorderDevice{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO recorder_devices (org_id, type, device_metadata)
		 VALUES ($1, $2, $3::jsonb)
		 RETURNING id, org_id, type, device_metadata::text, created_at, updated_at, deleted_at`,
		orgID, deviceType, deviceMetadata,
	).Scan(&rd.ID, &rd.OrgID, &rd.Type, &rd.DeviceMetadata, &rd.CreatedAt, &rd.UpdatedAt, &rd.DeletedAt)
	if err != nil {
		return nil, err
	}
	return rd, nil
}

// GetRecorderDevice returns a single recorder device by ID.
func (s *Store) GetRecorderDevice(ctx context.Context, id uuid.UUID) (*RecorderDevice, error) {
	rd := &RecorderDevice{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, org_id, type, device_metadata::text, created_at, updated_at, deleted_at
		 FROM recorder_devices WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&rd.ID, &rd.OrgID, &rd.Type, &rd.DeviceMetadata, &rd.CreatedAt, &rd.UpdatedAt, &rd.DeletedAt)
	if err != nil {
		return nil, err
	}
	return rd, nil
}

// ListRecorderDevicesByOrg returns a paginated list of active devices for an organization.
func (s *Store) ListRecorderDevicesByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]RecorderDevice, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, type, device_metadata::text, created_at, updated_at, deleted_at
		 FROM recorder_devices WHERE org_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []RecorderDevice
	for rows.Next() {
		var d RecorderDevice
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Type, &d.DeviceMetadata, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// ListFieldDevices returns a paginated list of active field devices across all organizations.
func (s *Store) ListFieldDevices(ctx context.Context, limit, offset int) ([]RecorderDevice, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, org_id, type, device_metadata::text, created_at, updated_at, deleted_at
		 FROM recorder_devices WHERE type = 'field_device' AND deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []RecorderDevice
	for rows.Next() {
		var d RecorderDevice
		if err := rows.Scan(&d.ID, &d.OrgID, &d.Type, &d.DeviceMetadata, &d.CreatedAt, &d.UpdatedAt, &d.DeletedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// DeleteRecorderDevice soft-deletes a recorder device by setting deleted_at.
func (s *Store) DeleteRecorderDevice(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE recorder_devices SET deleted_at = now(), updated_at = now()
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

// AssignDeviceToUser creates an assignment record linking a field device to a user.
func (s *Store) AssignDeviceToUser(ctx context.Context, userID, deviceID uuid.UUID, from time.Time) (*UserRecorderDevice, error) {
	urd := &UserRecorderDevice{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO user_recorder_devices (user_id, recorder_device_id, assigned_during)
		 VALUES ($1, $2, tstzrange($3, NULL))
		 RETURNING id, user_id, recorder_device_id, lower(assigned_during), upper(assigned_during), created_at`,
		userID, deviceID, from,
	).Scan(&urd.ID, &urd.UserID, &urd.RecorderDeviceID, &urd.AssignedFrom, &urd.AssignedTo, &urd.CreatedAt)
	if err != nil {
		return nil, err
	}
	return urd, nil
}

// RevokeDeviceFromUser closes an open device assignment by setting the upper bound.
func (s *Store) RevokeDeviceFromUser(ctx context.Context, userID, deviceID uuid.UUID, revokedAt time.Time) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE user_recorder_devices
		 SET assigned_during = tstzrange(lower(assigned_during), $3)
		 WHERE user_id = $1 AND recorder_device_id = $2
		   AND upper_inf(assigned_during)`, userID, deviceID, revokedAt,
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

// ListDeviceAssignmentsByUser returns all device assignments for a given user.
func (s *Store) ListDeviceAssignmentsByUser(ctx context.Context, userID uuid.UUID) ([]UserRecorderDevice, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, recorder_device_id, lower(assigned_during), upper(assigned_during), created_at
		 FROM user_recorder_devices WHERE user_id = $1
		 ORDER BY lower(assigned_during) DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []UserRecorderDevice
	for rows.Next() {
		var a UserRecorderDevice
		if err := rows.Scan(&a.ID, &a.UserID, &a.RecorderDeviceID, &a.AssignedFrom, &a.AssignedTo, &a.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}
