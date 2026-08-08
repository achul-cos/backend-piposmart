package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	filePath := filepath.Join("asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx")
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Fatalf("Failed to open excel: %v", err)
	}
	defer f.Close()

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for i := 0; i < len(rows) && i < 3; i++ {
			fmt.Printf("Sheet: %s, Row %d: %s\n", sheet, i, strings.Join(rows[i], " | "))
		}
	}
}
