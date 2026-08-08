package reporting

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func ownerOutletExcelPath() string {
	return filepath.Join("asset", "data_admin", "01. Owner & Outlet 2021 - 2024 & 2025 - 2026.xlsx")
}

func nonRegisterExcelPath() string {
	return filepath.Join("asset", "data_admin", "04. Data Belum Registrasi 2026 - User Temp (Copy).xlsx")
}

const (
	exColNo            = 0
	exColDateOfWork    = 1
	exColNamaPenginput = 2
	exColKategoriAkun  = 3
	exColKodeBaris     = 4
	exColKodeOwner     = 5
	exColNamaOwner     = 6
	exColEmailOwner    = 7
	exColNoHpOwner     = 8
	exColNoHpOutlet    = 9
	exColCreateDate    = 10
	exColBulan         = 11
	exColBrandName     = 12
	exColNamaOutlet    = 13
	exColKelurahan     = 14
	exColKecamatan     = 15
	exColKota          = 16
	exColProvinsi      = 17
	exColAlamatLengkap = 18
)

func getExcelCol(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}

	value := strings.TrimSpace(fmt.Sprint(row[idx]))
	if value == "<nil>" || value == "" {
		return ""
	}

	return value
}

func hasValidOutlet(row []string) bool {
	outletName := getExcelCol(row, exColNamaOutlet)
	outletPhone := getExcelCol(row, exColNoHpOutlet)
	return outletName != "" || outletPhone != ""
}

// isRowInDateRange mengecek apakah tanggal pada baris Excel masuk dalam rentang dateFrom s/d dateTo
func isRowInDateRange(row []string, dateFrom, dateTo string) bool {
	if dateFrom == "" && dateTo == "" {
		return true
	}

	rawDate := getExcelCol(row, exColDateOfWork)
	if rawDate == "" {
		rawDate = getExcelCol(row, exColCreateDate)
	}
	if rawDate == "" {
		return true
	}

	parsedDate, err := parseExcelDate(rawDate)
	if err != nil {
		return true
	}

	if dateFrom != "" {
		if tFrom, err := time.Parse("2006-01-02", dateFrom); err == nil {
			if parsedDate.Before(tFrom) {
				return false
			}
		}
	}

	if dateTo != "" {
		if tTo, err := time.Parse("2006-01-02", dateTo); err == nil {
			tTo = tTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			if parsedDate.After(tTo) {
				return false
			}
		}
	}

	return true
}

func parseExcelDate(val string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"2006/01/02",
		"02/01/06",
		"2006-01-02 15:04:05",
	}
	for _, fmtStr := range formats {
		if t, err := time.Parse(fmtStr, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparsed date: %s", val)
}

func readAllExcelRows() ([][]string, error) {
	f, err := excelize.OpenFile(ownerOutletExcelPath())
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file Excel: %w", err)
	}
	defer f.Close()

	var allDataRows [][]string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) <= 1 {
			continue
		}
		allDataRows = append(allDataRows, rows[1:]...)
	}
	return allDataRows, nil
}

func readNonRegisterExcelRows() ([][]string, error) {
	f, err := excelize.OpenFile(nonRegisterExcelPath())
	if err != nil {
		return nil, fmt.Errorf("gagal membuka file Excel non register: %w", err)
	}
	defer f.Close()

	var allDataRows [][]string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) <= 2 {
			continue
		}
		allDataRows = append(allDataRows, rows[2:]...)
	}
	return allDataRows, nil
}

func isNrRowInDateRange(row []string, dateFrom, dateTo string) bool {
	if dateFrom == "" && dateTo == "" {
		return true
	}

	rawDate := getExcelCol(row, 0)
	if rawDate == "" {
		rawDate = getExcelCol(row, 5)
	}
	if rawDate == "" {
		return true
	}

	parsedDate, err := parseExcelDate(rawDate)
	if err != nil {
		return true
	}

	if dateFrom != "" {
		if tFrom, err := time.Parse("2006-01-02", dateFrom); err == nil {
			if parsedDate.Before(tFrom) {
				return false
			}
		}
	}

	if dateTo != "" {
		if tTo, err := time.Parse("2006-01-02", dateTo); err == nil {
			tTo = tTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			if parsedDate.After(tTo) {
				return false
			}
		}
	}

	return true
}

