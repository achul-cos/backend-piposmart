package transfer

import "errors"

var (
	ErrNotFound        = errors.New("data transfer tidak ditemukan")
	ErrForbidden       = errors.New("akses ditolak")
	ErrInvalidMoney    = errors.New("nilai uang harus decimal valid")
	ErrInvalidSort     = errors.New("sort tidak valid")
	ErrAlreadyMatched  = errors.New("transfer sudah matched")
	ErrPaymentNotOwner = errors.New("wallet payment bukan milik owner transfer ini")
	ErrOwnerNotFound   = errors.New("owner tidak ditemukan")
)
