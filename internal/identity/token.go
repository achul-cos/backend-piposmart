package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/config"
)

type TokenManager struct {
	cfg config.AuthConfig
}

type AccessClaims struct {
	Issuer      string   `json:"iss"`
	Subject     string   `json:"sub"`
	ExpiresAt   int64    `json:"exp"`
	IssuedAt    int64    `json:"iat"`
	UserID      int64    `json:"uid"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func NewTokenManager(cfg config.AuthConfig) TokenManager {
	return TokenManager{cfg: cfg}
}

func (m TokenManager) CreateAccessToken(user User, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(m.cfg.AccessTTL)
	claims := AccessClaims{
		Issuer:      m.cfg.Issuer,
		Subject:     strconv.FormatInt(user.ID, 10),
		ExpiresAt:   expiresAt.Unix(),
		IssuedAt:    now.Unix(),
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.RoleCode,
		Permissions: user.Permissions,
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", time.Time{}, err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := sign(unsigned, m.cfg.AccessSecret)
	return unsigned + "." + signature, expiresAt, nil
}

func (m TokenManager) ParseAccessToken(token string, now time.Time) (AccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AccessClaims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expected := sign(unsigned, m.cfg.AccessSecret)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return AccessClaims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AccessClaims{}, ErrInvalidToken
	}
	var claims AccessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return AccessClaims{}, ErrInvalidToken
	}
	if claims.Issuer != m.cfg.Issuer || claims.UserID == 0 {
		return AccessClaims{}, ErrInvalidToken
	}
	if now.Unix() >= claims.ExpiresAt {
		return AccessClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func GenerateUUIDLike() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}

func sign(unsigned, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("authorization header kosong")
	}
	kind, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(kind, "Bearer") || strings.TrimSpace(token) == "" {
		return "", errors.New("authorization header harus Bearer token")
	}
	return strings.TrimSpace(token), nil
}
