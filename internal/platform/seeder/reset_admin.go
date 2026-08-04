package seeder

import (
	"context"
	"database/sql"
	"fmt"

	"backend_crm_piposmart/internal/platform/factory"
	"backend_crm_piposmart/internal/platform/password"
)

func ResetAdminPasswordToDefault(ctx context.Context, db *sql.DB) error {
	passHash, err := password.HashArgon2id(factory.DummyPassword)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE email = 'admin@piposmart.id'", passHash)
	if err != nil {
		return fmt.Errorf("update admin password: %w", err)
	}
	return nil
}
