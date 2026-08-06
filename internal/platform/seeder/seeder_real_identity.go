package seeder

import (
	"regexp"
	"strings"
)

// reNameBeforeCS matches "Magdalena ( CS 5 )" / "Kristina (CS 3)" -> captures "Magdalena".
var reNameBeforeCS = regexp.MustCompile(`(?i)^(.+?)\s*\(\s*CS\s*\d+\s*\)\s*$`)

// reCSPrefixName matches "CS 2 - Lidya" / "CS 4 RISKY" / "CS 1- Septi" -> captures the name part.
// Requires at least one non-space character after the optional dash, otherwise a bare "CS 2"
// (no name attached) falls through unmatched and is kept as its own identity.
var reCSPrefixName = regexp.MustCompile(`(?i)^CS\s*\d+\s*[-–]?\s*(\S.*)$`)

// normalizeSalesName extracts a stable, human display name from the messy "PIC"/"Share N" values
// found in the real Owner & Outlet Excel export (see docs/sprint-16f). Real examples observed:
//
//	"CS 2 - Lidya"        -> "Lidya"
//	"CS 4 RISKY"          -> "Risky"
//	"CS 2 CINDY"          -> "Cindy"
//	"CS 1- Septi"         -> "Septi"
//	"Magdalena ( CS 5 )"  -> "Magdalena"
//	"Kristina ( CS 3 )"   -> "Kristina"
//	"Meta"                -> "Meta"
//	"CS 2"                -> "CS 2"   (no name attached, kept as its own identity)
//	"TRAINEE CS 1"        -> "Trainee Cs 1" (no name attached, kept as its own identity)
//	"Perusahaan"          -> "Perusahaan"
//	"Akun Testing"        -> "Akun Testing"
//
// Different raw spellings of the same person (e.g. "CS 4 RISKY" and "CS 4 - Risky") normalize to
// the same display name, so they resolve to one sales account rather than duplicates. Different
// people who happened to occupy the same CS slot number at different times (a real turnover
// pattern in this data) are intentionally kept as separate accounts, since identity here is keyed
// by person name, not by desk/slot number.
func normalizeSalesName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if m := reNameBeforeCS.FindStringSubmatch(raw); m != nil {
		return titleCaseName(m[1])
	}
	if m := reCSPrefixName.FindStringSubmatch(raw); m != nil {
		return titleCaseName(m[1])
	}
	return titleCaseName(raw)
}

// salesIdentityKey is the case-insensitive dedup key for a normalized sales name.
func salesIdentityKey(raw string) string {
	return strings.ToUpper(normalizeSalesName(raw))
}

func titleCaseName(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		lower := strings.ToLower(f)
		fields[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(fields, " ")
}
