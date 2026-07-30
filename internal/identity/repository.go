package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

type AuditEntry struct {
	ActorUserID sql.NullInt64
	Action      string
	EntityType  string
	EntityID    sql.NullInt64
	Before      any
	After       any
	IPAddress   string
	UserAgent   string
	RequestID   string
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (User, error) {
	return r.findUser(ctx, "u.email = ?", strings.ToLower(strings.TrimSpace(email)))
}

func (r *Repository) FindUserByID(ctx context.Context, id int64) (User, error) {
	return r.findUser(ctx, "u.id = ?", id)
}

func (r *Repository) findUser(ctx context.Context, where string, arg any) (User, error) {
	query := `
		SELECT
			u.id, u.role_id, r.code, r.name, u.code, u.name, u.email, u.phone,
			u.password_hash, u.status, u.must_change_password, u.password_changed_at,
			u.last_login_at, u.deactivated_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE ` + where + ` AND u.deleted_at IS NULL
		LIMIT 1`
	user, err := scanUser(r.db.QueryRowContext(ctx, query, arg))
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	permissions, err := r.UserPermissions(ctx, user.ID)
	if err != nil {
		return User{}, err
	}
	user.Permissions = permissions
	return user, nil
}

func (r *Repository) UserPermissions(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN users u ON u.role_id = rp.role_id
		WHERE u.id = ?
		ORDER BY p.code`, userID)
	if err != nil {
		return nil, fmt.Errorf("ambil permission user: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permission: %w", err)
	}
	return permissions, nil
}

func (r *Repository) RoleIDByCode(ctx context.Context, code string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, "SELECT id FROM roles WHERE code = ? AND deleted_at IS NULL", code).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return id, nil
}

type CreateUserInput struct {
	RoleCode           string
	Code               string
	Name               string
	Email              string
	Phone              string
	PasswordHash       string
	Status             string
	MustChangePassword bool
}

func (r *Repository) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	roleID, err := r.RoleIDByCode(ctx, input.RoleCode)
	if err != nil {
		return User{}, err
	}
	status := input.Status
	if status == "" {
		status = UserStatusActive
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO users
			(role_id, code, name, email, phone, password_hash, status, must_change_password, password_changed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roleID,
		nullableString(input.Code),
		strings.TrimSpace(input.Name),
		strings.ToLower(strings.TrimSpace(input.Email)),
		nullableString(input.Phone),
		input.PasswordHash,
		status,
		input.MustChangePassword,
		time.Now().UTC(),
	)
	if err != nil {
		return User{}, mapDuplicateUserError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("ambil user id: %w", err)
	}
	return r.FindUserByID(ctx, id)
}

func (r *Repository) UpdateSales(ctx context.Context, id int64, input UpdateSalesRequest) (User, error) {
	current, err := r.FindUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	if current.RoleCode != RoleSales {
		return User{}, ErrNotFound
	}

	code := current.Code
	name := current.Name
	email := current.Email
	phone := current.Phone
	if input.Code != nil {
		code = nullableString(*input.Code)
	}
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
	}
	if input.Email != nil {
		email = strings.ToLower(strings.TrimSpace(*input.Email))
	}
	if input.Phone != nil {
		phone = nullableString(*input.Phone)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE users
		SET code = ?, name = ?, email = ?, phone = ?
		WHERE id = ? AND deleted_at IS NULL`,
		code, name, email, phone, id,
	)
	if err != nil {
		return User{}, mapDuplicateUserError(err)
	}
	return r.FindUserByID(ctx, id)
}

