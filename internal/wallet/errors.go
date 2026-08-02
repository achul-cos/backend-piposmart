package wallet

import "errors"

var (
	ErrNotFound            = errors.New("data wallet tidak ditemukan")
	ErrForbidden           = errors.New("akses ditolak")
	ErrInvalidSort         = errors.New("sort tidak valid")
	ErrInvalidMoney        = errors.New("nilai uang harus decimal valid")
	ErrInvalidRequest      = errors.New("request wallet tidak valid")
	ErrInsufficientBalance = errors.New("saldo wallet tidak mencukupi")
	ErrIdempotencyRequired = errors.New("idempotency_key atau external_reference wajib dikirim")
	ErrInvalidDirection    = errors.New("arah transaksi tidak valid")
	ErrOwnerNotFound       = errors.New("owner tidak ditemukan")
	ErrLedgerOutOfSync     = errors.New("balance wallet tidak cocok dengan ledger")
	ErrTopupNotPending     = errors.New("top up tidak dalam status PENDING")
)
