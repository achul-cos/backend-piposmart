package activity

import "errors"

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrForbidden         = errors.New("akses ditolak")
	ErrInvalidSort       = errors.New("sort tidak valid")
	ErrInvalidScore      = errors.New("score remark harus 0 sampai 3")
	ErrInvalidType       = errors.New("tipe aktivitas tidak valid")
	ErrInteractionEmpty  = errors.New("minimal salah satu status call/chat atau type lama wajib diisi")
	ErrRemarkNotFound    = errors.New("remark reason tidak ditemukan")
	ErrInvalidTransition = errors.New("transisi status tidak valid")
	ErrLeadHasNoPIC      = errors.New("lead belum memiliki supervisor aktif")
)
