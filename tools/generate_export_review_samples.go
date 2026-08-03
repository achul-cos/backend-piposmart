package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

type column struct {
	Label string
}

type report struct {
	FileBase string
	Title    string
	Columns  []column
	Rows     [][]string
}

func main() {
	outDir := filepath.Join("storage", "exports", "review_samples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	reports := []report{
		{
			FileBase: "admin_owner_outlet_review_sample",
			Title:    "Admin Owner & Outlet Review Sample",
			Columns: []column{
				{Label: "Tahun Owner"},
				{Label: "Bulan Owner"},
				{Label: "Tanggal Owner"},
				{Label: "Kode Owner"},
				{Label: "Email Owner"},
				{Label: "No Hp Owner"},
				{Label: "Nama Project/Brand"},
				{Label: "Nama Outlet"},
				{Label: "No Hp Outlet"},
				{Label: "Tahun Outlet"},
				{Label: "Bulan Outlet"},
				{Label: "Tanggal Outlet"},
				{Label: "Kelurahan Outlet"},
				{Label: "Kecamatan Outlet"},
				{Label: "Kota/Kabupaten Outlet"},
				{Label: "Provinsi Outlet"},
				{Label: "Alamat Outlet"},
				{Label: "PIC/Lead"},
				{Label: "Waktu Dibagikan"},
				{Label: "Waktu Mulai Berlangganan"},
				{Label: "Paket Langganan"},
				{Label: "Tenor"},
				{Label: "Waktu Berakhir Berlangganan"},
				{Label: "Status Langganan"},
			},
			Rows: [][]string{
				{"2025", "Juni", "02/06/2025", "OWN-00012", "budi@laundrycerah.id", "081234567890", "Laundry Cerah", "Laundry Cerah Outlet 1", "081298765432", "2025", "Juli", "02/07/2025", "", "", "Bandung", "Jawa Barat", "Jl. Sukajadi No. 10", "Sales Rani", "05/07/2025 10:30", "10/07/2025", "Business", "12 Bulan", "04/07/2026", "BERLANGGANAN"},
				{"2025", "Juni", "02/06/2025", "OWN-00012", "budi@laundrycerah.id", "081234567890", "Laundry Cerah", "Laundry Cerah Outlet 2", "081211119999", "2025", "Agustus", "14/08/2025", "", "", "Bandung", "Jawa Barat", "Jl. Pasteur No. 88", "Sales Rani", "15/08/2025 14:10", "18/08/2025", "Business", "12 Bulan", "17/08/2026", "BERLANGGANAN"},
				{"2025", "Juli", "12/07/2025", "OWN-00013", "sari@freshclean.id", "081255577788", "Fresh Clean", "Fresh Clean Cabang Antapani", "081277766655", "2025", "Juli", "15/07/2025", "", "", "Bandung", "Jawa Barat", "Jl. Antapani Raya No. 5", "Supervisor Dimas", "16/07/2025 09:00", "", "", "", "", "BELUM BERLANGGANAN"},
			},
		},
		{
			FileBase: "admin_new_subscribe_review_sample",
			Title:    "Admin New & Subscribe Review Sample",
			Columns: []column{
				{Label: "Date Of Work"},
				{Label: "Kode Owner"},
				{Label: "Nama Owner"},
				{Label: "No. Hp Owner"},
				{Label: "No. Hp Outlet"},
				{Label: "Project/Outlet"},
				{Label: "Kota"},
				{Label: "Provinsi"},
				{Label: "Created Date"},
				{Label: "Date Top UP System"},
				{Label: "Nominal Aktivasi"},
				{Label: "Tanggal Aktivasi"},
				{Label: "Paket Membership"},
				{Label: "Status"},
			},
			Rows: [][]string{
				{"2026-07-23", "OWN-00021", "Andi Saputra", "081300001111", "081300009999", "Laundry Kilat Outlet 1", "Surabaya", "Jawa Timur", "2026-07-20", "2026-07-23 09:15", "300000", "2026-07-23 09:20", "Pro 1 Bulan", "RECONCILED"},
				{"2026-07-24", "OWN-00022", "Mega Putri", "081322233344", "081344455566", "Smart Wash", "Malang", "Jawa Timur", "2026-07-21", "2026-07-24 13:00", "150000", "2026-07-24 13:05", "Basic 1 Bulan", "PAID"},
			},
		},
		{
			FileBase: "admin_nasabah_baru_provinsi_review_sample",
			Title:    "Admin Nasabah Baru Per Provinsi Review Sample",
			Columns: []column{
				{Label: "Tahun Member"},
				{Label: "Bulan"},
				{Label: "Kode Owner"},
				{Label: "Nama Owner"},
				{Label: "No. Hp Owner"},
				{Label: "Email"},
				{Label: "Project/Outlet"},
				{Label: "Kota"},
				{Label: "Alamat"},
				{Label: "Provinsi"},
			},
			Rows: [][]string{
				{"2026", "Juli", "OWN-00031", "Rina Amelia", "081211112222", "rina@email.com", "Rina Laundry", "Semarang", "Jl. Pandanaran No. 1", "Jawa Tengah"},
				{"2026", "Juli", "OWN-00032", "Dodi Pratama", "081233334444", "dodi@email.com", "Dodi Clean", "Solo", "Jl. Slamet Riyadi No. 8", "Jawa Tengah"},
			},
		},
	}

	for _, rpt := range reports {
		if err := writeXLSX(filepath.Join(outDir, rpt.FileBase+".xlsx"), rpt); err != nil {
			panic(err)
		}
		if err := writeCSV(filepath.Join(outDir, rpt.FileBase+".csv"), rpt); err != nil {
			panic(err)
		}
	}
	if err := writePDF(filepath.Join(outDir, "admin_owner_outlet_review_sample.pdf"), reports[0]); err != nil {
		panic(err)
	}

	fmt.Println("review sample exports generated in", outDir)
}

