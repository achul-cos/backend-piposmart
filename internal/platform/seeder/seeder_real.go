package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/factory"

	"github.com/xuri/excelize/v2"
)

type realOwnerExcelRow struct {
	OwnerCode   string
	OwnerName   string
	OwnerPhone  string
	OwnerEmail  string
	BrandName   string
	OutletName  string
	OutletPhone string
	City        string
	Province    string
	Address     string
	DateOfWork  time.Time
}

func parseMonthIndonesian(mStr string) time.Month {
	mStr = strings.ToLower(strings.TrimSpace(mStr))
	switch {
	case strings.HasPrefix(mStr, "jan"):
		return time.January
	case strings.HasPrefix(mStr, "feb"):
		return time.February
	case strings.HasPrefix(mStr, "mar"):
		return time.March
	case strings.HasPrefix(mStr, "apr"):
		return time.April
	case strings.HasPrefix(mStr, "mei"), strings.HasPrefix(mStr, "may"):
		return time.May
	case strings.HasPrefix(mStr, "jun"):
		return time.June
	case strings.HasPrefix(mStr, "jul"):
		return time.July
	case strings.HasPrefix(mStr, "agust"), strings.HasPrefix(mStr, "agus"), strings.HasPrefix(mStr, "aug"):
		return time.August
	case strings.HasPrefix(mStr, "sep"):
		return time.September
	case strings.HasPrefix(mStr, "okt"), strings.HasPrefix(mStr, "oct"):
		return time.October
	case strings.HasPrefix(mStr, "nov"):
		return time.November
	case strings.HasPrefix(mStr, "des"), strings.HasPrefix(mStr, "dec"):
		return time.December
	default:
		return time.January
	}
}

func getCol(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func seedDemoReal(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)
	rng := rand.New(rand.NewSource(options.Seed))

	// 1. Setup Team (Admin)
	adminUser := fake.BuildUser("ADMIN", 1)
	adminUser.Email = "admin@piposmart.id"
	adminUser.Name = "Initial Admin"
	_, err := fake.CreateUser(ctx, tx, adminUser)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	// 2. Read All Authentic Real Owner Records from Excel files
	excelRows := readAllRealOwnerExcelFiles()

	if len(excelRows) == 0 {
		return fmt.Errorf("tidak ada data Excel yang ditemukan di asset/data_admin atau asset/data_sales")
	}

	// Untuk preset real: hanya iterasi data Excel yang ada, tidak boleh duplikat
	targetCount := len(excelRows)

	_ = rng

	timeline := generateGrowthTimeline(options.From, options.To, targetCount, options.Variation, rng)
	progress := newProgressBar(targetCount)

	seenOwners := make(map[string]int64)
	seenOutlets := make(map[string]bool)
	ownerOutletCount := make(map[string]int) // counter outlet per ownerCode

	for idx, defaultCreatedAt := range timeline {
		if idx >= len(excelRows) {
			break // jangan pernah melebihi data real
		}
		ownerIndex := idx + 1
		row := excelRows[idx]

		createdAt := defaultCreatedAt
		if !row.DateOfWork.IsZero() {
			createdAt = clampToAsOf(row.DateOfWork, options.AsOf)
		}

		ownerCode := row.OwnerCode
		if ownerCode == "" || len(ownerCode) > 20 || strings.Contains(ownerCode, " ") {
			if row.OwnerPhone != "" {
				ownerCode = "OWN-" + strings.TrimSpace(row.OwnerPhone)
			} else if row.OwnerName != "" {
				cleanName := strings.ToUpper(strings.ReplaceAll(row.OwnerName, " ", ""))
				if len(cleanName) > 10 {
					cleanName = cleanName[:10]
				}
				ownerCode = "OWN-" + cleanName
			} else {
				ownerCode = fmt.Sprintf("OWN-%05d", ownerIndex)
			}
		}

		var ownerID int64
		if existingID, exists := seenOwners[ownerCode]; exists {
			ownerID = existingID
		} else {
			ownerName := strings.TrimSpace(row.OwnerName)
			if ownerName == "" {
				ownerName = strings.TrimSpace(row.BrandName)
			}
			if ownerName == "" {
				ownerName = "-"
			}

			ownerPhone := row.OwnerPhone
			if len(ownerPhone) > 25 {
				ownerPhone = strings.TrimSpace(ownerPhone[:25])
			}
			if ownerPhone == "" {
				ownerPhone = "-"
			}

			brandName := row.BrandName
			if brandName == "" {
				brandName = ownerName
			}

			city := row.City
			province := row.Province

			owner := factory.Owner{
				Code:      ownerCode,
				Name:      ownerName,
				Phone:     ownerPhone,
				Email:     row.OwnerEmail,
				BrandName: brandName,
				Province:  province,
				City:      city,
				Address:   row.Address,
				CreatedAt: createdAt,
			}

			newOwnerID, err := fake.CreateOwner(ctx, tx, owner)
			if err != nil {
				return fmt.Errorf("create real owner %d: %w", ownerIndex, err)
			}
			ownerID = newOwnerID
			seenOwners[ownerCode] = ownerID
		}

		outletName := row.OutletName
		if outletName == "" {
			outletName = strings.TrimSpace(row.BrandName)
			if outletName == "" {
				outletName = "-"
			}
		}

		outletKey := ownerCode + "|" + strings.ToLower(strings.TrimSpace(outletName))
		if seenOutlets[outletKey] {
			progress.update(ownerIndex)
			continue
		}
		seenOutlets[outletKey] = true



		outletPhone := row.OutletPhone
		if len(outletPhone) > 25 {
			outletPhone = strings.TrimSpace(outletPhone[:25])
		}

		city := row.City
		province := row.Province

		ownerOutletCount[ownerCode]++
		ownerOutletIdx := ownerOutletCount[ownerCode]
		outletCode := fmt.Sprintf("OUT-%05d-%02d", ownerIndex, ownerOutletIdx)
		if len(ownerCode) <= 20 {
			outletCode = fmt.Sprintf("OUT-%s-%02d", ownerCode, ownerOutletIdx)
		}
		outlet := factory.Outlet{
			Code:     outletCode,
			Name:     outletName,
			Phone:    outletPhone,
			Province: province,
			City:     city,
			Address:  row.Address,
		}
		outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
		if err != nil {
			return fmt.Errorf("create real outlet owner=%d: %w", ownerIndex, err)
		}
		// Ensure outlet created_at is historical
		_, _ = tx.ExecContext(ctx, "UPDATE outlets SET created_at = ? WHERE id = ?", createdAt, outletID)

		progress.update(ownerIndex)
	}

	progress.finish()

	return nil
}

