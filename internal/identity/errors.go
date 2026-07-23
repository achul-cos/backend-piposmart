package identity

import "errors"

var (
	ErrInvalidCredentials = errors.New("email atau password tidak valid")
	ErrInactiveUser       = errors.New("user tidak aktif")
	ErrInvalidToken       = errors.New("token tidak valid")
	ErrForbidden          = errors.New("akses ditolak")
	ErrNotFound           = errors.New("data tidak ditemukan")
	ErrEmailAlreadyUsed   = errors.New("email sudah digunakan")
	ErrCodeAlreadyUsed    = errors.New("kode sudah digunakan")
	ErrWeakPassword       = errors.New("password minimal 8 karakter")
)
