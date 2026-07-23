package catalog

import "errors"

var (
	ErrNotFound       = errors.New("data tidak ditemukan")
	ErrForbidden      = errors.New("akses ditolak")
	ErrInvalidSort    = errors.New("sort tidak valid")
	ErrInvalidDecimal = errors.New("nilai uang harus decimal valid")
	ErrInvalidDate    = errors.New("tanggal harus format YYYY-MM-DD")
	ErrInvalidTenure  = errors.New("tenure_months harus lebih dari 0")
	ErrInvalidCharge  = errors.New("charge_type harus FREE atau PAID")
	ErrCodeExists     = errors.New("kode sudah digunakan")
	ErrEmptyBulk      = errors.New("payload bulk tidak boleh kosong")
)
