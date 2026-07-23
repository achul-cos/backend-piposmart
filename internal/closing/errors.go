package closing

import "errors"

var (
	ErrNotFound            = errors.New("data closing tidak ditemukan")
	ErrForbidden           = errors.New("akses ditolak")
	ErrInvalidSort         = errors.New("sort tidak valid")
	ErrInvalidMoney        = errors.New("nilai uang harus decimal valid")
	ErrInvalidStatus       = errors.New("status closing tidak valid")
	ErrInvalidPromotion    = errors.New("promo tidak eligible untuk plan yang dipilih")
	ErrAlreadyHasClosing   = errors.New("lead sudah memiliki closing pending atau confirmed")
	ErrFinalAmountNegative = errors.New("final amount tidak boleh negatif")
	ErrInvalidRequest      = errors.New("request closing tidak valid")
	ErrLeadHasNoPIC        = errors.New("lead belum memiliki PIC Sales aktif")
)
