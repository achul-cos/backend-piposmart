package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"backend_crm_piposmart/internal/platform/factory"
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

// seedRealIdentities scans the whole real Owner & Outlet dataset up front and creates one real
// user account per distinct "Nama Penginput" (ADMIN) and per distinct normalized PIC/Share sales
// identity (SALES, including non-person placeholder labels like "Perusahaan"/"Akun Testing" and
// the "Tanpa PIC" fallback for rows with no assignment info at all), plus the single hardcoded
// "Supervisor CRM" account. Doing this once up front (instead of per-row) keeps the ~11.8K row
// main loop to O(1) map lookups.
func seedRealIdentities(ctx context.Context, tx *sql.Tx, fake *factory.Factory, rows []realOwnerExcelRow) (adminIDByName map[string]int64, salesEmailByKey map[string]string, supervisorID int64, err error) {
	adminNames := map[string]bool{}
	salesNames := map[string]string{} // identity key -> first-seen display name
	salesNames[salesIdentityKey(noPICIdentity)] = normalizeSalesName(noPICIdentity)

	addSales := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		key := salesIdentityKey(raw)
		if key == "" {
			return
		}
		if _, exists := salesNames[key]; !exists {
			salesNames[key] = normalizeSalesName(raw)
		}
	}

	for _, row := range rows {
		if name := strings.TrimSpace(row.NamaPenginput); name != "" {
			adminNames[name] = true
		}
		addSales(row.PIC)
		for _, sh := range row.Shares {
			addSales(sh.RawName)
		}
	}

	supUser := fake.BuildRealUser("SUPERVISOR", "Supervisor CRM", 1)
	supervisorID, err = fake.CreateUser(ctx, tx, supUser)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("create supervisor: %w", err)
	}

	sortedAdmins := make([]string, 0, len(adminNames))
	for name := range adminNames {
		sortedAdmins = append(sortedAdmins, name)
	}
	sort.Strings(sortedAdmins)

	adminIDByName = make(map[string]int64, len(adminNames))
	for i, name := range sortedAdmins {
		u := fake.BuildRealUser("ADMIN", name, i+1)
		id, err := fake.CreateUser(ctx, tx, u)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("create admin %q: %w", name, err)
		}
		adminIDByName[name] = id
	}

	sortedSalesKeys := make([]string, 0, len(salesNames))
	for key := range salesNames {
		sortedSalesKeys = append(sortedSalesKeys, key)
	}
	sort.Strings(sortedSalesKeys)

	salesEmailByKey = make(map[string]string, len(salesNames))
	for i, key := range sortedSalesKeys {
		displayName := salesNames[key]
		u := fake.BuildRealUser("SALES", displayName, i+1)
		if _, err := fake.CreateUser(ctx, tx, u); err != nil {
			return nil, nil, 0, fmt.Errorf("create sales %q: %w", displayName, err)
		}
		salesEmailByKey[key] = u.Email
	}

	return adminIDByName, salesEmailByKey, supervisorID, nil
}

func titleCaseName(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		lower := strings.ToLower(f)
		fields[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(fields, " ")
}