// determineAccountCategory menentukan Kategori Akun otomatis, dan mengembalikan Akun Testing jika aslinya adalah akun testing
func determineAccountCategory(ownerCode, rawKategori string, seenOwners map[string]bool) string {
	if strings.EqualFold(strings.TrimSpace(rawKategori), "akun testing") {
		return "Akun Testing"
	}

	// Jika ownerCode ini belum pernah ditemui sebelumnya -> Akun Baru
	if !seenOwners[ownerCode] {
		seenOwners[ownerCode] = true
		return "Akun Baru"
	}
	// Jika ownerCode sudah pernah ditemui -> Outlet Baru
	return "Outlet Baru"
}

// BuildOwnerOutletFromExcel -> Export Owner-Outlet lengkap
func BuildOwnerOutletFromExcel(dateFrom, dateTo string) ([]byte, error) {
	dataRows, err := readAllExcelRows()
	if err != nil {
		return nil, err
	}

	columns := GetAdminOwnerOutletColumns()

	// Pre-pass: kumpulkan data owner (nama, email, phone) yang terisi
	// dan petakan kode_baris ke owner_code yang paling baru (terakhir muncul)
	ownerInfoMap := make(map[string]map[string]string)
	kodeBarisToOwnerCode := make(map[string]string)

	for _, row := range dataRows {
		code := getExcelCol(row, exColKodeOwner)
		kodeBaris := getExcelCol(row, exColKodeBaris)

		if kodeBaris != "" && code != "" {
			kodeBarisToOwnerCode[kodeBaris] = code
		}

		if code == "" {
			continue
		}
		if ownerInfoMap[code] == nil {
			ownerInfoMap[code] = make(map[string]string)
		}
		if name := getExcelCol(row, exColNamaOwner); name != "" {
			ownerInfoMap[code]["name"] = name
		}
		if email := getExcelCol(row, exColEmailOwner); email != "" {
			ownerInfoMap[code]["email"] = email
		}
		if phone := getExcelCol(row, exColNoHpOwner); phone != "" {
			ownerInfoMap[code]["phone"] = phone
		}
		if outletPhone := getExcelCol(row, exColNoHpOutlet); outletPhone != "" {
			ownerInfoMap[code]["outlet_phone"] = outletPhone
		}
		if brandName := getExcelCol(row, exColBrandName); brandName != "" {
			ownerInfoMap[code]["brand_name"] = brandName
		}
		if outletName := getExcelCol(row, exColNamaOutlet); outletName != "" {
			ownerInfoMap[code]["outlet_name"] = outletName
		}
		if alamat := getExcelCol(row, exColAlamatLengkap); alamat != "" {
			ownerInfoMap[code]["alamat_lengkap"] = alamat
		}
	}

	items := make([]map[string]any, 0, len(dataRows))
	seenOwnersForCategory := make(map[string]bool)
	existingPhones := make(map[string]bool)

	for _, row := range dataRows {
		if phone := getExcelCol(row, exColNoHpOwner); phone != "" {
			existingPhones[phone] = true
		}
		if phone := getExcelCol(row, exColNoHpOutlet); phone != "" {
			existingPhones[phone] = true
		}

		if !hasValidOutlet(row) {
			continue
		}

		if !isRowInDateRange(row, dateFrom, dateTo) {
			continue
		}

		originalOwnerCode := getExcelCol(row, exColKodeOwner)
		kodeBaris := getExcelCol(row, exColKodeBaris)
		ownerCode := originalOwnerCode

		// Jika kode_baris ini memiliki owner_code yang lebih baru, gunakan yang baru
		if kodeBaris != "" && kodeBarisToOwnerCode[kodeBaris] != "" {
			ownerCode = kodeBarisToOwnerCode[kodeBaris]
		}

		rawKategori := getExcelCol(row, exColKategoriAkun)
		
		// Penentuan kategori akun otomatis menggunakan ownerCode (yang mungkin sudah di-update)
		kategoriAkun := determineAccountCategory(ownerCode, rawKategori, seenOwnersForCategory)

		// Ambil data dari baris ini
		ownerName := getExcelCol(row, exColNamaOwner)
		ownerEmail := getExcelCol(row, exColEmailOwner)
		ownerPhone := getExcelCol(row, exColNoHpOwner)
		outletPhone := getExcelCol(row, exColNoHpOutlet)
		brandName := getExcelCol(row, exColBrandName)
		outletName := getExcelCol(row, exColNamaOutlet)
		alamatLengkap := getExcelCol(row, exColAlamatLengkap)

		kelurahan := getExcelCol(row, exColKelurahan)
		kecamatan := getExcelCol(row, exColKecamatan)
		kota := getExcelCol(row, exColKota)
		provinsi := getExcelCol(row, exColProvinsi)

		// Jika ownerCode berubah (ikut owner yang baru), jangan gunakan data dari baris lama
		if ownerCode != originalOwnerCode {
			ownerName = ""
			ownerEmail = ""
			ownerPhone = ""
			outletPhone = ""
			brandName = ""
			outletName = ""
			alamatLengkap = ""
			// kelurahan, kecamatan, kota, provinsi dibiarkan sesuai aslinya atau bisa dikosongkan? 
			// user minta "kecuali kelurahan sampe provinsi itu jangan terisi" -> artinya kita kosongkan juga, atau biarkan asli?
			// Biarkan di-clear agar tidak terisi autofill, tapi kalau data asli ada, biarkan saja.
			// Karena tidak di-autofill, data asli tetap dipakai.
		}

		// Isi data yang kosong dengan data dari pre-pass (dari ownerCode yang relevan)
		if ownerName == "" && ownerInfoMap[ownerCode] != nil {
			ownerName = ownerInfoMap[ownerCode]["name"]
		}
		if ownerEmail == "" && ownerInfoMap[ownerCode] != nil {
			ownerEmail = ownerInfoMap[ownerCode]["email"]
		}
		if ownerPhone == "" && ownerInfoMap[ownerCode] != nil {
			ownerPhone = ownerInfoMap[ownerCode]["phone"]
		}
		if outletPhone == "" && ownerInfoMap[ownerCode] != nil {
			outletPhone = ownerInfoMap[ownerCode]["outlet_phone"]
		}
		if brandName == "" && ownerInfoMap[ownerCode] != nil {
			brandName = ownerInfoMap[ownerCode]["brand_name"]
		}
		if outletName == "" && ownerInfoMap[ownerCode] != nil {
			outletName = ownerInfoMap[ownerCode]["outlet_name"]
		}
		if alamatLengkap == "" && ownerInfoMap[ownerCode] != nil {
			alamatLengkap = ownerInfoMap[ownerCode]["alamat_lengkap"]
		}

		item := map[string]any{
			"no":                  len(items) + 1,
			"date_of_work":        getExcelCol(row, exColDateOfWork),
			"nama_penginput":      getExcelCol(row, exColNamaPenginput),
			"kategori_akun":       kategoriAkun,
			"kode_baris":          kodeBaris,
			"owner_code":          ownerCode,
			"owner_name":          ownerName,
			"owner_email":         ownerEmail,
			"owner_phone":         ownerPhone,
			"outlet_phone":        outletPhone,
			"create_date_project": getExcelCol(row, exColCreateDate),
			"brand_name":          brandName,
			"outlet_name":         outletName,
			"kelurahan":           kelurahan,
			"kecamatan":           kecamatan,
			"kota":                kota,
			"provinsi":            provinsi,
			"alamat_lengkap":      alamatLengkap,
			"jumlah_outlet":       0,
		}
		items = append(items, item)
	}

	nonRegRows, _ := readNonRegisterExcelRows()
	for _, nrRow := range nonRegRows {
		phone := getExcelCol(nrRow, 4)
		if phone == "" || existingPhones[phone] {
			continue
		}
		if !isNrRowInDateRange(nrRow, dateFrom, dateTo) {
			continue
		}
		kategoriAkun := getExcelCol(nrRow, 8)
		if kategoriAkun == "" {
			kategoriAkun = "Belum punya akun"
		}
		
		item := map[string]any{
			"no":                  len(items) + 1,
			"date_of_work":        getExcelCol(nrRow, 0),
			"nama_penginput":      getExcelCol(nrRow, 2),
			"kategori_akun":       kategoriAkun,
			"kode_baris":          "",
			"owner_code":          "",
			"owner_name":          "",
			"owner_email":         "",
			"owner_phone":         phone,
			"outlet_phone":        "",
			"create_date_project": getExcelCol(nrRow, 5),
			"brand_name":          "",
			"outlet_name":         "",
			"kelurahan":           "",
			"kecamatan":           "",
			"kota":                "",
			"provinsi":            "",
			"alamat_lengkap":      getExcelCol(nrRow, 10),
			"jumlah_outlet":       0,
		}
		items = append(items, item)
	}

	ownerOutletCount := make(map[string]int)
	for _, item := range items {
		code := fmt.Sprint(item["owner_code"])
		if code != "" {
			ownerOutletCount[code]++
		}
	}
	for _, item := range items {
		code := fmt.Sprint(item["owner_code"])
		item["jumlah_outlet"] = ownerOutletCount[code]
		

		ownerPhone := fmt.Sprint(item["owner_phone"])
		outletPhone := fmt.Sprint(item["outlet_phone"])

		if ownerPhone == "" {
			if outletPhone != "" {
				item["owner_phone"] = outletPhone
			}
		}
		if outletPhone == "" {
			if ownerPhone != "" {
				item["outlet_phone"] = ownerPhone
			}
		}
	}

	return buildAdminOwnerOutletXLSX("Report Owner & Outlet", "Data Owner Outlet", columns, items, dateFrom, dateTo)
}

