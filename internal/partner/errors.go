package partner

import "errors"

var (
	ErrNotFound          = errors.New("partner: data not found")
	ErrForbidden         = errors.New("partner: forbidden")
	ErrDuplicatePartner  = errors.New("partner: partner code already exists")
	ErrDuplicateType     = errors.New("partner: partner type code already exists")
	ErrPartnerTypeInUse  = errors.New("partner: jenis mitra masih digunakan oleh mitra aktif")
	ErrInvalidAssignment = errors.New("partner: invalid partner assignment (only one active PIC allowed)")
	ErrDuplicateReferral = errors.New("partner: referral already exists for this partner-lead pair")
	ErrInvalidStatus     = errors.New("partner: invalid status")

	ErrInvalidMoney            = errors.New("partner: nilai uang harus decimal valid")
	ErrInvalidCommissionRate   = errors.New("partner: commission rate must be a decimal between 0 and 100")
	ErrInvalidCommissionMode   = errors.New("partner: commission mode must be PERCENTAGE or FIXED")
	ErrInvalidCommissionStatus = errors.New("partner: invalid commission status transition")
	ErrCommissionAlreadyExists = errors.New("partner: commission already exists for this closing")

	ErrInvalidCommissionTier = errors.New("partner: invalid commission tier configuration")
	ErrNoMatchingTier        = errors.New("partner: no commission tier matches this closing count")
	ErrCommissionInPayout    = errors.New("partner: commission is already reserved in an active payout")
	ErrNoPayableCommissions  = errors.New("partner: no approved commissions available to pay out")
	ErrMixedCurrency         = errors.New("partner: cannot batch commissions with different currencies into one payout")
	ErrInvalidPayoutStatus   = errors.New("partner: invalid payout status transition")
)
