package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"database/sql"
	"backend_crm_piposmart/internal/platform/config"
	"github.com/xuri/excelize/v2"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name))
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Load existing owner phones
	rows, err := db.QueryContext(ctx, "SELECT phone FROM owners")
	if err != nil {
		log.Fatalf("query owners: %v", err)
	}
	defer rows.Close()

	ownerPhones := make(map[string]bool)
	for rows.Next() {
		var phone string
		if err := rows.Scan(&phone); err == nil {
			phone = cleanPhone(phone)
			if phone != "" {
				ownerPhones[phone] = true
			}
		}
	}

	fn := filepath.Join("asset", "data_admin", "04. Data Belum Registrasi 2026 - User Temp (Copy).xlsx")
	
	f, err := excelize.OpenFile(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var duplicateCount, newCount int

	for _, sheet := range f.GetSheetList() {
		excelRows, err := f.GetRows(sheet)
		if err != nil || len(excelRows) < 2 {
			continue
		}

		// Find header
		headerRowIdx := -1
		colPhone := -1

		for rIdx := 0; rIdx < len(excelRows) && rIdx < 5; rIdx++ {
			row := excelRows[rIdx]
			for cIdx, cell := range row {
				u := strings.ToUpper(strings.TrimSpace(cell))
				if u == "NOMOR TELEPON" || u == "NO. HP" || u == "PHONE" {
					colPhone = cIdx
				}
			}
			if colPhone != -1 {
				headerRowIdx = rIdx
				break
			}
		}

		if headerRowIdx == -1 {
			continue
		}

		fmt.Println("| Nomor HP / Telepon | PIC Follow Up | Kode OTP | Status |")
		fmt.Println("|---|---|---|---|")

		for rIdx := headerRowIdx + 1; rIdx < len(excelRows); rIdx++ {
			row := excelRows[rIdx]
			if len(row) == 0 {
				continue
			}

			phoneRaw := ""
			if colPhone < len(row) {
				phoneRaw = row[colPhone]
			}
			phoneClean := cleanPhone(phoneRaw)

			if phoneClean == "" {
				continue
			}

			if ownerPhones[phoneClean] {
				duplicateCount++
				fmt.Printf("| %s | %s | %s | %s |\n", phoneRaw, getCol(row, 2), getCol(row, 3), "Sudah Ada (Duplikat)")
			} else {
				newCount++
				fmt.Printf("| %s | %s | %s | %s |\n", phoneRaw, getCol(row, 2), getCol(row, 3), "Belum Ada (Baru)")
			}
		}
	}

	fmt.Printf("\n**Ringkasan:**\n- Duplikat (sudah ada di Owner): %d\n- Data Baru (belum ada): %d\n", duplicateCount, newCount)
}

func getCol(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func cleanPhone(p string) string {
	p = strings.TrimSpace(p)
	// simple cleanup: keep only digits
	var res string
	for _, c := range p {
		if c >= '0' && c <= '9' {
			res += string(c)
		}
	}
	// normalize 62 to 0
	if strings.HasPrefix(res, "62") {
		res = "0" + res[2:]
	}
	return res
}