// BuildOwnerFromExcel -> Export Data Owner Saja (Setiap Owner pasti Akun Baru)
func BuildOwnerFromExcel(dateFrom, dateTo string) ([]byte, error) {
	dataRows, err := readAllExcelRows()
	if err != nil {
		return nil, err
	}

	columns := GetAdminOwnerColumns()

	ownerOutletCount := make(map[string]int)
	existingPhones := make(map[string]bool)
	for _, row := range dataRows {
		code := getExcelCol(row, exColKodeOwner)
		if code != "" {
			ownerOutletCount[code]++
		}
		if phone := getExcelCol(row, exColNoHpOwner); phone != "" {
			existingPhones[phone] = true
		}
		if phone := getExcelCol(row, exColNoHpOutlet); phone != "" {
			existingPhones[phone] = true
		}
	}

	ownerInfoMap := make(map[string]map[string]string)
	for _, row := range dataRows {
		code := getExcelCol(row, exColKodeOwner)
		if code == "" {
			continue
		}
		if ownerInfoMap[code] == nil {
			ownerInfoMap[code] = make(map[string]string)
		}
		if email := getExcelCol(row, exColEmailOwner); email != "" {
			ownerInfoMap[code]["email"] = email
		}
		if phone := getExcelCol(row, exColNoHpOwner); phone != "" {
			ownerInfoMap[code]["phone"] = phone
		}
		if outletPhone := getExcelCol(row, exColNoHpOutlet); outletPhone != "" {
			ownerInfoMap[code]["outlet_phone"] = outletPhone
		}
	}

	seenOwners := make(map[string]bool)
	items := make([]map[string]any, 0, len(dataRows))

	for _, row := range dataRows {
		code := getExcelCol(row, exColKodeOwner)
		if code == "" {
			continue
		}

		if seenOwners[code] {
			continue
		}

		if !isRowInDateRange(row, dateFrom, dateTo) {
			continue
		}

		seenOwners[code] = true

		ownerEmail := getExcelCol(row, exColEmailOwner)
		ownerPhone := getExcelCol(row, exColNoHpOwner)

		outletPhone := ""
		if ownerInfoMap[code] != nil {
			if ownerEmail == "" {
				ownerEmail = ownerInfoMap[code]["email"]
			}
			if ownerPhone == "" {
				ownerPhone = ownerInfoMap[code]["phone"]
			}
			outletPhone = ownerInfoMap[code]["outlet_phone"]
		}
		
		if ownerPhone == "" {
			if outletPhone != "" {
				ownerPhone = outletPhone
			}
		}

		item := map[string]any{
			"no":                  len(items) + 1,
			"date_of_work":        getExcelCol(row, exColDateOfWork),
			"kode_baris":          getExcelCol(row, exColKodeBaris),
			"owner_code":          code,
			"owner_name":          getExcelCol(row, exColNamaOwner),
			"owner_email":         ownerEmail,
			"owner_phone":         ownerPhone,
			"create_date_project": getExcelCol(row, exColCreateDate),
			"brand_name":          getExcelCol(row, exColBrandName),
			"kelurahan":           getExcelCol(row, exColKelurahan),
			"kecamatan":           getExcelCol(row, exColKecamatan),
			"kota":                getExcelCol(row, exColKota),
			"provinsi":            getExcelCol(row, exColProvinsi),
			"alamat_lengkap":      getExcelCol(row, exColAlamatLengkap),
			"jumlah_outlet":       ownerOutletCount[code],
			"saldo_owner":         0,
		}
		items = append(items, item)
	}

	nonRegRows, _ := readNonRegisterExcelRows()
	for _, nrRow := range nonRegRows {
		phone := getExcelCol(nrRow, 4)
		if phone == "" || existingPhones[phone] {
			continue
		}
		if !isNrRowInDateRange(nrRow, dateFrom, dateTo) {
			continue
		}
		
		item := map[string]any{
			"no":                  len(items) + 1,
			"date_of_work":        getExcelCol(nrRow, 0),
			"kode_baris":          "",
			"owner_code":          "",
			"owner_name":          "",
			"owner_email":         "",
			"owner_phone":         phone,
			"create_date_project": getExcelCol(nrRow, 5),
			"brand_name":          "",
			"kelurahan":           "",
			"kecamatan":           "",
			"kota":                "",
			"provinsi":            "",
			"alamat_lengkap":      getExcelCol(nrRow, 10),
			"jumlah_outlet":       0,
			"saldo_owner":         0,
		}
		items = append(items, item)
	}

	return buildAdminOwnerOutletXLSX("Report Data Owner", "Data Owner", columns, items, dateFrom, dateTo)
}

