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

	"backend_crm_piposmart/internal/platform/factory"
)

type subscribeRow struct {
	OwnerCode      string
	DateTopUp      string
	DateStruk      string
	DateAktivasi   string
	BalanceTf      string
	BalanceSystem  string
	MetodeBayar    string
	Paket          string
	Tenor          string
}

func readNewAndSubscribeExcel() ([]subscribeRow, error) {
	var results []subscribeRow
	fn := filepath.Join("asset", "data_admin", "02. New & Subscribe 2026 (Copy).xlsx")
	
	f, err := excelize.OpenFile(fn)
	if err != nil {
		fn = filepath.Join("..", "asset", "data_admin", "02. New & Subscribe 2026 (Copy).xlsx")
		f, err = excelize.OpenFile(fn)
		if err != nil {
			return nil, nil // Skip if not found
		}
	}
	defer f.Close()

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) <= 3 {
			continue
		}

		// Find header
		headerRowIdx := -1
		colOwnerCode, colTopUpSystem, colTopUpStruk, colAktivasi, colBalanceSystem, colBalanceTf, colMetode, colTenor, colPaket := -1, -1, -1, -1, -1, -1, -1, -1, -1

		for rIdx := 0; rIdx < len(rows) && rIdx < 10; rIdx++ {
			row := rows[rIdx]
			for cIdx, cell := range row {
				u := strings.ToUpper(strings.TrimSpace(cell))
				switch u {
				case "KODE OWNER":
					colOwnerCode = cIdx
				case "DATE TOP UP  SYSTEM":
					colTopUpSystem = cIdx
				case "DATE TOP UP STRUK":
					colTopUpStruk = cIdx
				case "TANGGAL AKTIVASI":
					colAktivasi = cIdx
				case "BALANCE TOP UP SYSTEM":
					colBalanceSystem = cIdx
				case "BALANCE BUKTI TF":
					colBalanceTf = cIdx
				case "METODE PEMBAYARAN":
					colMetode = cIdx
				case "TENOR":
					colTenor = cIdx
				case "PAKET MEMBERSHIP":
					colPaket = cIdx
				}
			}
			if colOwnerCode != -1 && colPaket != -1 {
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
			
			code := getCol(row, colOwnerCode)
			if code == "" {
				continue
			}

			results = append(results, subscribeRow{
				OwnerCode:     code,
				DateTopUp:     getCol(row, colTopUpSystem),
				DateStruk:     getCol(row, colTopUpStruk),
				DateAktivasi:  getCol(row, colAktivasi),
				BalanceTf:     getCol(row, colBalanceTf),
				BalanceSystem: getCol(row, colBalanceSystem),
				MetodeBayar:   getCol(row, colMetode),
				Paket:         getCol(row, colPaket),
				Tenor:         getCol(row, colTenor),
			})
		}
	}
	return results, nil
}

func SeedSubscriptionsFromExcel(ctx context.Context, tx *sql.Tx, fake *factory.Factory, adminID int64) error {
	rows, err := readNewAndSubscribeExcel()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	for i, row := range rows {
		// 1. Find Owner (might have -index suffix from deduplication bypass)
		var ownerID int64
		err := tx.QueryRowContext(ctx, "SELECT id FROM owners WHERE code = ? LIMIT 1", row.OwnerCode).Scan(&ownerID)
		if err == sql.ErrNoRows {
			// Skip if owner not found
			continue
		} else if err != nil {
			return err
		}

		// 2. Parse Dates
		paidAt := parseDateRobust(row.DateTopUp)
		transferDate := parseDateRobust(row.DateStruk)
		aktivasiDate := parseDateRobust(row.DateAktivasi)
		if aktivasiDate.IsZero() {
			if !paidAt.IsZero() {
				aktivasiDate = paidAt
			} else {
				aktivasiDate = time.Now()
			}
		}
		if paidAt.IsZero() {
			paidAt = aktivasiDate
		}

		// 3. Create TopUp
		balanceStr := row.BalanceSystem
		if balanceStr == "" || balanceStr == "0" {
			balanceStr = row.BalanceTf
		}
		if balanceStr != "" && balanceStr != "0" {
			// Strip non-numeric
			balanceStr = strings.ReplaceAll(balanceStr, ",", "")
			balanceStr = strings.ReplaceAll(balanceStr, ".", "")
			
			topup := factory.WalletTopup{
				Amount:            balanceStr,
				PaymentChannel:    row.MetodeBayar,
				ExternalReference: fmt.Sprintf("EXCEL-TF-%s-%d", row.OwnerCode, i),
				IdempotencyKey:    fmt.Sprintf("excel-tf-%s-%d", row.OwnerCode, i),
				PaidAt:            paidAt,
				Note:              "Import dari Excel New & Subscribe",
				Status:            "ACCEPTED",
			}
			if !transferDate.IsZero() {
				topup.TransferDateAt = transferDate
			}
			_, err = fake.CreateWalletTopup(ctx, tx, ownerID, topup)
			if err != nil {
				return fmt.Errorf("create topup owner=%s: %w", row.OwnerCode, err)
			}
		}

		// 4. Create Subscription
		if row.Paket != "" {
			tenor, _ := strconv.Atoi(row.Tenor)
			if tenor == 0 {
				tenor = 1
			}
			
			paketStr := strings.ToUpper(strings.TrimSpace(row.Paket))
			if paketStr == "BISNIS" {
				paketStr = "BUSINESS"
			}
			planCode := fmt.Sprintf("%s_%02d_MONTHS", paketStr, tenor)
			
			order := factory.SubscriptionOrder{
				PlanCode:              planCode,
				ExternalReference:     fmt.Sprintf("EXCEL-SUB-%s-%d", row.OwnerCode, i),
				IdempotencyKey:        fmt.Sprintf("excel-sub-%s-%d", row.OwnerCode, i),
				PurchasedAt:           paidAt,
				SubscriptionStartDate: aktivasiDate,
				Note:                  "Import dari Excel New & Subscribe",
				AllowBalanceShortfall: true, // Biarkan minus kalau balance kurang
			}
			_, err = fake.CreateSubscriptionOrder(ctx, tx, ownerID, order)
			if err != nil {
				// if plan not found, ignore and continue
				if strings.Contains(err.Error(), "tidak ditemukan") || strings.Contains(err.Error(), "no rows in result set") {
					continue
				}
				return fmt.Errorf("create subscription owner=%s: %w", row.OwnerCode, err)
			}
		}
	}
	return nil
}

func parseDateRobust(d string) time.Time {
	d = strings.TrimSpace(d)
	if d == "" || strings.EqualFold(d, "No Date") {
		return time.Time{}
	}
	// format dd/mm/yy
	if t, err := time.Parse("02/01/06", d); err == nil {
		return t
	}
	if t, err := time.Parse("02/01/2006", d); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", d); err == nil {
		return t
	}
	return time.Time{}
}


