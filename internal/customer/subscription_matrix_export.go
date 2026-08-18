package customer

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type SubscriptionMatrixYearData struct {
	Year        int
	MonthlyVals [12]string
	MonthlyCode [12]string
}

type SubscriptionMatrixRow struct {
	OwnerCode     string
	OwnerName     string
	OwnerPhone    string
	BrandOutlet   string
	PICInfo       string
	City          string
	Province      string
	Region        string
	CreatedAtYear int
	YearsData     []SubscriptionMatrixYearData
}
// matrixRowHasActivity returns true jika minimal satu sel bulanan terisi di seluruh tahun.
func matrixRowHasActivity(row SubscriptionMatrixRow) bool {
	for _, yd := range row.YearsData {
		for _, v := range yd.MonthlyVals {
			if v != "" {
				return true
			}
		}
	}
	return false
}
func formatRegion(region string) string {
	r := strings.ToLower(strings.TrimSpace(region))
	switch r {
	case "jawa timur":
		return "JATIM"
	case "jawa barat":
		return "JABAR"
	case "jawa tengah":
		return "JATENG"
	case "kepulauan riau":
		return "KEPRI"
	case "daerah istimewa yogyakarta", "di yogyakarta":
		return "DIY"
	case "dki jakarta":
		return "DKI JAKARTA"
	case "sumatera utara":
		return "SUMUT"
	case "sumatera barat":
		return "SUMBAR"
	case "sumatera selatan":
		return "SUMSEL"
	case "kalimantan timur":
		return "KALTIM"
	case "kalimantan barat":
		return "KALBAR"
	case "kalimantan selatan":
		return "KALSEL"
	case "kalimantan tengah":
		return "KALTENG"
	case "kalimantan utara":
		return "KALTARA"
	case "sulawesi utara":
		return "SULUT"
	case "sulawesi selatan":
		return "SULSEL"
	case "sulawesi tengah":
		return "SULTENG"
	case "sulawesi barat":
		return "SULBAR"
	case "sulawesi tenggara":
		return "SULTRA"
	case "nusa tenggara timur":
		return "NTT"
	case "nusa tenggara barat":
		return "NTB"
	case "bangka belitung", "kepulauan bangka belitung":
		return "BABEL"

	default:
		if r == "" {
			return ""
		}
		return strings.ToUpper(region)
	}
}

func packageAbbrev(sub *SubscriptionDetail) string {
	name := strings.ToLower(sub.PackageName.String + " " + sub.PackageCode.String)
	switch {
	case strings.Contains(name, "pro"):
		return "PR"
	case strings.Contains(name, "bisnis") || strings.Contains(name, "business"):
		return "BS"
	case strings.Contains(name, "basic"):
		return "BC"
	default:
		code := strings.ToUpper(strings.TrimSpace(sub.PackageCode.String))
		if len(code) >= 2 {
			return code[:2]
		}
		return "??"
	}
}

func tenorSuffixNew(months int64) string {
	switch months {
	case 3:
		return "-3"
	case 6:
		return "-6"
	case 9:
		return "-9"
	case 12:
		return "-F"
	case 18:
		return "-18"
	case 24:
		return "-24"
	default:
		return ""
	}
}

func tenorSuffixFollowing(months int64) string {
	switch months {
	case 3:
		return "(3)"
	case 6:
		return "(6)"
	case 9:
		return "(9)"
	case 18:
		return "(18)"
	case 24:
		return "(24)"
	default:
		return ""
	}
}

// findLastPriorSub returns subscription terakhir yang berakhir (active_until)
// sebelum mStart, atau nil jika tidak ada.
// Digunakan untuk mengisi kolom bulan GAP dengan kode "U-XX" (misal U-BS, U-PR, U-BC)
// berdasarkan paket terakhir yang dimiliki outlet sebelum gap tersebut.
func findLastPriorSub(sorted []SubscriptionDetail, mStart time.Time) *SubscriptionDetail {
	var last *SubscriptionDetail
	for i := range sorted {
		sub := &sorted[i]
		if !sub.ActiveUntil.Valid {
			continue
		}
		// Langganan ini sudah berakhir sebelum bulan target dimulai
		if sub.ActiveUntil.Time.Before(mStart) {
			if last == nil || sub.ActiveUntil.Time.After(last.ActiveUntil.Time) ||
				(sub.ActiveUntil.Time.Equal(last.ActiveUntil.Time) && sub.ID > last.ID) {
				last = sub
			}
		}
	}
	return last
}

