package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type mitraRow struct {
	Code        string
	Name        string
	Category    string
	CreatedAt   string
	Province    string
	City        string
	District    string
	Address     string
	Phone       string
	Email       string
	PIC         string
}

func readMitraExcel() ([]mitraRow, error) {
	var results []mitraRow
	fn := filepath.Join("asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx")
	
	f, err := excelize.OpenFile(fn)
	if err != nil {
		fn = filepath.Join("..", "asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx")
		f, err = excelize.OpenFile(fn)
		if err != nil {
			return nil, nil // Skip if not found
		}
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
	userRows, err := tx.QueryContext(ctx, "SELECT id, name, email FROM users WHERE role = 'SALES' OR role = 'ADMIN'")
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
				district=VALUES(district)
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
			_, err = tx.ExecContext(ctx, `
				INSERT INTO partner_assignments 
					(partner_id, user_id, assigned_by_id, assigned_at, active, created_at)
				VALUES (?, ?, ?, ?, true, ?)
			`, partnerID, picID, adminID, createdAt, createdAt)
			if err != nil {
				// Assignment might already exist, ignore error for simplicity or handle gracefully
			}
		}
	}

	return nil
}
