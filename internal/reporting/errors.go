package reporting

import "errors"

var (
	ErrForbidden        = errors.New("reporting: akses ditolak")
	ErrInvalidReportKey = errors.New("reporting: report_key tidak didukung")
	ErrInvalidFormat    = errors.New("reporting: format export tidak didukung")
	ErrExportNotFound   = errors.New("reporting: export tidak ditemukan")
	ErrExportNotReady   = errors.New("reporting: file export belum siap diunduh")
)