// buildMatrixRowForYears menghitung sel bulanan untuk satu outlet melintasi beberapa tahun.
func buildMatrixRowForYears(startYear, endYear int, snapshot OutletSubscriptionSnapshot, outletSubs []SubscriptionDetail) SubscriptionMatrixRow {
	brand := strings.TrimSpace(snapshot.OwnerBrandName.String)
	name := strings.TrimSpace(snapshot.OutletName)
	brandOutlet := name
	if brand != "" && name != "" {
		brandOutlet = fmt.Sprintf("%s / %s", brand, name)
	} else if brand != "" {
		brandOutlet = brand
	}

	pic := strings.TrimSpace(snapshot.LatestPIC.String)
	if pic == "" {
		pic = "-"
	}

	row := SubscriptionMatrixRow{
		OwnerCode:     snapshot.OwnerCode.String,
		OwnerName:     snapshot.OwnerName.String,
		OwnerPhone:    snapshot.OwnerPhone.String,
		BrandOutlet:   brandOutlet,
		PICInfo:       pic,
		City:          snapshot.OutletCity.String,
		Province:      snapshot.OutletProvince.String,
		Region:        formatRegion(snapshot.OutletProvince.String),
		CreatedAtYear: snapshot.CreatedAt.Year(),
	}

	sorted := make([]SubscriptionDetail, len(outletSubs))
	copy(sorted, outletSubs)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].ActiveFrom.Valid {
			return true
		}
		if !sorted[j].ActiveFrom.Valid {
			return false
		}
		return sorted[i].ActiveFrom.Time.Before(sorted[j].ActiveFrom.Time)
	})

	var firstSubFrom time.Time
	for _, sub := range sorted {
		if sub.ActiveFrom.Valid {
			firstSubFrom = sub.ActiveFrom.Time
			break
		}
	}

	var effectiveStartMonth time.Time
	if !snapshot.CreatedAt.IsZero() {
		effectiveStartMonth = time.Date(snapshot.CreatedAt.Year(), snapshot.CreatedAt.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if !firstSubFrom.IsZero() {
		firstSubMonth := time.Date(firstSubFrom.Year(), firstSubFrom.Month(), 1, 0, 0, 0, 0, time.UTC)
		if effectiveStartMonth.IsZero() || firstSubMonth.Before(effectiveStartMonth) {
			effectiveStartMonth = firstSubMonth
		}
	}

	for year := startYear; year <= endYear; year++ {
		yd := SubscriptionMatrixYearData{Year: year}
		for m := 1; m <= 12; m++ {
			// Atur default
			yd.MonthlyVals[m-1] = ""
			yd.MonthlyCode[m-1] = ""

			mStart := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
			mNext := mStart.AddDate(0, 1, 0) // awal bulan berikutnya (eksklusif)

			// Jangan proses data langganan di bulan sebelum outlet dibuat atau langganan pertama dimulai
			if !effectiveStartMonth.IsZero() && mStart.Before(effectiveStartMonth) {
				continue
			}

			// Cari langganan yang aktif di bulan ini.
			// Jika ada overlap, pilih yang active_until paling jauh (durasi terlama).
			var activeSub *SubscriptionDetail
			for i := range sorted {
				sub := &sorted[i]
				if !sub.ActiveFrom.Valid || !sub.ActiveUntil.Valid {
					continue
				}
				subFrom := sub.ActiveFrom.Time
				subUntil := sub.ActiveUntil.Time
				// Aktif di bulan ini: active_from < mNext && active_until >= mStart
				if subFrom.Before(mNext) && !subUntil.Before(mStart) {
					if activeSub == nil ||
						subUntil.After(activeSub.ActiveUntil.Time) ||
						(subUntil.Equal(activeSub.ActiveUntil.Time) && sub.ID > activeSub.ID) {
						activeSub = sub
					}
				}
			}

			if activeSub != nil {
				subFrom := activeSub.ActiveFrom.Time
				subUntil := activeSub.ActiveUntil.Time
				pkg := packageAbbrev(activeSub)
				tenure := activeSub.TenureMonths.Int64

				isFirstMonth := subFrom.Year() == year && int(subFrom.Month()) == m
				isFirstEver := !firstSubFrom.IsZero() && subFrom.Equal(firstSubFrom)

				// Jika langganan habis di bulan ini (active_until >= mStart AND < mNext),
				// tampilkan tanggal jatuh tempo atau U-XX.
				if !subUntil.Before(mStart) && subUntil.Before(mNext) {
					if subUntil.Before(time.Now()) {
						// Jatuh tempo sudah lewat → tampilkan U-XX (contoh: U-BS, U-PR, U-BC)
						yd.MonthlyVals[m-1] = "U-" + packageAbbrev(activeSub)
						yd.MonthlyCode[m-1] = OutletSubscriptionStatusInactive
					} else {
						yd.MonthlyVals[m-1] = subUntil.Format("02/01/06")
						yd.MonthlyCode[m-1] = OutletSubscriptionStatusDue
					}
					continue
				}

				if isFirstMonth && isFirstEver {
					yd.MonthlyVals[m-1] = "N-" + pkg + tenorSuffixNew(tenure)
					yd.MonthlyCode[m-1] = OutletSubscriptionStatusNew
				} else if isFirstMonth {
					if tenure == 1 {
						yd.MonthlyVals[m-1] = "S-" + pkg
					} else {
						yd.MonthlyVals[m-1] = "S-" + pkg + "-F" + tenorSuffixFollowing(tenure)
					}
					yd.MonthlyCode[m-1] = OutletSubscriptionStatusSubscribed
				} else {
					yd.MonthlyVals[m-1] = "F-" + pkg + tenorSuffixFollowing(tenure)
					yd.MonthlyCode[m-1] = OutletSubscriptionStatusSubscribed
				}
				continue
			} else {
				// Tidak ada langganan aktif di bulan ini.
				// "U-XX" hanya diisi jika:
				//   1. Bulan ini sudah lewat (mStart ≤ hari ini) — bulan masa depan tetap kosong
				//   2. Outlet pernah berlangganan sebelumnya (findLastPriorSub != nil)
				// Kode paket diambil dari langganan terakhir yang berakhir sebelum bulan ini.
				now := time.Now().UTC()
				monthHasPassed := !mStart.After(now)
				if monthHasPassed {
					if priorSub := findLastPriorSub(sorted, mStart); priorSub != nil {
						yd.MonthlyVals[m-1] = "U-" + packageAbbrev(priorSub)
						yd.MonthlyCode[m-1] = OutletSubscriptionStatusInactive
					}
				}
			}
		}
		row.YearsData = append(row.YearsData, yd)
	}

	return row
}

func (s *Service) ExportSubscriptionMatrix(ctx context.Context, actor Actor, params OutletSubscriptionStatusParams, year int) ([]SubscriptionMatrixRow, int, error) {
	if year <= 0 {
		year = time.Now().UTC().Year()
	}
	params.All = true

	var baseRefMonth time.Time
	if params.Month != "" {
		if t, err := time.Parse("2006-01", params.Month); err == nil {
			baseRefMonth = t
		}
	}
	if baseRefMonth.IsZero() {
		baseRefMonth = time.Date(year, time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Filters are no longer cleared here so the user can filter the matrix by subscription status or due date.

	snapshots, total, err := s.repo.ListOutletSubscriptionSnapshots(ctx, actor, params, baseRefMonth, false)
	if err != nil {
		return nil, year, err
	}
	log.Printf("[ExportMatrix] snapshots count=%d total=%d", len(snapshots), total)

	var outletIDs []int64
	for _, snap := range snapshots {
		outletIDs = append(outletIDs, snap.OutletID)
	}

	subsMap, err := s.repo.GetSubscriptionsForOutlets(ctx, outletIDs)
	if err != nil {
		return nil, year, err
	}

	const yearLookahead = 2
	maxYear := year + yearLookahead

	minYear := year
	for _, snap := range snapshots {
		// Fallback to CreatedAt (Outlet creation year)
		if !snap.CreatedAt.IsZero() {
			y := snap.CreatedAt.Year()
			if y > 0 && y < minYear {
				minYear = y
			}
		}
		// Look for earliest subscription date
		if subs, ok := subsMap[snap.OutletID]; ok {
			for _, sub := range subs {
				if sub.ActiveFrom.Valid {
					y := sub.ActiveFrom.Time.Year()
					if y > 0 && y < minYear {
						minYear = y
					}
				}
			}
		}
	}
	log.Printf("[ExportMatrix] year range=%d..%d (lookahead=%d)", minYear, maxYear, yearLookahead)

	matrixRows := make([]SubscriptionMatrixRow, 0, len(snapshots))

	for _, snapshot := range snapshots {
		outletSubs := subsMap[snapshot.OutletID]
		row := buildMatrixRowForYears(minYear, maxYear, snapshot, outletSubs)
		if !params.HasActivityOnly || matrixRowHasActivity(row) {
			matrixRows = append(matrixRows, row)
		}
	}

	return matrixRows, minYear, nil
}

type matrixSheetStyles struct {
	header         int
	subscribed     int
	renewal        int
	dueDate        int
	newStyle       int
	unsubscribe    int
	textCell       int
	centerTextCell int
	yellowHL       int
	emptyCell      int
}

func buildLegendSheet(file *excelize.File, sheet string, st matrixSheetStyles) {
	_ = file.SetColWidth(sheet, "A", "A", 18)
	_ = file.SetColWidth(sheet, "B", "B", 40)
	_ = file.SetColWidth(sheet, "C", "C", 60)

	headers := []string{"KODE", "NAMA STATUS", "KETERANGAN"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = file.SetCellValue(sheet, cell, h)
		_ = file.SetCellStyle(sheet, cell, cell, st.header)
	}
	_ = file.SetRowHeight(sheet, 1, 26)

	legends := [][]string{
		{"N-PR-24", "Nasabah Baru-Pro-Tenor 24", "Nasabah baru yang berlangganan paket Pro 24 Bulan."},
		{"N-BS-24", "Nasabah Baru-Bisnis-Tenor 24", "Nasabah baru yang berlangganan paket bisnis 2 tahun."},
		{"N-PR-18", "Nasabah Baru-Pro-Tenor 18", "Nasabah baru yang berlangganan paket Pro 18 Bulan."},
		{"N-BS-18", "Nasabah Baru-Bisnis-Tenor 18", "Nasabah baru yang berlangganan paket bisnis 18 Bulan."},
		{"N-PR-F", "Nasabah Baru-Pro-Following", "Nasabah baru yang berlangganan paket Pro 1 tahun."},
		{"N-BS-F", "Nasabah Baru-Bisnis-Following", "Nasabah baru yang berlangganan paket bisnis 1 tahun."},
		{"N-BC-F", "Nasabah Baru-Basic-Following", "Nasabah baru yang berlangganan paket basic 1 tahun."},
		{"N-PR-9", "Nasabah Baru-Pro-Tenor 9", "Nasabah baru yang berlangganan paket Pro 9 Bulan."},
		{"N-BS-9", "Nasabah Baru-Bisnis-Tenor 9", "Nasabah baru yang berlangganan paket bisnis 9 bulan."},
		{"N-BC-9", "Nasabah Baru-Basic-Tenor 9", "Nasabah baru yang berlangganan paket Basic 9 bulan."},
		{"N-PR-6", "Nasabah Baru-Pro-Tenor 6", "Nasabah baru yang berlangganan paket Pro 6 Bulan."},
		{"N-BS-6", "Nasabah Baru-Bisnis-Tenor 6", "Nasabah baru yang berlangganan paket bisnis 6 Bulan."},
		{"N-BC-6", "Nasabah Baru-Basic-Tenor 6", "Nasabah baru yang berlangganan paket basic 6 Bulan."},
		{"N-PR-3", "Nasabah Baru-Pro-Tenor 3", "Nasabah baru yang berlangganan paket Pro 3 Bulan."},
		{"N-BS-3", "Nasabah Baru-Bisnis-Tenor 3", "Nasabah baru yang berlangganan paket bisnis 3 Bulan."},
		{"N-BC-3", "Nasabah Baru-Basic-Tenor 3", "Nasabah baru yang berlangganan paket basic 3 Bulan."},
		{"N-PR", "Nasabah Baru-Pro", "Nasabah baru yang berlangganan paket Pro 1 Bulan."},
		{"N-BS", "Nasabah Baru-Bisnis", "Nasabah baru yang berlangganan paket Bisnis 1 Bulan."},
		{"N-BC", "Nasabah Baru-Basic", "Nasabah baru yang berlangganan paket Basic 1 Bulan."},
		{"S-PR", "Subscribe Pro", "Nasabah yang berlangganan per bulan paket Pro"},
		{"S-BS", "Subscribe-Bisnis", "Nasabah yang berlangganan per bulan paket bisnis."},
		{"S-BC", "Subscribe-Basic", "Nasabah yang berlangganan per bulan paket basic."},
		{"F-PR(24)", "Following-Pro(Tenor) 24", "Mengikuti aktif paket Pro dari bulan sebelumnya ex. 24 bln"},
		{"F-PR(18)", "Following-Pro(Tenor) 18", "Mengikuti aktif paket Pro dari bulan sebelumnya ex. 18 bln"},
		{"F-PR", "Following-Pro(Tenor) 12", "Mengikuti aktif paket Pro dari bulan sebelumnya ex. 12 bln"},
		{"F-PR(6)", "Following-Pro(Tenor) 6", "Mengikuti aktif paket Pro dari bulan sebelumnya ex. 6 bln"},
		{"F-PR(3)", "Following-Pro", "Mengikuti aktif paket Pro dari bulan sebelumnya ex. 3 bln"},
		{"F-BS(24)", "Following-Bisnis(Tenor) 24", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 24 bln"},
		{"F-BS(18)", "Following-Bisnis(Tenor) 18", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 18 bln"},
		{"F-BS", "Following-Bisnis", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 12 bln"},
		{"F-BS(9)", "Following-Bisnis(Tenor) 9", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 9 bln"},
		{"F-BC(9)", "Following-Basic(Tenor) 9", "Mengikuti aktif paket Basic dari bulan sebelumnya ex. 9 bln"},
		{"F-BS(6)", "Following-Bisnis(Tenor) 6", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 6 bln"},
		{"F-BC(6)", "Following-Basic(Tenor) 6", "Mengikuti aktif paket Basic dari bulan sebelumnya ex. 6 bln"},
		{"F-BS(3)", "Following-Bisnis(Tenor) 3", "Mengikuti aktif paket Bisnis dari bulan sebelumnya ex. 3 bln"},
		{"F-BC(3)", "Following-Basic(Tenor) 3", "Mengikuti aktif paket Basic dari bulan sebelumnya ex. 3 bln"},
		{"F-BC(24)", "Following-Basic(Tenor) 24", "Mengikuti aktif paket Basic dari bulan sebelumnya ex. 24 bln"},
		{"F-BC", "Following-Basic", "Mengikuti aktif paket Basic dari bulan sebelumnya ex. 12 bln"},
		{"S-PR-F(24)", "Subscribe-Pro-Following(Tenor) 24", "Nasabah lama yang berlangganan Pro 2 tahun."},
		{"S-PR-F(18)", "Subscribe-Pro-Following(Tenor) 18", "Nasabah lama yang berlangganan Pro 18 bulan."},
		{"S-PR-F(9)", "Subscribe-Pro-Following(Tenor) 9", "Nasabah lama yang berlangganan Pro 9 bulan."},
		{"S-PR-F(6)", "Subscribe-Pro-Following(Tenor) 6", "Nasabah lama yang berlangganan Pro 6 bulan."},
		{"S-PR-F(3)", "Subscribe-Pro-Following(Tenor) 3", "Nasabah lama yang berlangganan Pro 3 bulan."},
		{"S-BS-F(24)", "Subscribe-Bisnis-Following(Tenor) 24", "Nasabah lama yang berlangganan Bisnis 2 tahun."},
		{"S-BS-F(18)", "Subscribe-Bisnis-Following(Tenor) 18", "Nasabah lama yang berlangganan Bisnis 18 Bulan."},
		{"S-PR-F(12)", "Subscribe-Pro-Following(Tenor) 12", "Nasabah lama yang berlangganan Pro 1 tahun."},
		{"S-BS-F(12)", "Subscribe-Bisnis-Following(Tenor) 12", "Nasabah lama yang berlangganan Bisnis 1 tahun."},
		{"S-BS-(9)", "Subscribe-Bisnis-(Tenor) 9", "Nasabah lama yang berlangganan Bisnis 9 bulan."},
		{"S-BS-(6)", "Subscribe-Bisnis-Following(Tenor) 6", "Nasabah lama yang berlangganan Bisnis 6 bulan."},
		{"S-BC-(6)", "Subscribe-Basic-Following(Tenor) 6", "Nasabah lama yang berlangganan Basic 6 bulan."},
		{"S-BS-(3)", "Subscribe-Bisnis-Following(Tenor) 3", "Nasabah lama yang berlangganan Bisnis 3 bulan."},
		{"S-BC-(3)", "Subscribe-Basic-Following(Tenor) 3", "Nasabah lama yang berlangganan Basic 3 bulan."},
		{"S-BC-F(24)", "Subscribe-Bisnis-Following(Tenor) 24", "Nasabah lama yang berlangganan Bisnis 2 tahun."},
		{"S-BC-F(12)", "Subscribe-Basic-Following(Tenor) 12", "Nasabah lama yang berlangganan Basic 1 tahun."},
		{"U-PR", "Unsubscribe Pro", "Nasabah yang tidak berlangganan kembali di bulan tersebut (paket Pro)."},
		{"U-BS", "Unsubscribe Bisnis", "Nasabah yang tidak berlangganan kembali di bulan tersebut (paket Bisnis)."},
		{"U-BC", "Unsubscribe Basic", "Nasabah yang tidak berlangganan kembali di bulan tersebut (paket Basic)."},
	}

	for rIdx, l := range legends {
		rowNum := rIdx + 2
		c1, _ := excelize.CoordinatesToCellName(1, rowNum)
		c2, _ := excelize.CoordinatesToCellName(2, rowNum)
		c3, _ := excelize.CoordinatesToCellName(3, rowNum)
		_ = file.SetCellValue(sheet, c1, l[0])
		_ = file.SetCellValue(sheet, c2, l[1])
		_ = file.SetCellValue(sheet, c3, l[2])

		_ = file.SetCellStyle(sheet, c1, c1, st.yellowHL)
		_ = file.SetCellStyle(sheet, c2, c3, st.textCell)
	}

	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      1,
		YSplit:      1,
		TopLeftCell: "B2",
		ActivePane:  "bottomRight",
	})
}

// buildMatrixSheet mengisi satu sheet Excel untuk multi-tahun secara horizontal.
func buildMatrixSheet(file *excelize.File, sheet string, baseYear int, rows []SubscriptionMatrixRow, st matrixSheetStyles) {
	if len(rows) == 0 {
		return
	}
	
	totalYears := len(rows[0].YearsData)

	// Row 1: Merged Headers
	_ = file.MergeCell(sheet, "A1", "I1")
	_ = file.SetCellValue(sheet, "A1", "INFORMASI OWNER & OUTLET")
	_ = file.SetCellStyle(sheet, "A1", "I1", st.header)

	for i := 0; i < totalYears; i++ {
		startCol := 10 + (i * 12)
		endCol := startCol + 11
		startCell, _ := excelize.CoordinatesToCellName(startCol, 1)
		endCell, _ := excelize.CoordinatesToCellName(endCol, 1)
		
		yearStr := fmt.Sprintf("BERLANGGANAN %d", baseYear+i)
		_ = file.MergeCell(sheet, startCell, endCell)
		_ = file.SetCellValue(sheet, startCell, yearStr)
		_ = file.SetCellStyle(sheet, startCell, endCell, st.header)
	}

	// Row 2: Sub Headers
	headers := []string{
		"Kode Owner", "Nama Owner", "No. Hp Owner", "Nama Brand/Outlet",
		"No Hp & Nama PIC Kelolaan", "Kota", "Provinsi", "Wilayah", "Tahun Dibuat",
	}
	
	// Add months for each year
	for i := 0; i < totalYears; i++ {
		headers = append(headers, "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des")
	}

	for idx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 2)
		_ = file.SetCellValue(sheet, cell, h)
		_ = file.SetCellStyle(sheet, cell, cell, st.header)
	}

	_ = file.SetRowHeight(sheet, 1, 26)
	_ = file.SetRowHeight(sheet, 2, 26)

	// Fill data rows
	for rIdx, r := range rows {
		rowNum := rIdx + 3

		_ = file.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), r.OwnerCode)
		_ = file.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), r.OwnerName)
		_ = file.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), r.OwnerPhone)
		_ = file.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), r.BrandOutlet)
		_ = file.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), r.PICInfo)
		_ = file.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), r.City)
		_ = file.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), r.Province)
		_ = file.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), r.Region)
		_ = file.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), r.CreatedAtYear)

		for col := 1; col <= 9; col++ {
			cName, _ := excelize.CoordinatesToCellName(col, rowNum)
			if col == 8 || col == 9 { // Wilayah & Tahun Dibuat
				_ = file.SetCellStyle(sheet, cName, cName, st.yellowHL)
			} else if col == 1 || col == 3 { // Kode Owner & No HP Owner
				_ = file.SetCellStyle(sheet, cName, cName, st.centerTextCell)
			} else {
				_ = file.SetCellStyle(sheet, cName, cName, st.textCell)
			}
		}

		// Fill year data
		for yIdx, yd := range r.YearsData {
			for m := 0; m < 12; m++ {
				col := 10 + (yIdx * 12) + m
				cName, _ := excelize.CoordinatesToCellName(col, rowNum)
				val := yd.MonthlyVals[m]
				
				_ = file.SetCellValue(sheet, cName, val)

				switch {
				case val == "":
					_ = file.SetCellStyle(sheet, cName, cName, st.emptyCell)
				case strings.Contains(val, "/"): // Tanggal Jatuh Tempo DD/MM/YY
					_ = file.SetCellStyle(sheet, cName, cName, st.dueDate)
				case strings.HasPrefix(val, "S-"): // Subscribe (Perpanjangan)
					_ = file.SetCellStyle(sheet, cName, cName, st.renewal)
				case strings.HasPrefix(val, "F-"): // Following
					_ = file.SetCellStyle(sheet, cName, cName, st.subscribed)
				case strings.HasPrefix(val, "N-"): // Nasabah Baru
					_ = file.SetCellStyle(sheet, cName, cName, st.newStyle)
				default:
					_ = file.SetCellStyle(sheet, cName, cName, st.unsubscribe)
				}
			}
		}
	}

	// Column widths
	for c := 1; c <= 9; c++ {
		cLetter, _ := excelize.ColumnNumberToName(c)
		_ = file.SetColWidth(sheet, cLetter, cLetter, 18)
	}
	
	totalCols := 9 + (totalYears * 12)
	for c := 10; c <= totalCols; c++ {
		cLetter, _ := excelize.ColumnNumberToName(c)
		_ = file.SetColWidth(sheet, cLetter, cLetter, 12)
	}

	// Freeze panes: beku 9 kolom info & 2 baris header
	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      9,
		YSplit:      2,
		TopLeftCell: "J3",
		ActivePane:  "bottomRight",
	})
}