func buildAdminRekapHeader(file *excelize.File, sheetName string, totalRows int) {
	headerBgColor := "E8C2B9"

	headerStyle, _ := file.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBgColor}, Pattern: 1},
	})
	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBgColor}, Pattern: 1},
	})
	metaLabelStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{headerBgColor}, Pattern: 1},
	})
	metaValueStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "right"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{headerBgColor}, Pattern: 1},
	})

	for r := 1; r <= 5; r++ {
		_ = file.SetRowHeight(sheetName, r, 24)
	}

	_ = file.SetColWidth(sheetName, "A", "A", 12)
	_ = file.SetColWidth(sheetName, "B", "B", 12)
	_ = file.SetColWidth(sheetName, "C", "C", 12)
	_ = file.SetColWidth(sheetName, "D", "D", 12)
	_ = file.SetColWidth(sheetName, "E", "E", 16)
	_ = file.SetColWidth(sheetName, "F", "F", 20)

	_ = file.SetCellStyle(sheetName, "A1", "H5", headerStyle)

	_ = file.MergeCell(sheetName, "A1", "D5")

	if logoPath := findPiposmartLogoPath(); logoPath != "" {
		_ = file.AddPicture(sheetName, "A1", logoPath, &excelize.GraphicOptions{
			AutoFit:     false,
			OffsetX:     6,
			OffsetY:     6,
			Positioning: "oneCell",
			ScaleX:      0.55,
			ScaleY:      0.55,
		})
	}

	_ = file.MergeCell(sheetName, "E1", "H2")
	_ = file.SetCellValue(sheetName, "E1", "Report Owner & Outlet")
	_ = file.SetCellStyle(sheetName, "E1", "H2", titleStyle)

	_ = file.SetCellValue(sheetName, "E3", "Diekspor:")
	_ = file.SetCellStyle(sheetName, "E3", "E3", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "F3", time.Now().In(time.Local).Format("02/01/2006"))
	_ = file.SetCellStyle(sheetName, "F3", "F3", metaValueStyle)

	_ = file.SetCellValue(sheetName, "E4", "Total Baris:")
	_ = file.SetCellStyle(sheetName, "E4", "E4", metaLabelStyle)
	_ = file.SetCellValue(sheetName, "F4", totalRows)
	_ = file.SetCellStyle(sheetName, "F4", "F4", metaValueStyle)
}

