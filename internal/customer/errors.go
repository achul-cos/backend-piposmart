package customer

import "errors"

var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrCodeAlreadyUsed = errors.New("kode sudah digunakan")
	ErrInvalidPhone    = errors.New("nomor telepon tidak valid")
	ErrInvalidSort     = errors.New("parameter sort tidak valid")
	ErrEmptyBulk       = errors.New("minimal satu data wajib dikirim")
	ErrForbidden       = errors.New("akses ditolak")
)