func writeXLSX(path string, rpt report) error {
	f := excelize.NewFile()
	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"C92C1E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "C92C1E"},
	})
	metaLabelStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10},
	})
	metaValueStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10},
	})
	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})

	startRow := 1
	if rpt.FileBase == "admin_owner_outlet_review_sample" {
		startRow = 7
		_ = f.MergeCell(sheet, "C1", "H1")
		_ = f.SetCellValue(sheet, "C1", "Report Owner & Outlet")
		_ = f.SetCellStyle(sheet, "C1", "H1", titleStyle)
		_ = f.SetCellValue(sheet, "C3", "Tanggal Export")
		_ = f.SetCellStyle(sheet, "C3", "C3", metaLabelStyle)
		_ = f.SetCellValue(sheet, "D3", time.Now().Format("02/01/2006"))
		_ = f.SetCellStyle(sheet, "D3", "D3", metaValueStyle)
		if logoPath := filepath.Join("asset", "piposmart-vertical.png"); fileExists(logoPath) {
			_ = f.AddPicture(sheet, "A1", logoPath, &excelize.GraphicOptions{
				AutoFit:     false,
				ScaleX:      0.55,
				ScaleY:      0.55,
				Positioning: "oneCell",
			})
		}
	}

	for i, col := range rpt.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, startRow)
		f.SetCellValue(sheet, cell, col.Label)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colName, colName, 20)
	}

	for rowIdx, row := range rpt.Rows {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+startRow+1)
			f.SetCellValue(sheet, cell, value)
			f.SetCellStyle(sheet, cell, cell, bodyStyle)
		}
	}

	return f.SaveAs(path)
}

func writeCSV(path string, rpt report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, col := range rpt.Columns {
		if i > 0 {
			_, _ = f.WriteString(",")
		}
		_, _ = f.WriteString(csvEscape(col.Label))
	}
	_, _ = f.WriteString("\n")

	for _, row := range rpt.Rows {
		for i, value := range row {
			if i > 0 {
				_, _ = f.WriteString(",")
			}
			_, _ = f.WriteString(csvEscape(value))
		}
		_, _ = f.WriteString("\n")
	}
	return nil
}

func writePDF(path string, rpt report) error {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, rpt.Title, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Arial", "B", 8)
	colWidth := 277.0 / float64(len(rpt.Columns))
	for _, col := range rpt.Columns {
		pdf.CellFormat(colWidth, 8, col.Label, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)
	for _, row := range rpt.Rows {
		for i := range rpt.Columns {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			pdf.CellFormat(colWidth, 8, value, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	return pdf.OutputFileAndClose(path)
}

func csvEscape(value string) string {
	return `"` + escapeQuotes(value) + `"`
}

func escapeQuotes(value string) string {
	out := ""
	for _, ch := range value {
		if ch == '"' {
			out += `""`
			continue
		}
		out += string(ch)
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