func ExcelFileTotalRows() (int, error) {
	dataRows, err := readAllExcelRows()
	if err != nil {
		return 0, err
	}
	return len(dataRows), nil
}

func BuildExcelRekapReport() ([]byte, error) {
	f, err := excelize.OpenFile(ownerOutletExcelPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	file := excelize.NewFile()
	sheetName := "Rekap"
	defaultSheet := file.GetSheetName(0)
	file.SetSheetName(defaultSheet, sheetName)

	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "C92C1E"},
	})
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C92C1E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})
	bodyStyle, _ := file.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "EAEAEA", Style: 1},
			{Type: "right", Color: "EAEAEA", Style: 1},
			{Type: "top", Color: "EAEAEA", Style: 1},
			{Type: "bottom", Color: "EAEAEA", Style: 1},
		},
	})

	_ = file.SetRowHeight(sheetName, 1, 30)
	_ = file.SetRowHeight(sheetName, 2, 30)

	headerBgStyle, _ := file.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFFFF"}, Pattern: 1},
	})
	_ = file.SetCellStyle(sheetName, "C1", "F2", headerBgStyle)

	_ = file.MergeCell(sheetName, "A1", "B2")
	if logoPath := findPiposmartLogoPath(); logoPath != "" {
		_ = file.AddPicture(sheetName, "A1", logoPath, &excelize.GraphicOptions{
			AutoFit:     false,
			OffsetX:     4,
			OffsetY:     4,
			Positioning: "oneCell",
			ScaleX:      0.5,
			ScaleY:      0.5,
		})
	}

	_ = file.MergeCell(sheetName, "C1", "F1")
	_ = file.SetCellValue(sheetName, "C1", "Rekap Data Owner & Outlet")
	_ = file.SetCellStyle(sheetName, "C1", "F1", titleStyle)

	row := 4
	headers := []string{"No", "Nama Sheet", "Jumlah Baris Data"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		_ = file.SetCellValue(sheetName, cell, h)
		_ = file.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	_ = file.SetColWidth(sheetName, "A", "A", 6)
	_ = file.SetColWidth(sheetName, "B", "B", 40)
	_ = file.SetColWidth(sheetName, "C", "C", 24)

	totalAll := 0
	for i, sheet := range f.GetSheetList() {
		rows, _ := f.GetRows(sheet)
		dataCount := 0
		if len(rows) > 1 {
			dataCount = len(rows) - 1
		}
		totalAll += dataCount
		row++
		cellA, _ := excelize.CoordinatesToCellName(1, row)
		cellB, _ := excelize.CoordinatesToCellName(2, row)
		cellC, _ := excelize.CoordinatesToCellName(3, row)
		_ = file.SetCellValue(sheetName, cellA, i+1)
		_ = file.SetCellValue(sheetName, cellB, sheet)
		_ = file.SetCellValue(sheetName, cellC, dataCount)
		_ = file.SetCellStyle(sheetName, cellA, cellC, bodyStyle)
	}

	row++
	totalLabelStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
		},
	})
	cellA, _ := excelize.CoordinatesToCellName(1, row)
	cellB, _ := excelize.CoordinatesToCellName(2, row)
	cellC, _ := excelize.CoordinatesToCellName(3, row)
	_ = file.MergeCell(sheetName, cellA, cellB)
	_ = file.SetCellValue(sheetName, cellA, "TOTAL")
	_ = file.SetCellValue(sheetName, cellC, totalAll)
	_ = file.SetCellStyle(sheetName, cellA, cellC, totalLabelStyle)

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}