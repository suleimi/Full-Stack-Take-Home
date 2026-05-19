package store

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// CreateUser inserts a new user and returns it.
func (s *Store) CreateUser(ctx context.Context, firstName, lastName, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (first_name, last_name, email)
		 VALUES ($1, $2, $3)
		 RETURNING id, first_name, last_name, email, created_at, updated_at, deleted_at`,
		firstName, lastName, email,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser returns a single user by ID.
func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, first_name, last_name, email, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers returns a paginated list of active users.
func (s *Store) ListUsers(ctx context.Context, limit, offset int) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, first_name, last_name, email, created_at, updated_at, deleted_at
		 FROM users WHERE deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUser updates an existing user and returns it.
func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, firstName, lastName, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`UPDATE users SET first_name = $2, last_name = $3, email = $4, updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, first_name, last_name, email, created_at, updated_at, deleted_at`,
		id, firstName, lastName, email,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser soft-deletes a user by setting deleted_at.
func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE users SET deleted_at = now(), updated_at = now()
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

// GetUserByEmail returns a single user matching the given email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, first_name, last_name, email, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`, email,
	).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CountUsers returns the total number of active users.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`,
	).Scan(&count)
	return count, err
}
