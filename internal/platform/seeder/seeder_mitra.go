package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type mitraRow struct {
	Code      string
	Name      string
	Category  string
	CreatedAt string
	Province  string
	City      string
	District  string
	Address   string
	Phone     string
	Email     string
	PIC       string
}

func readMitraExcel() ([]mitraRow, error) {
	var results []mitraRow
	searchPaths := []string{
		filepath.Join("asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
		filepath.Join("..", "asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
		filepath.Join("backend", "asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
	}

	var (
		f   *excelize.File
		err error
	)
	for _, fn := range searchPaths {
		f, err = excelize.OpenFile(fn)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, nil // Skip if not found
	}
	defer f.Close()

	rows, err := f.GetRows("Daftar Mitra")
	if err != nil || len(rows) <= 2 {
		return nil, nil
	}

	headerRowIdx := -1
	colCode, colName, colCategory, colDate, colProvince, colCity, colDistrict, colAddress, colPhone, colEmail, colPIC := -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1

	for rIdx := 0; rIdx < len(rows) && rIdx < 5; rIdx++ {
		row := rows[rIdx]
		for cIdx, cell := range row {
			u := strings.ToUpper(strings.TrimSpace(cell))
			switch u {
			case "KODE OWNER":
				colCode = cIdx
			case "NAMA BRAND / OUTLET":
				colName = cIdx
			case "KATEGORI MITRA":
				colCategory = cIdx
			case "TANGGAL KERJASAMA":
				colDate = cIdx
			case "PROVINSI":
				colProvince = cIdx
			case "KABUPATEN/KOTA":
				colCity = cIdx
			case "KECAMATAN":
				colDistrict = cIdx
			case "ALAMAT":
				colAddress = cIdx
			case "NO HP":
				colPhone = cIdx
			case "EMAIL":
				colEmail = cIdx
			case "PIC":
				colPIC = cIdx
			}
		}
		if colCode != -1 && colPIC != -1 {
			headerRowIdx = rIdx
			break
		}
	}
	if headerRowIdx == -1 {
		return nil, fmt.Errorf("mitra headers not found")
	}

	for rIdx := headerRowIdx + 1; rIdx < len(rows); rIdx++ {
		row := rows[rIdx]
		if len(row) == 0 {
			continue
		}

		code := getCol(row, colCode)
		if code == "" {
			continue
		}

		name := getCol(row, colName)
		if name == "" {
			name = "Mitra " + code
		}

		results = append(results, mitraRow{
			Code:      code,
			Name:      name,
			Category:  getCol(row, colCategory),
			CreatedAt: getCol(row, colDate),
			Province:  getCol(row, colProvince),
			City:      getCol(row, colCity),
			District:  getCol(row, colDistrict),
			Address:   getCol(row, colAddress),
			Phone:     getCol(row, colPhone),
			Email:     getCol(row, colEmail),
			PIC:       getCol(row, colPIC),
		})
	}
	return results, nil
}

func SeedMitraFromExcel(ctx context.Context, tx *sql.Tx, adminID int64, salesEmails []string) error {
	rows, err := readMitraExcel()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	// Build users map for fuzzy PIC search
	users := make(map[string]int64)
	userRows, err := tx.QueryContext(ctx, `
		SELECT u.id, u.name, u.email
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE r.code IN ('SALES', 'SUPERVISOR', 'ADMIN')`)
	if err == nil {
		for userRows.Next() {
			var id int64
			var name, email string
			if err := userRows.Scan(&id, &name, &email); err == nil {
				users[strings.ToLower(name)] = id
				users[strings.ToLower(email)] = id
			}
		}
		userRows.Close()
	}

	// Fetch partner types
	partnerTypes := make(map[string]int64)
	ptRows, err := tx.QueryContext(ctx, "SELECT id, code FROM partner_types")
	if err == nil {
		for ptRows.Next() {
			var id int64
			var code string
			if err := ptRows.Scan(&id, &code); err == nil {
				partnerTypes[code] = id
			}
		}
		ptRows.Close()
	}

	defaultPtID := int64(1)
	if len(partnerTypes) > 0 {
		for _, id := range partnerTypes {
			defaultPtID = id
			break
		}
	}
	if refID, ok := partnerTypes["REFERRAL"]; ok {
		defaultPtID = refID
	}

	for i, row := range rows {
		createdAt := parseDateRobust(row.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		// Find partner type ID
		ptID := defaultPtID
		catUpper := strings.ToUpper(row.Category)
		if strings.Contains(catUpper, "REFERRAL") || strings.Contains(catUpper, "REFERAL") {
			if id, ok := partnerTypes["REFERRAL"]; ok {
				ptID = id
			}
		} else if strings.Contains(catUpper, "AFILIASI") {
			if id, ok := partnerTypes["STRATEGIC"]; ok {
				ptID = id
			}
		} else if strings.Contains(catUpper, "FRANCHISE") {
			if id, ok := partnerTypes["PARTNERSHIP"]; ok {
				ptID = id
			}
		}

		// Insert into partners
		res, err := tx.ExecContext(ctx, `
			INSERT INTO partners 
				(partner_type_id, code, name, phone, email, address, province, city, district, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?)
			ON DUPLICATE KEY UPDATE 
				partner_type_id=VALUES(partner_type_id),
				name=VALUES(name),
				phone=VALUES(phone),
				email=VALUES(email),
				address=VALUES(address),
				province=VALUES(province),
				city=VALUES(city),
				district=VALUES(district),
				status='ACTIVE'
		`, ptID, row.Code, row.Name, row.Phone, row.Email, row.Address, row.Province, row.City, row.District, createdAt)
		if err != nil {
			return fmt.Errorf("insert partner %d (%s): %w", i+1, row.Code, err)
		}

		partnerID, err := res.LastInsertId()
		if err != nil || partnerID == 0 {
			err = tx.QueryRowContext(ctx, "SELECT id FROM partners WHERE code = ?", row.Code).Scan(&partnerID)
			if err != nil {
				continue
			}
		}

		// Assign PIC
		picName := strings.ToLower(strings.TrimSpace(row.PIC))
		var picID int64
		if picName != "" {
			// Find PIC by exact name or email
			if id, ok := users[picName]; ok {
				picID = id
			} else {
				// Fuzzy search
				for uname, uid := range users {
					if strings.Contains(uname, picName) || strings.Contains(picName, uname) {
						picID = uid
						break
					}
				}
			}
		}

		if picID != 0 {
			if err := syncSeedPartnerAssignment(ctx, tx, partnerID, picID, adminID, createdAt); err != nil {
				return fmt.Errorf("assign partner %d (%s): %w", partnerID, row.Code, err)
			}
		}
	}

	return nil
}

func syncSeedPartnerAssignment(ctx context.Context, tx *sql.Tx, partnerID, picID, adminID int64, assignedAt time.Time) error {
	var currentAssignmentID int64
	var currentUserID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id
		FROM partner_assignments
		WHERE partner_id = ? AND active = 1
		ORDER BY id DESC
		LIMIT 1`, partnerID).Scan(&currentAssignmentID, &currentUserID)
	switch {
	case err == nil:
		if currentUserID == picID {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE partner_assignments
			SET active = FALSE, unassigned_at = ?, updated_at = NOW()
			WHERE id = ?`, assignedAt, currentAssignmentID); err != nil {
			return err
		}
	case err != sql.ErrNoRows:
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO partner_assignments
			(partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, TRUE, ?, ?)`,
		partnerID, picID, adminID, assignedAt, assignedAt, assignedAt,
	)
	return err
}

func SeedKomisiMitraFromExcel(ctx context.Context, tx *sql.Tx, adminID int64) error {
	searchPaths := []string{
		filepath.Join("asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
		filepath.Join("..", "asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
		filepath.Join("backend", "asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
	}

	var f *excelize.File
	var err error
	for _, fn := range searchPaths {
		f, err = excelize.OpenFile(fn)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil // skip if not found
	}
	defer f.Close()

	rows, err := f.GetRows("Mitra - Referral")
	if err != nil || len(rows) <= 2 {
		return nil
	}

	for rIdx := 2; rIdx < len(rows); rIdx++ {
		row := rows[rIdx]
		if len(row) < 70 {
			padded := make([]string, 70)
			copy(padded, row)
			row = padded
		}

		partnerCode := strings.TrimSpace(getCol(row, 2))
		if partnerCode == "" {
			continue
		}
		leadCode := strings.TrimSpace(getCol(row, 5))
		if leadCode == "" {
			continue
		}
		leadName := strings.TrimSpace(getCol(row, 7))
		if leadName == "" {
			leadName = "Lead " + leadCode
		}

		totalStr := strings.ReplaceAll(getCol(row, 69), ",", "")
		totalStr = strings.ReplaceAll(totalStr, ".", "")
		totalAmount, _ := strconv.ParseFloat(totalStr, 64)
		if totalAmount <= 0 {
			continue
		}

		var partnerID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM partners WHERE code = ?", partnerCode).Scan(&partnerID)
		if err != nil {
			continue
		}

		var ownerID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM owners WHERE code = ?", leadCode).Scan(&ownerID)
		if err != nil {
			// Jika owner tidak ada, buat dummy owner agar bisa lanjut
			res, err := tx.ExecContext(ctx, `
				INSERT INTO owners (code, name, phone, status, created_at, updated_at)
				VALUES (?, ?, ?, 'ACTIVE', NOW(), NOW())
				ON DUPLICATE KEY UPDATE name=VALUES(name)
			`, leadCode, leadName, "0000000000")
			if err != nil {
				return err
			}
			ownerID, _ = res.LastInsertId()
			if ownerID == 0 {
				_ = tx.QueryRowContext(ctx, "SELECT id FROM owners WHERE code = ?", leadCode).Scan(&ownerID)
			}
		}

		var leadID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM customer_leads WHERE owner_id = ?", ownerID).Scan(&leadID)
		if err != nil {
			res, err := tx.ExecContext(ctx, `
				INSERT INTO customer_leads (code, owner_id, source_type, stage, status, created_at, updated_at)
				VALUES (?, ?, 'REFERRAL', 'NEW', 'OPEN', NOW(), NOW())`,
				leadCode+"_LEAD", ownerID)
			if err != nil {
				return err
			}
			leadID, _ = res.LastInsertId()
		}

		var referralID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM partner_referrals WHERE partner_id = ? AND lead_id = ?", partnerID, leadID).Scan(&referralID)
		if err != nil {
			res, err := tx.ExecContext(ctx, `
				INSERT INTO partner_referrals (partner_id, lead_id, referral_date, notes, created_at)
				VALUES (?, ?, NOW(), 'Imported from Excel', NOW())`,
				partnerID, leadID)
			if err != nil {
				return err
			}
			referralID, _ = res.LastInsertId()
		}

		var closingID int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM sales_closings WHERE lead_id = ? LIMIT 1", leadID).Scan(&closingID)
		if err != nil {
			closingCode := fmt.Sprintf("CLS-%d-%d", partnerID, leadID)
			res, err := tx.ExecContext(ctx, `
				INSERT INTO sales_closings (
					code, lead_id, package_snapshot_json, plan_snapshot_json, 
					tenure_months, duration_days, base_price, final_amount, 
					currency, closed_at, confirmed_at, status, created_at, updated_at
				) VALUES (?, ?, '{}', '{}', 1, 30, 0, ?, 'IDR', NOW(), NOW(), 'CONFIRMED', NOW(), NOW())`,
				closingCode, leadID, 0)
			if err != nil {
				return err
			}
			closingID, _ = res.LastInsertId()
		}

		commissionMode := "FIXED"
		commissionValue := totalAmount
		statusVal := "PENDING"
		statusCol := strings.ToUpper(strings.TrimSpace(getCol(row, 68)))
		if statusCol == "FINISH" || statusCol == "PAY" {
			statusVal = "PAID"
		}

		code := fmt.Sprintf("COM-%s-%s", partnerCode, leadCode)
		res, err := tx.ExecContext(ctx, `
			INSERT INTO partner_commissions (
				code, partner_id, referral_id, closing_id, commission_mode, commission_value,
				base_amount, commission_amount, currency, status, note, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'IDR', ?, 'Seeded from Excel', NOW(), NOW())
			ON DUPLICATE KEY UPDATE commission_amount=VALUES(commission_amount), status=VALUES(status)
		`, code, partnerID, referralID, closingID, commissionMode, commissionValue,
			0, totalAmount, statusVal)
		
		if err != nil {
			return err
		}

		commissionID, _ := res.LastInsertId()
		if commissionID == 0 {
			_ = tx.QueryRowContext(ctx, "SELECT id FROM partner_commissions WHERE code = ?", code).Scan(&commissionID)
		}

		if statusVal == "PAID" && commissionID > 0 {
			payoutCode := fmt.Sprintf("PAYOUT-%s-%s", partnerCode, leadCode)
			res, err = tx.ExecContext(ctx, `
				INSERT INTO partner_payouts (
					code, partner_id, total_amount, currency, status, note, prepared_by_user_id, paid_by_user_id, paid_at, created_at, updated_at
				) VALUES (?, ?, ?, 'IDR', 'PAID', 'Seeded from Excel', ?, ?, NOW(), NOW(), NOW())
				ON DUPLICATE KEY UPDATE status='PAID'
			`, payoutCode, partnerID, totalAmount, adminID, adminID)
			if err != nil {
				return err
			}
			payoutID, _ := res.LastInsertId()
			if payoutID == 0 {
				_ = tx.QueryRowContext(ctx, "SELECT id FROM partner_payouts WHERE code = ?", payoutCode).Scan(&payoutID)
			}
			if payoutID > 0 {
				tx.ExecContext(ctx, `
					INSERT IGNORE INTO partner_payout_items (payout_id, commission_id, amount, released_at, created_at, updated_at)
					VALUES (?, ?, ?, NOW(), NOW(), NOW())
				`, payoutID, commissionID, totalAmount)
			}
		}
	}

	return nil
}
