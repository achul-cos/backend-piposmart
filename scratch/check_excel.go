package main

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

func main() {
	searchPaths := []string{
		filepath.Join("asset", "data_admin", "06. Data Bonus Mitra 2025-2026 (Copy).xlsx"),
	}

	for _, fn := range searchPaths {
		fmt.Printf("Trying to open %s\n", fn)
		f, err := excelize.OpenFile(fn)
		if err == nil {
			fmt.Println("Successfully opened file")
			sheets := f.GetSheetList()
			fmt.Println("Sheets:")
			for _, sheet := range sheets {
				fmt.Println("-", sheet)
				
				// Read a few rows from sheet
				rows, err := f.GetRows(sheet)
				if err != nil {
					fmt.Println("  Error reading rows:", err)
					continue
				}
				fmt.Println("  Total rows:", len(rows))
				for i, row := range rows {
					if i >= 3 {
						break
					}
					fmt.Printf("  Row %d: %v\n", i, row)
				}
			}
			f.Close()
			return
		} else {
			fmt.Println("Error:", err)
		}
	}
}