func (r *Repository) UpdateUserProfile(ctx context.Context, id int64, input UpdateProfileRequest) (User, error) {
	current, err := r.FindUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	name := current.Name
	email := current.Email
	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		name = strings.TrimSpace(*input.Name)
	}
	if input.Email != nil && strings.TrimSpace(*input.Email) != "" {
		email = strings.ToLower(strings.TrimSpace(*input.Email))
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE users
		SET name = ?, email = ?
		WHERE id = ? AND deleted_at IS NULL`,
		name, email, id,
	)
	if err != nil {
		return User{}, mapDuplicateUserError(err)
	}
	return r.FindUserByID(ctx, id)
}

func (r *Repository) ListSales(ctx context.Context, status string) ([]User, int64, error) {
	args := []any{RoleSales}
	where := "r.code = ? AND u.deleted_at IS NULL"
	if status != "" {
		where += " AND u.status = ?"
		args = append(args, strings.ToUpper(strings.TrimSpace(status)))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id, u.role_id, r.code, r.name, u.code, u.name, u.email, u.phone,
			u.password_hash, u.status, u.must_change_password, u.password_changed_at,
			u.last_login_at, u.deactivated_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where+`
		ORDER BY u.created_at DESC, u.id DESC`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListSupervisors returns all users with role SUPERVISOR, optionally filtered by status.
func (r *Repository) ListSupervisors(ctx context.Context, status string) ([]User, int64, error) {
	args := []any{RoleSupervisor}
	where := "r.code = ? AND u.deleted_at IS NULL"
	if status != "" {
		where += " AND u.status = ?"
		args = append(args, strings.ToUpper(strings.TrimSpace(status)))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id, u.role_id, r.code, r.name, u.code, u.name, u.email, u.phone,
			u.password_hash, u.status, u.must_change_password, u.password_changed_at,
			u.last_login_at, u.deactivated_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where+`
		ORDER BY u.created_at DESC, u.id DESC`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListAdmins returns all users with role ADMIN, optionally filtered by status.
func (r *Repository) ListAdmins(ctx context.Context, status string) ([]User, int64, error) {
	args := []any{RoleAdmin}
	where := "r.code = ? AND u.deleted_at IS NULL"
	if status != "" {
		where += " AND u.status = ?"
		args = append(args, strings.ToUpper(strings.TrimSpace(status)))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id, u.role_id, r.code, r.name, u.code, u.name, u.email, u.phone,
			u.password_hash, u.status, u.must_change_password, u.password_changed_at,
			u.last_login_at, u.deactivated_at, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE `+where+`
		ORDER BY u.created_at DESC, u.id DESC`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *Repository) SetUserStatus(ctx context.Context, id int64, status string) (User, error) {
	current, err := r.FindUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	if current.RoleCode != RoleSales {
		return User{}, ErrNotFound
	}

	var deactivatedAt sql.NullTime
	if status == UserStatusInactive {
		deactivatedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE users
		SET status = ?, deactivated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		status, deactivatedAt, id,
	)
	if err != nil {
		return User{}, err
	}
	return r.FindUserByID(ctx, id)
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, passwordHash string, mustChange bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, must_change_password = ?, password_changed_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		passwordHash, mustChange, time.Now().UTC(), id,
	)
	return err
}

func (r *Repository) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET last_login_at = ? WHERE id = ?", time.Now().UTC(), id)
	return err
}

func (r *Repository) CreateSession(ctx context.Context, userID int64, refreshHash, family, ip, userAgent string, expiresAt time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_sessions
			(user_id, refresh_token_hash, refresh_token_family, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, refreshHash, family, expiresAt, nullableString(ip), nullableString(userAgent),
	)
	if err != nil {
		return 0, fmt.Errorf("buat auth session: %w", err)
	}
	return result.LastInsertId()
}

func (r *Repository) FindSessionByRefreshHash(ctx context.Context, refreshHash string) (Session, error) {
	var session Session
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, refresh_token_hash, refresh_token_family, expires_at, revoked_at
		FROM auth_sessions
		WHERE refresh_token_hash = ?
		LIMIT 1`, refreshHash).
		Scan(&session.ID, &session.UserID, &session.RefreshTokenHash, &session.RefreshTokenFamily, &session.ExpiresAt, &session.RevokedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Session{}, ErrInvalidToken
		}
		return Session{}, err
	}
	return session, nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?)
		WHERE id = ?`, time.Now().UTC(), sessionID)
	return err
}

func (r *Repository) ReplaceSession(ctx context.Context, oldSessionID int64, userID int64, refreshHash, family, ip, userAgent string, expiresAt time.Time) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO auth_sessions
			(user_id, refresh_token_hash, refresh_token_family, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, refreshHash, family, expiresAt, nullableString(ip), nullableString(userAgent),
	)
	if err != nil {
		return 0, err
	}
	newSessionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?, replaced_by_session_id = ?, last_used_at = ?
		WHERE id = ? AND revoked_at IS NULL`,
		time.Now().UTC(), newSessionID, time.Now().UTC(), oldSessionID,
	)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newSessionID, nil
}

func (r *Repository) RevokeUserSessions(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, ?)
		WHERE user_id = ? AND revoked_at IS NULL`, time.Now().UTC(), userID)
	return err
}

func (r *Repository) Audit(ctx context.Context, entry AuditEntry) error {
	beforeJSON, err := nullableJSON(entry.Before)
	if err != nil {
		return err
	}
	afterJSON, err := nullableJSON(entry.After)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(actor_user_id, action, entity_type, entity_id, before_json, after_json, ip_address, user_agent, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ActorUserID,
		entry.Action,
		entry.EntityType,
		entry.EntityID,
		beforeJSON,
		afterJSON,
		nullableString(entry.IPAddress),
		nullableString(entry.UserAgent),
		nullableString(entry.RequestID),
	)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.RoleID,
		&user.RoleCode,
		&user.RoleName,
		&user.Code,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Status,
		&user.MustChangePassword,
		&user.PasswordChangedAt,
		&user.LastLoginAt,
		&user.DeactivatedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	return user, err
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableJSON(value any) (sql.NullString, error) {
	if value == nil {
		return sql.NullString{}, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(bytes), Valid: true}, nil
}

func mapDuplicateUserError(err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "uq_users_email"):
		return ErrEmailAlreadyUsed
	case strings.Contains(message, "uq_users_code"):
		return ErrCodeAlreadyUsed
	case strings.Contains(message, "Duplicate entry"):
		return ErrEmailAlreadyUsed
	default:
		return err
	}
}