func readAllRealOwnerExcelFiles() []realOwnerExcelRow {
	var results []realOwnerExcelRow

	searchDirs := []string{
		filepath.Join("asset", "data_admin"),
		filepath.Join("..", "asset", "data_admin"),
		filepath.Join("backend", "asset", "data_admin"),
	}

	validFiles := []string{
		"01. Owner & Outlet 2021 - 2024 & 2025 - 2026.xlsx",
	}

	var files []string
	for _, dir := range searchDirs {
		for _, v := range validFiles {
			files = append(files, filepath.Join(dir, v))
		}
	}

	for _, fn := range files {
		f, err := excelize.OpenFile(fn)
		if err != nil {
			continue
		}

		for _, sheet := range f.GetSheetList() {
			rows, err := f.GetRows(sheet)
			if err != nil || len(rows) <= 1 {
				continue
			}

			// 1. File "01. Owner & Outlet" — mapping kolom spesifik sesuai header aslinya
			// Row[0] = header: col0=No, col1=Date of Work, col2=Nama Penginput, col3=Kategori Akun,
			// col4=Kode Baris, col5=Kode Owner, col6=Nama Owner, col7=Email Owner,
			// col8=No Hp Owner, col9=No. Hp Outlet, col11=Create Date, col12=Bulan,
			// col13=Nama Project/BRAND, col14=Nama Outlet, col17=Kota, col18=Provinsi, col19=Alamat Lengkap
			if strings.Contains(fn, "01. Owner") {
				// Mapping kolom EKSAK dari header sheet "Owner & Outlet 2025-2026":
				// [0]=No, [1]=Date of Work, [2]=Nama Penginput, [3]=Kategori Akun,
				// [4]=Kode Baris, [5]=Kode Owner, [6]=Nama Owner, [7]=Email Owner,
				// [8]=No Hp Owner, [9]=No. Hp Outlet, [10]=Create Date Project,
				// [11]=Bulan, [12]=Nama Project/BRAND, [13]=Nama Outlet,
				// [14]=Kelurahan, [15]=Kecamatan, [16]=Kota, [17]=Provinsi, [18]=Alamat Lengkap
				const (
					colOwnerCode   = 5
					colOwnerName   = 6
					colOwnerEmail  = 7
					colOwnerPhone  = 8
					colOutletPhone = 9
					colCreateDate  = 10 // "Create Date Project"
					colBrandName   = 12 // "Nama Project/BRAND"
					colOutletName  = 13 // "Nama Outlet"
					colKota        = 16 // "Kota"
					colProvinsi    = 17 // "Provinsi"
					colAlamat      = 18 // "Alamat Lengkap"
				)

				for rIdx := 1; rIdx < len(rows); rIdx++ {
					row := rows[rIdx]
					if len(row) < 14 {
						continue
					}

					// Hapus pengecekan `#ref`, `#value`, `#n/a` sesuai instruksi agar total pas 11.792

					code := getCol(row, colOwnerCode)
					name := getCol(row, colOwnerName)
					brand := getCol(row, colBrandName)
					outlet := getCol(row, colOutletName)

					// Hapus pengecekan if code=="" && name=="" && brand=="" agar masuk semua baris

					ownerPhone := getCol(row, colOwnerPhone)
					outletPhone := getCol(row, colOutletPhone)
					phone := ownerPhone
					if phone == "" {
						phone = outletPhone
					}

					rawDate := getCol(row, colCreateDate)
					var dt time.Time
					for _, layout := range []string{"02/01/2026", "02/01/06", "02/01/2006", "2006-01-02", "01/02/2006"} {
						if t, err := time.Parse(layout, rawDate); err == nil {
							dt = t
							break
						}
					}
					if dt.IsZero() {
						dt = parseDateRobust(rawDate)
					}

					results = append(results, realOwnerExcelRow{
						OwnerCode:   code,
						OwnerName:   name,
						BrandName:   brand,
						OutletName:  outlet,
						OwnerPhone:  ownerPhone,
						OwnerEmail:  getCol(row, colOwnerEmail),
						OutletPhone: outletPhone,
						City:        getCol(row, colKota),
						Province:    getCol(row, colProvinsi),
						Address:     getCol(row, colAlamat),
						DateOfWork:  dt,
					})
				}
				continue
			}

			if len(rows) <= 3 {
				continue
			}

			// 2. Province sheets pattern in 03. Nasabah Baru
			if strings.Contains(fn, "03. Nasabah Baru") && sheet != "TOTAL NASABAH BARU" && sheet != "Kode Wilayah" {
				for rIdx := 4; rIdx < len(rows); rIdx++ {
					row := rows[rIdx]
					if len(row) < 5 {
						continue
					}
					tahun := getCol(row, 1)
					bulanStr := getCol(row, 2)
					code := getCol(row, 3)
					name := getCol(row, 4)
					phone := getCol(row, 5)
					email := getCol(row, 6)
					brand := getCol(row, 7)
					city := getCol(row, 8)
					addr := getCol(row, 9)

					if name == "" && brand == "" {
						continue
					}

					key := code
					if key == "" {
						key = phone
					}
					if key == "" {
						key = name
					}

					var dt time.Time
					if yr, err := strconv.Atoi(tahun); err == nil && yr >= 2020 {
						month := parseMonthIndonesian(bulanStr)
						dt = time.Date(yr, month, 15, 0, 0, 0, 0, time.UTC)
					}

					results = append(results, realOwnerExcelRow{
						OwnerCode:   code,
						OwnerName:   name,
						OwnerPhone:  phone,
						OwnerEmail:  email,
						BrandName:   brand,
						OutletName:  brand,
						City:        city,
						Province:    sheet,
						Address:     addr,
						DateOfWork:  dt,
					})
				}
				continue
			}

			// 3. Standard Header matching for all other sheets
			headerRowIdx := -1
			var colCode, colName, colBrand, colOutlet, colPhone, colEmail, colCity, colProv, colAddr, colDate int = -1, -1, -1, -1, -1, -1, -1, -1, -1, -1

			for rIdx := 0; rIdx < len(rows) && rIdx < 10; rIdx++ {
				row := rows[rIdx]
				for cIdx, cell := range row {
					u := strings.ToUpper(strings.TrimSpace(cell))
					switch {
					case u == "KODE OWNER" || u == "KODE NASABAH" || u == "KODE":
						colCode = cIdx
					case u == "NAMA OWNER" || u == "NAMA NASABAH" || u == "NAMA":
						colName = cIdx
					case strings.Contains(u, "BRAND") || strings.Contains(u, "PROJECT") || u == "NAMA BRAND / OUTLET":
						colBrand = cIdx
					case u == "NAMA OUTLET" || u == "OUTLET":
						colOutlet = cIdx
					case strings.Contains(u, "NO HP") || strings.Contains(u, "NO. HP") || u == "HP" || u == "TELEPON":
						if colPhone == -1 {
							colPhone = cIdx
						}
					case strings.Contains(u, "EMAIL"):
						colEmail = cIdx
					case strings.Contains(u, "KOTA") || strings.Contains(u, "KABUPATEN"):
						colCity = cIdx
					case strings.Contains(u, "PROVINSI"):
						colProv = cIdx
					case strings.Contains(u, "ALAMAT"):
						colAddr = cIdx
					case strings.Contains(u, "DATE") || strings.Contains(u, "TANGGAL") || u == "CREATED":
						if colDate == -1 {
							colDate = cIdx
						}
					}
				}
				if colName != -1 || colBrand != -1 || colCode != -1 {
					headerRowIdx = rIdx
					break
				}
			}

			if headerRowIdx == -1 {
				continue
			}

			for rIdx := headerRowIdx + 1; rIdx < len(rows); rIdx++ {
				row := rows[rIdx]
				if len(row) == 0 {
					continue
				}

				// Skip baris yang ada error formula di kolom mana saja
				hasError := false
				for _, cell := range row {
					if strings.Contains(cell, "#REF!") || strings.Contains(cell, "#VALUE!") || strings.Contains(cell, "#N/A") {
						hasError = true
						break
					}
				}
				if hasError {
					continue
				}

				code := getCol(row, colCode)
				name := getCol(row, colName)
				brand := getCol(row, colBrand)
				outlet := getCol(row, colOutlet)
				phone := getCol(row, colPhone)
				email := getCol(row, colEmail)
				city := getCol(row, colCity)
				prov := getCol(row, colProv)
				addr := getCol(row, colAddr)
				rawDate := getCol(row, colDate)

				if code == "" && name == "" && brand == "" {
					continue
				}
				
				if strings.Contains(strings.ToUpper(getCol(row, 0)), "TOTAL") {
					continue
				}

				key := code
				if key == "" {
					key = phone
				}
				if key == "" {
					key = name
				}

				var dt time.Time
				if rawDate != "" {
					if t, err := time.Parse("2006-01-02", rawDate); err == nil {
						dt = t
					} else if t, err := time.Parse("02/01/06", rawDate); err == nil {
						dt = t
					} else if t, err := time.Parse("02/01/2006", rawDate); err == nil {
						dt = t
					}
				}

				results = append(results, realOwnerExcelRow{
					OwnerCode:   code,
					OwnerName:   name,
					OwnerPhone:  phone,
					OwnerEmail:  email,
					BrandName:   brand,
					OutletName:  outlet,
					City:        city,
					Province:    prov,
					Address:     addr,
					DateOfWork:  dt,
				})
			}
		}
		f.Close()
	}

	return results
}

