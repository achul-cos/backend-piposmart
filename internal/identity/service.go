package identity

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/password"
)

type Service struct {
	repo   *Repository
	tokens TokenManager
	cfg    config.Config
	now    func() time.Time
}

type RequestMeta struct {
	Actor     *User
	IPAddress string
	UserAgent string
	RequestID string
}

func NewService(repo *Repository, cfg config.Config) *Service {
	return &Service{
		repo:   repo,
		tokens: NewTokenManager(cfg.Auth),
		cfg:    cfg,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Login(ctx context.Context, req LoginRequest, meta RequestMeta) (AuthTokenResponse, error) {
	user, err := s.repo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return AuthTokenResponse{}, ErrInvalidCredentials
		}
		return AuthTokenResponse{}, err
	}
	if user.Status != UserStatusActive {
		return AuthTokenResponse{}, ErrInactiveUser
	}

	ok, err := password.VerifyArgon2id(req.Password, user.PasswordHash)
	if err != nil || !ok {
		return AuthTokenResponse{}, ErrInvalidCredentials
	}
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		return AuthTokenResponse{}, err
	}
	user, err = s.repo.FindUserByID(ctx, user.ID)
	if err != nil {
		return AuthTokenResponse{}, err
	}
	response, err := s.issueTokenPair(ctx, user, "", meta)
	if err != nil {
		return AuthTokenResponse{}, err
	}
	_ = s.repo.Audit(ctx, AuditEntry{
		ActorUserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Action:      "auth.login",
		EntityType:  "users",
		EntityID:    sql.NullInt64{Int64: user.ID, Valid: true},
		IPAddress:   meta.IPAddress,
		UserAgent:   meta.UserAgent,
		RequestID:   meta.RequestID,
	})
	return response, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (AuthTokenResponse, error) {
	session, err := s.repo.FindSessionByRefreshHash(ctx, HashRefreshToken(refreshToken))
	if err != nil {
		return AuthTokenResponse{}, ErrInvalidToken
	}
	if session.RevokedAt.Valid || !session.ExpiresAt.After(s.now()) {
		return AuthTokenResponse{}, ErrInvalidToken
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return AuthTokenResponse{}, err
	}
	if user.Status != UserStatusActive {
		return AuthTokenResponse{}, ErrInactiveUser
	}
	return s.issueTokenPair(ctx, user, session.RefreshTokenFamily, meta, session.ID)
}

func (s *Service) Logout(ctx context.Context, refreshToken string, user User, meta RequestMeta) error {
	if strings.TrimSpace(refreshToken) != "" {
		session, err := s.repo.FindSessionByRefreshHash(ctx, HashRefreshToken(refreshToken))
		if err == nil {
			if err := s.repo.RevokeSession(ctx, session.ID); err != nil {
				return err
			}
		}
	}
	_ = s.repo.Audit(ctx, AuditEntry{
		ActorUserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Action:      "auth.logout",
		EntityType:  "users",
		EntityID:    sql.NullInt64{Int64: user.ID, Valid: true},
		IPAddress:   meta.IPAddress,
		UserAgent:   meta.UserAgent,
		RequestID:   meta.RequestID,
	})
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, user User, req ChangePasswordRequest, meta RequestMeta) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}
	ok, err := password.VerifyArgon2id(req.CurrentPassword, user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}
	hash, err := password.HashArgon2id(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, user.ID, hash, false); err != nil {
		return err
	}
	if err := s.repo.RevokeUserSessions(ctx, user.ID); err != nil {
		return err
	}
	_ = s.repo.Audit(ctx, AuditEntry{
		ActorUserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Action:      "auth.change_password",
		EntityType:  "users",
		EntityID:    sql.NullInt64{Int64: user.ID, Valid: true},
		IPAddress:   meta.IPAddress,
		UserAgent:   meta.UserAgent,
		RequestID:   meta.RequestID,
	})
	return nil
}

func (s *Service) BootstrapAdmin(ctx context.Context) (User, error) {
	if strings.TrimSpace(s.cfg.Bootstrap.AdminEmail) == "" || strings.TrimSpace(s.cfg.Bootstrap.AdminPassword) == "" {
		return User{}, errors.New("BOOTSTRAP_ADMIN_EMAIL dan BOOTSTRAP_ADMIN_PASSWORD wajib diisi")
	}
	if err := validatePassword(s.cfg.Bootstrap.AdminPassword); err != nil {
		return User{}, err
	}
	existing, err := s.repo.FindUserByEmail(ctx, s.cfg.Bootstrap.AdminEmail)
	if err == nil {
		if existing.RoleCode != RoleAdmin {
			return User{}, errors.New("email bootstrap sudah dipakai oleh role non-admin")
		}
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return User{}, err
	}

	hash, err := password.HashArgon2id(s.cfg.Bootstrap.AdminPassword)
	if err != nil {
		return User{}, err
	}
	return s.repo.CreateUser(ctx, CreateUserInput{
		RoleCode:           RoleAdmin,
		Code:               "ADM-BOOTSTRAP",
		Name:               s.cfg.Bootstrap.AdminName,
		Email:              s.cfg.Bootstrap.AdminEmail,
		Phone:              s.cfg.Bootstrap.AdminPhone,
		PasswordHash:       hash,
		Status:             UserStatusActive,
		MustChangePassword: false,
	})
}

