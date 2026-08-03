package subscription

import "errors"

var (
	ErrNotFound               = errors.New("data subscription tidak ditemukan")
	ErrForbidden              = errors.New("akses ditolak")
	ErrInvalidSort            = errors.New("sort tidak valid")
	ErrInvalidMoney           = errors.New("nilai uang harus decimal valid")
	ErrInvalidRequest         = errors.New("request subscription tidak valid")
	ErrInvalidDate            = errors.New("tanggal harus format YYYY-MM-DD")
	ErrInvalidAction          = errors.New("aksi reconciliation tidak valid")
	ErrInsufficientBalance    = errors.New("saldo wallet tidak mencukupi")
	ErrIdempotencyRequired    = errors.New("idempotency_key atau external_reference wajib dikirim")
	ErrOwnerNotFound          = errors.New("owner tidak ditemukan")
	ErrWalletNotFound         = errors.New("wallet owner tidak ditemukan")
	ErrInvalidPromotion       = errors.New("promo tidak eligible untuk plan yang dipilih")
	ErrClosingMismatch        = errors.New("closing tidak sesuai dengan order atau owner")
	ErrOrderAlreadyReconciled = errors.New("order sudah direkonsiliasi")
	ErrSubscriptionNotActive  = errors.New("subscription tidak aktif untuk di-upgrade")
	ErrUpgradeNotAllowed      = errors.New("upgrade paket tidak valid")
)
