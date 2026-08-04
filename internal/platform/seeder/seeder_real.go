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

	// 1. Setup Team (Admin, Supervisors and Sales)
	adminUser := fake.BuildUser("ADMIN", 1)
	adminUser.Email = "admin@piposmart.id"
	adminUser.Name = "Initial Admin"
	adminID, err := fake.CreateUser(ctx, tx, adminUser)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	supervisorCount := 3
	salesCount := 10

	for i := 1; i <= supervisorCount; i++ {
		user := fake.BuildUser("SUPERVISOR", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return fmt.Errorf("create supervisor %d: %w", i, err)
		}
	}

	salesEmails := make([]string, 0, salesCount)
	for i := 1; i <= salesCount; i++ {
		user := fake.BuildUser("SALES", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return fmt.Errorf("create sales %d: %w", i, err)
		}
		salesEmails = append(salesEmails, user.Email)
	}

	// 2. Read All Authentic Real Owner Records from Excel files
	excelRows := readAllRealOwnerExcelFiles()

	if len(excelRows) == 0 {
		return fmt.Errorf("tidak ada data Excel yang ditemukan di asset/data_admin atau asset/data_sales")
	}

	targetCount := len(excelRows)
	if options.Scale > 0 {
		scaleCount, err := largeSeedOwnerCountForScale(options.Scale)
		if err == nil && scaleCount > 0 {
			targetCount = scaleCount
		}
	}

	_ = rng
	_ = targetCount

	timeline := generateGrowthTimeline(options.From, options.To, targetCount, options.Variation, rng)
	progress := newProgressBar(targetCount)

	seenOwners := make(map[string]int64)
	seenOutlets := make(map[string]bool)
	var outletCount int

	for idx, defaultCreatedAt := range timeline {
		ownerIndex := idx + 1
		row := excelRows[idx%len(excelRows)]

		createdAt := defaultCreatedAt
		if !row.DateOfWork.IsZero() {
			createdAt = clampToAsOf(row.DateOfWork, options.AsOf)
		}

		ownerCode := row.OwnerCode
		if ownerCode == "" || len(ownerCode) > 20 || strings.Contains(ownerCode, " ") {
			ownerCode = fmt.Sprintf("OWN-%05d", ownerIndex)
		}

		var ownerID int64
		isNewOwner := false
		if existingID, exists := seenOwners[ownerCode]; exists {
			ownerID = existingID
		} else {
			isNewOwner = true
			ownerName := strings.TrimSpace(row.OwnerName)
			if ownerName == "" {
				ownerName = strings.TrimSpace(row.BrandName)
			}
			if ownerName == "" {
				ownerName = fmt.Sprintf("Owner %s", ownerCode)
			}

			ownerPhone := row.OwnerPhone
			if len(ownerPhone) > 25 {
				ownerPhone = strings.TrimSpace(ownerPhone[:25])
			}
			if ownerPhone == "" {
				ownerPhone = fmt.Sprintf("0812345%05d", ownerIndex)
			}

			brandName := row.BrandName
			if brandName == "" {
				brandName = ownerName + " Brand"
			}

			city := row.City
			if city == "" {
				city = "Jakarta Selatan"
			}

			province := row.Province
			if province == "" {
				province = "DKI Jakarta"
			}

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
				outletName = fmt.Sprintf("Outlet %d", outletCount+1)
			}
		}

		outletKey := ownerCode + "|" + strings.ToLower(strings.TrimSpace(outletName))
		if seenOutlets[outletKey] {
			progress.update(ownerIndex)
			continue
		}
		seenOutlets[outletKey] = true

		outletCount++

		outletPhone := row.OutletPhone
		if len(outletPhone) > 25 {
			outletPhone = strings.TrimSpace(outletPhone[:25])
		}

		city := row.City
		if city == "" {
			city = "Jakarta Selatan"
		}

		province := row.Province
		if province == "" {
			province = "DKI Jakarta"
		}

		outletCode := fmt.Sprintf("OUT-%05d-%02d", ownerIndex, outletCount)
		if len(ownerCode) <= 20 {
			outletCode = fmt.Sprintf("OUT-%s-%02d", ownerCode, outletCount)
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

		if isNewOwner {
			// 3. Lead Creation & Multi-Stage Assignment (Tugas 1 s.d. 5)
			leadCode := fmt.Sprintf("LEAD-%05d", ownerIndex)
			if len(ownerCode) <= 25 {
				leadCode = fmt.Sprintf("LEAD-%s-%02d", ownerCode, outletCount)
			}
			assignedSalesEmail := salesEmails[rng.Intn(len(salesEmails))]

			leadCreatedAt := clampToAsOf(createdAt, options.AsOf)
			lead := factory.Lead{
				Code:             leadCode,
				SourceType:       "EXCEL_IMPORT",
				SourceReference:  "Data Admin & Sales Excel",
				Stage:            "NEW_LEAD",
				Status:           "IN_PROGRESS",
				NextFollowUpAt:   clampToAsOf(leadCreatedAt.AddDate(0, 0, 3+rng.Intn(5)), options.AsOf),
				ActiveSalesEmail: assignedSalesEmail,
			}

			leadID, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead)
			if err != nil {
				return fmt.Errorf("create real lead owner=%d: %w", ownerIndex, err)
			}
			// Update customer_leads created_at to match historical date
			_, _ = tx.ExecContext(ctx, "UPDATE customer_leads SET created_at = ? WHERE id = ?", leadCreatedAt, leadID)

			stepCount := 2 + rng.Intn(4) // 2 to 5 steps
			if err := seedMultistageLeadAssignments(ctx, tx, leadID, ownerID, salesEmails, leadCreatedAt, stepCount, rng, options.AsOf); err != nil {
				return fmt.Errorf("seed multistage lead assignments lead=%d: %w", leadID, err)
			}
		}

		progress.update(ownerIndex)
	}

	progress.finish()

	// 4. Seed Subscriptions from New & Subscribe Excel
	fmt.Println("Seeding subscriptions dari Excel...")
	if err := SeedSubscriptionsFromExcel(ctx, tx, fake, adminID); err != nil {
		return fmt.Errorf("seed subscriptions: %w", err)
	}

	// 5. Seed Mitra
	fmt.Println("Seeding mitra dari Excel...")
	if err := SeedMitraFromExcel(ctx, tx, adminID, salesEmails); err != nil {
		return fmt.Errorf("seed mitra: %w", err)
	}

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
		"01. Owner & Outlet 2026 (Copy).xlsx",
		"03. Nasabah Baru Per Provinsi 2026 (Copy).xlsx",
		"04. Data Belum Registrasi 2026 - User Temp (Copy).xlsx",
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
			if err != nil || len(rows) <= 3 {
				continue
			}

			// 1. Province sheets pattern in 03. Nasabah Baru
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

			// 2. Standard Header matching for all other sheets
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
				
				// Ignore rows with Excel formula errors
				if strings.Contains(code, "#REF!") || strings.Contains(name, "#REF!") || strings.Contains(brand, "#REF!") || strings.Contains(strings.ToUpper(row[0]), "TOTAL") {
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
