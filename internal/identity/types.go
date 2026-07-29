package identity

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	UserStatusActive   = "ACTIVE"
	UserStatusInactive = "INACTIVE"
)

type User struct {
	ID                 int64
	RoleID             int64
	RoleCode           string
	RoleName           string
	Code               sql.NullString
	Name               string
	Email              string
	Phone              sql.NullString
	PasswordHash       string
	Status             string
	MustChangePassword bool
	PasswordChangedAt  sql.NullTime
	LastLoginAt        sql.NullTime
	DeactivatedAt      sql.NullTime
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Permissions        []string
}

type Session struct {
	ID                 int64
	UserID             int64
	RefreshTokenHash   string
	RefreshTokenFamily string
	ExpiresAt          time.Time
	RevokedAt          sql.NullTime
}

type UserResponse struct {
	ID                 int64     `json:"id"`
	Code               string    `json:"code,omitempty"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Phone              string    `json:"phone,omitempty"`
	Role               string    `json:"role"`
	Status             string    `json:"status"`
	MustChangePassword bool      `json:"must_change_password"`
	Permissions        []string  `json:"permissions,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func NewUserResponse(user User) UserResponse {
	return UserResponse{
		ID:                 user.ID,
		Code:               user.Code.String,
		Name:               user.Name,
		Email:              user.Email,
		Phone:              user.Phone.String,
		Role:               user.RoleCode,
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		Permissions:        user.Permissions,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type AuthTokenResponse struct {
	AccessToken           string       `json:"access_token"`
	RefreshToken          string       `json:"refresh_token"`
	TokenType             string       `json:"token_type"`
	ExpiresIn             int64        `json:"expires_in"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  UserResponse `json:"user"`
}

type CreateUserRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type CreateSalesRequest = CreateUserRequest

type UpdateSalesRequest struct {
	Code  *string `json:"code"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type ResetPasswordResponse struct {
	User              UserResponse `json:"user"`
	TemporaryPassword string       `json:"temporary_password,omitempty"`
}

type SalesListResponse struct {
	Items []UserResponse `json:"items"`
	Total int64          `json:"total"`
}
