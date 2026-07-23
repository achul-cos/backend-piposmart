package identity

import (
	"testing"
	"time"

	"backend_crm_piposmart/internal/platform/config"
)

func TestAccessTokenCanBeCreatedAndParsed(t *testing.T) {
	manager := NewTokenManager(config.AuthConfig{
		Issuer:       "crm-test",
		AccessSecret: "test-secret-that-has-at-least-32-byte",
		AccessTTL:    15 * time.Minute,
	})
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	user := User{ID: 9, Email: "sales@example.test", RoleCode: RoleSales, Permissions: []string{"leads.work"}}

	token, _, err := manager.CreateAccessToken(user, now)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	claims, err := manager.ParseAccessToken(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.UserID != user.ID || claims.Role != RoleSales {
		t.Fatalf("claims tidak sesuai: %+v", claims)
	}
}

func TestAccessTokenRejectsExpiredToken(t *testing.T) {
	manager := NewTokenManager(config.AuthConfig{
		Issuer:       "crm-test",
		AccessSecret: "test-secret-that-has-at-least-32-byte",
		AccessTTL:    time.Minute,
	})
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	token, _, err := manager.CreateAccessToken(User{ID: 1, Email: "admin@example.test", RoleCode: RoleAdmin}, now)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	if _, err := manager.ParseAccessToken(token, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expired token seharusnya ditolak")
	}
}