func seedMultistageLeadAssignments(
	ctx context.Context,
	tx *sql.Tx,
	leadID, ownerID int64,
	salesEmails []string,
	startDate time.Time,
	totalSteps int,
	rng *rand.Rand,
	asOf time.Time,
) error {
	// First deactivate existing assignments for this lead
	if _, err := tx.ExecContext(ctx, `UPDATE lead_assignments SET active = FALSE WHERE lead_id = ?`, leadID); err != nil {
		return err
	}

	// Fetch supervisor ID
	var supervisorID int64
	_ = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE role_code = 'SUPERVISOR' LIMIT 1`).Scan(&supervisorID)

	// Fetch sales user IDs
	salesIDs := make([]int64, 0, len(salesEmails))
	for _, email := range salesEmails {
		var uID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ?`, email).Scan(&uID); err == nil {
			salesIDs = append(salesIDs, uID)
		}
	}
	if len(salesIDs) == 0 {
		return nil
	}

	currentStart := startDate
	actions := []string{
		"TUGAS_1_PEMBAGIAN_SALES",
		"TUGAS_2_FOLLOW_UP_SKORING",
		"TUGAS_3_PRESENTASI_DEMO",
		"TUGAS_4_NEGOSIASI_PROPOSAL",
		"TUGAS_5_CLOSING_HANDOVER",
	}

	scores := []int{20, 45, 65, 85, 100}

	for s := 0; s < totalSteps; s++ {
		isLast := (s == totalSteps-1)
		salesID := salesIDs[(s+rng.Intn(len(salesIDs)))%len(salesIDs)]
		score := scores[minInt(s, len(scores)-1)]
		action := actions[minInt(s, len(actions)-1)]

		var endedAt *time.Time
		if !isLast {
			durationDays := 2 + rng.Intn(6)
			endT := clampToAsOf(currentStart.AddDate(0, 0, durationDays), asOf)
			endedAt = &endT
		}

		active := isLast

		res, err := tx.ExecContext(ctx, `
			INSERT INTO lead_assignments
				(lead_id, owner_id, from_user_id, from_role, to_user_id, to_role, supervisor_id, assigned_by_user_id, action, reason, score, active, started_at, ended_at, created_at)
			VALUES (?, ?, ?, 'SUPERVISOR', ?, 'SALES', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			leadID, ownerID, supervisorID, salesID, supervisorID, supervisorID, action,
			fmt.Sprintf("Distribusi tugas tahapan %d oleh Supervisor", s+1),
			score, active, currentStart, endedAt, currentStart,
		)
		if err != nil {
			return err
		}

		if active {
			// Update customer_leads with current score and active sales user
			_, _ = tx.ExecContext(ctx, `
				UPDATE customer_leads
				SET current_score = ?, active_sales_id = ?, current_owner_user_id = ?
				WHERE id = ?`,
				score, salesID, salesID, leadID,
			)
		}

		if endedAt != nil {
			currentStart = *endedAt
		}
		_ = res
	}

	return nil
}