// BuildSubscriptionMatrixXLSX membuat file Excel.
func BuildSubscriptionMatrixXLSX(exportYear int, baseYear int, rows []SubscriptionMatrixRow) ([]byte, error) {
	file := excelize.NewFile()

	// --- Buat shared styles (level file, bisa dipakai di semua sheet) ---
	baseCellBorder := []excelize.Border{
		{Type: "top", Style: 1, Color: "E5E7EB"},
		{Type: "bottom", Style: 1, Color: "E5E7EB"},
		{Type: "left", Style: 1, Color: "E5E7EB"},
		{Type: "right", Style: 1, Color: "E5E7EB"},
	}

	headerRedStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Color: "FFFFFF", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C92C1E"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "top", Style: 1, Color: "B02015"},
			{Type: "bottom", Style: 1, Color: "B02015"},
			{Type: "left", Style: 1, Color: "B02015"},
			{Type: "right", Style: 1, Color: "B02015"},
		},
	})
	subscribeStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Color: "276749", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E6F4EA"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	renewalStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Color: "975A16", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEEBC8"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	dueDateStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Color: "C92C1E", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FCE8E6"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	newStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Bold: true, Color: "1C598A", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F4FD"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	unsubscribeStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Color: "718096", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F7FAFC"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	textCellStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 9, Color: "1A202C"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	centerTextCellStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 9, Color: "1A202C"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	yellowHighlightStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 9, Color: "744210", Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FEFCBF"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})
	emptyCellStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Calibri", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFFFFF"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    baseCellBorder,
	})

	st := matrixSheetStyles{
		header:         headerRedStyle,
		subscribed:     subscribeStyle,
		renewal:        renewalStyle,
		dueDate:        dueDateStyle,
		newStyle:       newStyle,
		unsubscribe:    unsubscribeStyle,
		textCell:       textCellStyle,
		centerTextCell: centerTextCellStyle,
		yellowHL:       yellowHighlightStyle,
		emptyCell:      emptyCellStyle,
	}

	sheetYear := exportYear
	if sheetYear <= 0 {
		sheetYear = baseYear
	}
	sheetName := fmt.Sprintf("Monthly Active %d", sheetYear)
	_ = file.SetSheetName(file.GetSheetName(0), sheetName)
	buildMatrixSheet(file, sheetName, baseYear, rows, st)

	legendSheet := "Copy of Kode"
	_, _ = file.NewSheet(legendSheet)
	buildLegendSheet(file, legendSheet, st)

	if idx, err := file.GetSheetIndex(sheetName); err == nil && idx >= 0 {
		file.SetActiveSheet(idx)
	}

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BuildSubscriptionImportTemplateXLSX() ([]byte, error) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	file.SetSheetName(sheet, "Template Import Langganan")

	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"C92C1E"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	sampleStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "4A5568"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	headers := []string{
		"KODE_OUTLET", "KODE_OWNER", "KODE_PAKET", "KODE_PLAN",
		"TANGGAL_MULAI", "TANGGAL_BERAKHIR", "CATATAN",
	}

	for idx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = file.SetCellValue(sheet, cell, h)
		_ = file.SetCellStyle(sheet, cell, cell, headerStyle)
		colLetter, _ := excelize.ColumnNumberToName(idx + 1)
		_ = file.SetColWidth(sheet, colLetter, colLetter, 22)
	}

	sampleRow := []string{
		"OUT-1001", "OWN-1020", "PKG-BASIC", "PLAN-1M",
		"2026-08-01", "2026-09-01", "Perpanjangan Paket Langganan 1 Bulan",
	}

	for idx, val := range sampleRow {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 2)
		_ = file.SetCellValue(sheet, cell, val)
		_ = file.SetCellStyle(sheet, cell, cell, sampleStyle)
	}

	_ = file.SetRowHeight(sheet, 1, 26)
	_ = file.SetRowHeight(sheet, 2, 22)

	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