func (s *Service) ListSales(ctx context.Context, actor User, status string) (SalesListResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor && actor.RoleCode != RoleSales {
		return SalesListResponse{}, ErrForbidden
	}
	users, total, err := s.repo.ListSales(ctx, status)
	if err != nil {
		return SalesListResponse{}, err
	}
	items := make([]UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, NewUserResponse(user))
	}
	return SalesListResponse{Items: items, Total: total}, nil
}

// ListSupervisors returns all active supervisors. Accessible by ADMIN and SUPERVISOR roles.
func (s *Service) ListSupervisors(ctx context.Context, actor User, status string) (SalesListResponse, error) {
	if actor.RoleCode != RoleAdmin && actor.RoleCode != RoleSupervisor {
		return SalesListResponse{}, ErrForbidden
	}
	users, total, err := s.repo.ListSupervisors(ctx, status)
	if err != nil {
		return SalesListResponse{}, err
	}
	items := make([]UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, NewUserResponse(user))
	}
	return SalesListResponse{Items: items, Total: total}, nil
}


func (s *Service) GetSales(ctx context.Context, actor User, id int64) (UserResponse, error) {
	if !hasPermission(actor, "users.read") && !hasPermission(actor, "users.manage_sales") {
		return UserResponse{}, ErrForbidden
	}
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}
	if user.RoleCode != RoleSales {
		return UserResponse{}, ErrNotFound
	}
	return NewUserResponse(user), nil
}

func (s *Service) CreateSales(ctx context.Context, actor User, req CreateSalesRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_sales") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.createManagedUser(ctx, actor, RoleSales, req, meta, "users.create_sales")
}

func (s *Service) CreateSupervisor(ctx context.Context, actor User, req CreateUserRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_all") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.createManagedUser(ctx, actor, RoleSupervisor, req, meta, "users.create_supervisor")
}

func (s *Service) CreateAdmin(ctx context.Context, actor User, req CreateUserRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_all") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.createManagedUser(ctx, actor, RoleAdmin, req, meta, "users.create_admin")
}

func (s *Service) createManagedUser(ctx context.Context, actor User, roleCode string, req CreateUserRequest, meta RequestMeta, auditAction string) (ResetPasswordResponse, error) {
	plain := strings.TrimSpace(req.Password)
	if plain == "" {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			return ResetPasswordResponse{}, err
		}
		plain = generated
	}
	if err := validatePassword(plain); err != nil {
		return ResetPasswordResponse{}, err
	}
	hash, err := password.HashArgon2id(plain)
	if err != nil {
		return ResetPasswordResponse{}, err
	}
	user, err := s.repo.CreateUser(ctx, CreateUserInput{
		RoleCode:           roleCode,
		Code:               req.Code,
		Name:               req.Name,
		Email:              req.Email,
		Phone:              req.Phone,
		PasswordHash:       hash,
		Status:             UserStatusActive,
		MustChangePassword: true,
	})
	if err != nil {
		return ResetPasswordResponse{}, err
	}
	_ = s.auditUserChange(ctx, actor, auditAction, user.ID, nil, NewUserResponse(user), meta)
	return ResetPasswordResponse{User: NewUserResponse(user), TemporaryPassword: plain}, nil
}

func (s *Service) UpdateSales(ctx context.Context, actor User, id int64, req UpdateSalesRequest, meta RequestMeta) (UserResponse, error) {
	if !hasPermission(actor, "users.manage_sales") {
		return UserResponse{}, ErrForbidden
	}
	before, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}
	if before.RoleCode != RoleSales {
		return UserResponse{}, ErrNotFound
	}
	after, err := s.repo.UpdateSales(ctx, id, req)
	if err != nil {
		return UserResponse{}, err
	}
	_ = s.auditUserChange(ctx, actor, "users.update_sales", id, NewUserResponse(before), NewUserResponse(after), meta)
	return NewUserResponse(after), nil
}

func (s *Service) SetSalesStatus(ctx context.Context, actor User, id int64, active bool, meta RequestMeta) (UserResponse, error) {
	if !hasPermission(actor, "users.manage_sales") {
		return UserResponse{}, ErrForbidden
	}
	status := UserStatusInactive
	action := "users.deactivate_sales"
	if active {
		status = UserStatusActive
		action = "users.activate_sales"
	}
	before, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}
	after, err := s.repo.SetUserStatus(ctx, id, status)
	if err != nil {
		return UserResponse{}, err
	}
	if !active {
		if err := s.repo.RevokeUserSessions(ctx, id); err != nil {
			return UserResponse{}, err
		}
	}
	_ = s.auditUserChange(ctx, actor, action, id, NewUserResponse(before), NewUserResponse(after), meta)
	return NewUserResponse(after), nil
}

