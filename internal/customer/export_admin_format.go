package customer

import (
	"fmt"
	"time"
)

// ToAdminOwnerOutletRows remaps ExportOwnerOutlets' DB rows (one per owner+outlet pair) into the
// column keys reporting.GetAdminOwnerOutletColumns expects — the same layout admins are used to
// from the original Excel report, but sourced entirely from the database.
//
// "Kategori Akun" stays derived from the row order admins were already used to, except that a
// persisted testing-account flag now wins absolutely: testing owners export as "Akun Testing" so
// they can be identified immediately in the sheet as non-prospects.
func ToAdminOwnerOutletRows(rows []map[string]any) []map[string]any {
	earliestOutletAt := make(map[string]string, len(rows))
	for _, row := range rows {
		code := stringVal(row["owner_code"])
		createdAt := stringVal(row["outlet_created_at"])
		if createdAt == "" {
			continue
		}
		if cur, ok := earliestOutletAt[code]; !ok || createdAt < cur {
			earliestOutletAt[code] = createdAt
		}
	}

	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		code := stringVal(row["owner_code"])
		outletCreatedAt := stringVal(row["outlet_created_at"])
		kategori := "Outlet Baru"
		if outletCreatedAt != "" && outletCreatedAt == earliestOutletAt[code] {
			kategori = "Akun Baru"
		}
		if boolVal(row["is_testing_account"]) {
			kategori = "Akun Testing"
		}
		namaPenginput := stringVal(row["entered_by_name"])
		if namaPenginput == "" {
			namaPenginput = "-"
		}
		kodeBaris := stringVal(row["row_code"])
		if kodeBaris == "" {
			// Fallback to owner_id if row_code is not yet populated
			if id := int64Val(row["owner_id"]); id > 0 {
				kodeBaris = fmt.Sprint(id)
			} else {
				kodeBaris = "-"
			}
		}
		ownerName := stringVal(row["owner_name"])
		if ownerName == "" {
			ownerName = "-"
		}
		ownerEmail := stringVal(row["owner_email"])
		if ownerEmail == "" {
			ownerEmail = "-"
		}
		ownerPhone := stringVal(row["owner_phone"])
		if ownerPhone == "" {
			ownerPhone = "-"
		}
		kel := stringVal(row["outlet_sub_district"])
		if kel == "" {
			kel = "-"
		}
		kec := stringVal(row["outlet_district"])
		if kec == "" {
			kec = "-"
		}
		if code == "" {
			code = "-"
		}
		item := map[string]any{
			"date_of_work":        formatAdminDate(stringVal(row["owner_created_at"])),
			"nama_penginput":      namaPenginput,
			"kategori_akun":       kategori,
			"kode_baris":          kodeBaris,
			"owner_code":          code, // code is already stringVal(row["owner_code"])
			"owner_name":          ownerName,
			"owner_email":         ownerEmail,
			"owner_phone":         ownerPhone,
			"outlet_phone":        row["outlet_phone"],
			"create_date_project": formatAdminDate(outletCreatedAt),
			"brand_name":          row["brand_name"],
			"outlet_name":         row["outlet_name"],
			"kelurahan":           kel,
			"kecamatan":           kec,
			"kota":                row["outlet_city"],
			"provinsi":            row["outlet_province"],
			"alamat_lengkap":      row["outlet_address"],
			"status_terbaru":      row["status_terbaru"],
			"akuisisi":            row["akuisisi"],
			"pic":                 row["pic"],
			"jumlah_outlet":       row["outlet_count"],
		}
		for index := 1; index <= adminOwnerOutletShareLimit; index++ {
			item[fmt.Sprintf("tanggal_dibagikan_%d", index)] = row[fmt.Sprintf("tanggal_dibagikan_%d", index)]
			item[fmt.Sprintf("share_%d", index)] = row[fmt.Sprintf("share_%d", index)]
			item[fmt.Sprintf("kategori_nasabah_%d", index)] = row[fmt.Sprintf("kategori_nasabah_%d", index)]
		}
		out = append(out, item)
	}
	return out
}

// ToAdminOwnerRows remaps the same DB rows into reporting.GetAdminOwnerColumns' layout: one row
// per unique owner (deduped, first row encountered per owner_code kept), with a wallet balance
// column instead of the per-outlet fields.
func ToAdminOwnerRows(rows []map[string]any) []map[string]any {
	earliestOutletAt := make(map[string]string, len(rows))
	enteredByNameMap := make(map[string]string, len(rows))
	for _, row := range rows {
		code := stringVal(row["owner_code"])
		createdAt := stringVal(row["outlet_created_at"])
		if createdAt == "" {
			continue
		}
		if cur, ok := earliestOutletAt[code]; !ok || createdAt < cur {
			earliestOutletAt[code] = createdAt
			enteredByNameMap[code] = stringVal(row["entered_by_name"])
		}
	}

	seen := make(map[string]bool, len(rows))
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		code := stringVal(row["owner_code"])
		if seen[code] {
			continue
		}
		seen[code] = true

		namaPenginput := enteredByNameMap[code]
		if namaPenginput == "" {
			namaPenginput = "-"
		}
		kodeBaris := stringVal(row["row_code"])
		if kodeBaris == "" {
			if id := int64Val(row["owner_id"]); id > 0 {
				kodeBaris = fmt.Sprint(id)
			} else {
				kodeBaris = "-"
			}
		}
		ownerName := stringVal(row["owner_name"])
		ownerEmail := stringVal(row["owner_email"])
		ownerPhone := stringVal(row["owner_phone"])
		kel := stringVal(row["outlet_sub_district"])
		if kel == "" {
			kel = "-"
		}
		kec := stringVal(row["outlet_district"])
		if kec == "" {
			kec = "-"
		}
		if code == "" {
			code = ""
		}

		out = append(out, map[string]any{
			"date_of_work":        formatAdminDate(stringVal(row["owner_created_at"])),
			"nama_penginput":      namaPenginput,
			"kode_baris":          kodeBaris,
			"owner_code":          code,
			"owner_name":          ownerName,
			"owner_email":         ownerEmail,
			"owner_phone":         ownerPhone,
			"create_date_project": formatAdminDate(stringVal(row["outlet_created_at"])),
			"brand_name":          row["brand_name"],
			"kelurahan":           kel,
			"kecamatan":           kec,
			"kota":                row["outlet_city"],
			"provinsi":            row["outlet_province"],
			"alamat_lengkap":      row["outlet_address"],
			"jumlah_outlet":       row["outlet_count"],
			"saldo_owner":         row["owner_balance"],
		})
	}
	return out
}

func stringVal(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func int64Val(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	case []byte:
		var out int64
		for _, ch := range val {
			if ch < '0' || ch > '9' {
				return 0
			}
			out = out*10 + int64(ch-'0')
		}
		return out
	case string:
		var out int64
		for _, ch := range val {
			if ch < '0' || ch > '9' {
				return 0
			}
			out = out*10 + int64(ch-'0')
		}
		return out
	default:
		return 0
	}
}

func boolVal(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case []byte:
		return string(val) == "1"
	case string:
		return val == "1" || val == "true" || val == "TRUE"
	default:
		return false
	}
}

// formatAdminDate turns the repository's RFC3339 timestamp strings into the "02/01/2006" style
// admins expect (matching the export builder's own date formatting elsewhere in export.go).
func formatAdminDate(rfc3339 string) string {
	if rfc3339 == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", rfc3339)
	if err != nil {
		return rfc3339
	}
	return t.Format("02/01/2006")
}
