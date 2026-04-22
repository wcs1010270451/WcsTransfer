package postgres

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (s *Store) EnsureBootstrapAdmin(ctx context.Context, username string, password string, displayName string) error {
	trimmedUsername := strings.TrimSpace(username)
	trimmedPassword := strings.TrimSpace(password)
	if trimmedUsername == "" || trimmedPassword == "" {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(trimmedPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	const query = `
INSERT INTO admin_users (username, password_hash, display_name, status)
VALUES ($1, $2, $3, 'active')
ON CONFLICT (username)
DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    display_name = EXCLUDED.display_name,
    status = 'active',
    updated_at = NOW()`

	if _, err := s.pool.Exec(ctx, query, trimmedUsername, string(passwordHash), strings.TrimSpace(displayName)); err != nil {
		return fmt.Errorf("ensure bootstrap admin: %w", err)
	}

	return nil
}
