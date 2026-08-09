package lead

import "errors"

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrForbidden         = errors.New("akses ditolak")
	ErrInvalidTransition = errors.New("perpindahan ownership tidak valid")
	ErrInvalidSort       = errors.New("parameter sort tidak valid")
	ErrEmptyBulk         = errors.New("minimal satu data wajib dikirim")
	ErrUserNotValid      = errors.New("user tujuan tidak valid atau tidak aktif")
	ErrLeadAlreadyExists = errors.New("lead untuk owner ini sudah ada")
	ErrTestingAccount    = errors.New("akun testing tidak boleh masuk pipeline lead/sales")
)
