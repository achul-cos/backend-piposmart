package importing

import "errors"

var (
	ErrNotFound              = errors.New("importing: data not found")
	ErrForbidden             = errors.New("importing: forbidden")
	ErrInvalidFileType       = errors.New("importing: only .xlsx files are accepted")
	ErrFileTooLarge          = errors.New("importing: file exceeds the maximum allowed size")
	ErrEmptyFile             = errors.New("importing: uploaded file is empty")
	ErrProfileRequired       = errors.New("importing: profile could not be auto-detected; specify it explicitly")
	ErrProfileHeaderMismatch = errors.New("importing: file headers do not match the declared profile")
	ErrUnknownProfile        = errors.New("importing: unknown profile")
	ErrInvalidBatchStatus    = errors.New("importing: batch is not in a state that allows this action")
)
