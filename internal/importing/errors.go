package importing

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound                = errors.New("importing: data not found")
	ErrForbidden               = errors.New("importing: forbidden")
	ErrInvalidFileType         = errors.New("importing: only .xlsx files are accepted")
	ErrFileTooLarge            = errors.New("importing: file exceeds the maximum allowed size")
	ErrFileUnavailable         = errors.New("importing: original file is no longer available")
	ErrEmptyFile               = errors.New("importing: uploaded file is empty")
	ErrProfileRequired         = errors.New("importing: profile could not be auto-detected; specify it explicitly")
	ErrProfileHeaderMismatch   = errors.New("importing: file headers do not match the declared profile")
	ErrUnknownProfile          = errors.New("importing: unknown profile")
	ErrSheetNameRequired       = errors.New("importing: this profile requires an explicit sheet_name (its workbook has multiple similar sheets)")
	ErrSheetNameNeedsProfile   = errors.New("importing: sheet_name requires an explicit profile to verify it against")
	ErrSheetNotFound           = errors.New("importing: declared sheet_name was not found in the workbook")
	ErrTargetSalesUserRequired = errors.New("importing: this profile requires an explicit target_sales_user_id (the sales rep is only encoded in the sheet name)")
	ErrInvalidBatchStatus      = errors.New("importing: batch is not in a state that allows this action")
)

type BatchStatusActionError struct {
	Action          string
	CurrentStatus   string
	AllowedStatuses []string
}

func (e *BatchStatusActionError) Error() string {
	if e == nil {
		return ErrInvalidBatchStatus.Error()
	}
	allowed := strings.Join(e.AllowedStatuses, ", ")
	if allowed == "" {
		return fmt.Sprintf("%s (action=%s, current_status=%s)", ErrInvalidBatchStatus.Error(), e.Action, e.CurrentStatus)
	}
	return fmt.Sprintf("%s (action=%s, current_status=%s, allowed_statuses=%s)", ErrInvalidBatchStatus.Error(), e.Action, e.CurrentStatus, allowed)
}

func (e *BatchStatusActionError) Unwrap() error {
	return ErrInvalidBatchStatus
}

func newBatchStatusActionError(action string, currentStatus string, allowedStatuses ...string) error {
	return &BatchStatusActionError{
		Action:          action,
		CurrentStatus:   currentStatus,
		AllowedStatuses: allowedStatuses,
	}
}