func (s *Service) ResetSalesPassword(ctx context.Context, actor User, id int64, req ResetPasswordRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_sales") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.resetManagedPassword(ctx, actor, id, req, meta, RoleSales, "users.reset_sales_password")
}

func (s *Service) ResetSupervisorPassword(ctx context.Context, actor User, id int64, req ResetPasswordRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_all") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.resetManagedPassword(ctx, actor, id, req, meta, RoleSupervisor, "users.reset_supervisor_password")
}

func (s *Service) ResetAdminPassword(ctx context.Context, actor User, id int64, req ResetPasswordRequest, meta RequestMeta) (ResetPasswordResponse, error) {
	if !hasPermission(actor, "users.manage_all") {
		return ResetPasswordResponse{}, ErrForbidden
	}
	return s.resetManagedPassword(ctx, actor, id, req, meta, RoleAdmin, "users.reset_admin_password")
}

func (s *Service) resetManagedPassword(ctx context.Context, actor User, id int64, req ResetPasswordRequest, meta RequestMeta, roleCode string, auditAction string) (ResetPasswordResponse, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return ResetPasswordResponse{}, err
	}
	if user.RoleCode != roleCode {
		return ResetPasswordResponse{}, ErrNotFound
	}
	plain := strings.TrimSpace(req.NewPassword)
	if plain == "" {
		generated, err := GenerateTemporaryPassword()
		if err != nil {
			return ResetPasswordResponse{}, err
		}
		plain = generated
	}
	if err := validatePassword(plain); err != nil {
		return ResetPasswordResponse{}, err
	}
	hash, err := password.HashArgon2id(plain)
	if err != nil {
		return ResetPasswordResponse{}, err
	}
	if err := s.repo.UpdatePassword(ctx, id, hash, true); err != nil {
		return ResetPasswordResponse{}, err
	}
	if err := s.repo.RevokeUserSessions(ctx, id); err != nil {
		return ResetPasswordResponse{}, err
	}
	updated, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return ResetPasswordResponse{}, err
	}
	_ = s.auditUserChange(ctx, actor, auditAction, id, nil, map[string]any{"user_id": id, "role": roleCode}, meta)
	return ResetPasswordResponse{User: NewUserResponse(updated), TemporaryPassword: plain}, nil
}

func (s *Service) UserFromAccessToken(ctx context.Context, token string) (User, error) {
	claims, err := s.tokens.ParseAccessToken(token, s.now())
	if err != nil {
		return User{}, err
	}
	user, err := s.repo.FindUserByID(ctx, claims.UserID)
	if err != nil {
		return User{}, err
	}
	if user.Status != UserStatusActive {
		return User{}, ErrInactiveUser
	}
	return user, nil
}

func (s *Service) issueTokenPair(ctx context.Context, user User, family string, meta RequestMeta, oldSessionID ...int64) (AuthTokenResponse, error) {
	accessToken, accessExpiresAt, err := s.tokens.CreateAccessToken(user, s.now())
	if err != nil {
		return AuthTokenResponse{}, err
	}
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return AuthTokenResponse{}, err
	}
	if family == "" {
		family, err = GenerateUUIDLike()
		if err != nil {
			return AuthTokenResponse{}, err
		}
	}
	refreshExpiresAt := s.now().Add(s.cfg.Auth.RefreshTTL)
	refreshHash := HashRefreshToken(refreshToken)
	if len(oldSessionID) > 0 && oldSessionID[0] > 0 {
		if _, err := s.repo.ReplaceSession(ctx, oldSessionID[0], user.ID, refreshHash, family, meta.IPAddress, meta.UserAgent, refreshExpiresAt); err != nil {
			return AuthTokenResponse{}, err
		}
	} else if _, err := s.repo.CreateSession(ctx, user.ID, refreshHash, family, meta.IPAddress, meta.UserAgent, refreshExpiresAt); err != nil {
		return AuthTokenResponse{}, err
	}

	return AuthTokenResponse{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		TokenType:             "Bearer",
		ExpiresIn:             int64(accessExpiresAt.Sub(s.now()).Seconds()),
		RefreshTokenExpiresAt: refreshExpiresAt,
		User:                  NewUserResponse(user),
	}, nil
}

func (s *Service) auditUserChange(ctx context.Context, actor User, action string, entityID int64, before, after any, meta RequestMeta) error {
	return s.repo.Audit(ctx, AuditEntry{
		ActorUserID: sql.NullInt64{Int64: actor.ID, Valid: true},
		Action:      action,
		EntityType:  "users",
		EntityID:    sql.NullInt64{Int64: entityID, Valid: true},
		Before:      before,
		After:       after,
		IPAddress:   meta.IPAddress,
		UserAgent:   meta.UserAgent,
		RequestID:   meta.RequestID,
	})
}

func hasPermission(user User, permission string) bool {
	for _, item := range user.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func validatePassword(value string) error {
	if len(value) < 8 {
		return ErrWeakPassword
	}
	return nil
}

func GenerateTemporaryPassword() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate temporary password: %w", err)
	}
	return "Tmp-" + base64.RawURLEncoding.EncodeToString(bytes), nil
}
