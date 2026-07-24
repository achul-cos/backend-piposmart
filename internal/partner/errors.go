package partner

import "errors"

var (
	ErrNotFound          = errors.New("partner: data not found")
	ErrForbidden         = errors.New("partner: forbidden")
	ErrDuplicatePartner  = errors.New("partner: partner code already exists")
	ErrDuplicateType     = errors.New("partner: partner type code already exists")
	ErrInvalidAssignment = errors.New("partner: invalid partner assignment (only one active PIC allowed)")
	ErrDuplicateReferral = errors.New("partner: referral already exists for this partner-lead pair")
	ErrInvalidStatus     = errors.New("partner: invalid status")
)