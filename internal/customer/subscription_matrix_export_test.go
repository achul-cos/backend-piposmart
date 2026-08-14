package customer

import (
	"database/sql"
	"testing"
	"time"
)

func dt(year int, month time.Month, day int) sql.NullTime {
	return sql.NullTime{Valid: true, Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// TestBuildMatrixRowForYears memverifikasi logika sel bulanan untuk satu outlet melintasi multi-tahun.
func TestBuildMatrixRowForYears(t *testing.T) {
	baseSnap := OutletSubscriptionSnapshot{
		OutletID:       1,
		OutletCode:     "OUT-001",
		OutletName:     "Outlet Test",
		OwnerCode:      sql.NullString{Valid: true, String: "OWN-001"},
		OwnerName:      sql.NullString{Valid: true, String: "Owner Test"},
		OwnerBrandName: sql.NullString{Valid: true, String: "Brand Test"},
		OutletCity:     sql.NullString{Valid: true, String: "Jakarta"},
		OutletProvince: sql.NullString{Valid: true, String: "DKI"},
		OwnerCreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Skenario 1: Langganan 1 bulan (Mei 2090)
	t.Run("1_bulan_mei", func(t *testing.T) {
		subs := []SubscriptionDetail{
			{
				ID: 1, OutletID: 1,
				ActiveFrom: dt(2090, time.May, 1), ActiveUntil: dt(2090, time.May, 31),
				PackageName: sql.NullString{Valid: true, String: "Pro"},
				TenureMonths: sql.NullInt64{Valid: true, Int64: 1},
			},
		}
		row := buildMatrixRowForYears(2090, 2090, baseSnap, subs)
		yd := row.YearsData[0]
		for m := 0; m < 4; m++ {
			if yd.MonthlyVals[m] != "" {
				t.Errorf("bulan %d: want empty, got %q", m+1, yd.MonthlyVals[m])
			}
		}
		if yd.MonthlyVals[4] != "31/05/90" {
			t.Errorf("Mei: want 31/05/90, got %q", yd.MonthlyVals[4])
		}
		for m := 5; m < 12; m++ {
			if yd.MonthlyVals[m] != "" {
				t.Errorf("bulan %d: want empty, got %q", m+1, yd.MonthlyVals[m])
			}
		}
	})

	// Skenario 2: Langganan setahun Mei 2090 - Apr 2091 -> N-BS-F
	t.Run("1_tahun_mei2090", func(t *testing.T) {
		subs := []SubscriptionDetail{
			{
				ID: 1, OutletID: 1,
				ActiveFrom: dt(2090, time.May, 1), ActiveUntil: dt(2091, time.April, 30),
				PackageName: sql.NullString{Valid: true, String: "Bisnis"},
				TenureMonths: sql.NullInt64{Valid: true, Int64: 12},
			},
		}
		row := buildMatrixRowForYears(2090, 2091, baseSnap, subs)
		yd90 := row.YearsData[0]
		yd27 := row.YearsData[1]

		for m := 0; m < 4; m++ {
			if yd90.MonthlyVals[m] != "" {
				t.Errorf("2090 bulan %d: want empty, got %q", m+1, yd90.MonthlyVals[m])
			}
		}
		if yd90.MonthlyVals[4] != "N-BS-F" {
			t.Errorf("2090 Mei: want N-BS-F, got %q", yd90.MonthlyVals[4])
		}
		for m := 5; m < 12; m++ {
			if yd90.MonthlyVals[m] != "F-BS" {
				t.Errorf("2090 bulan %d: want F-BS, got %q", m+1, yd90.MonthlyVals[m])
			}
		}

		for m := 0; m < 3; m++ {
			if yd27.MonthlyVals[m] != "F-BS" {
				t.Errorf("2091 bulan %d: want F-BS, got %q", m+1, yd27.MonthlyVals[m])
			}
		}
		if yd27.MonthlyVals[3] != "30/04/91" {
			t.Errorf("2091 Apr: want '30/04/91', got %q", yd27.MonthlyVals[3])
		}
		for m := 4; m < 12; m++ {
			if yd27.MonthlyVals[m] != "" {
				t.Errorf("2091 bulan %d: want empty, got %q", m+1, yd27.MonthlyVals[m])
			}
		}
	})

	// Skenario 3: Perpanjangan (S-)
	t.Run("perpanjangan_s_code", func(t *testing.T) {
		subs := []SubscriptionDetail{
			{
				ID: 1, OutletID: 1,
				ActiveFrom: dt(2090, time.January, 1), ActiveUntil: dt(2090, time.January, 31),
				PackageName: sql.NullString{Valid: true, String: "Basic"},
				TenureMonths: sql.NullInt64{Valid: true, Int64: 1},
			},
			{
				ID: 2, OutletID: 1,
				ActiveFrom: dt(2090, time.March, 1), ActiveUntil: dt(2090, time.May, 31),
				PackageName: sql.NullString{Valid: true, String: "Pro"},
				TenureMonths: sql.NullInt64{Valid: true, Int64: 3},
			},
		}
		row := buildMatrixRowForYears(2090, 2090, baseSnap, subs)
		yd := row.YearsData[0]
		if yd.MonthlyVals[0] != "31/01/90" {
			t.Errorf("Jan: want 31/01/90, got %q", yd.MonthlyVals[0])
		}
		if yd.MonthlyVals[1] != "" {
			t.Errorf("Feb: want empty, got %q", yd.MonthlyVals[1])
		}
		if yd.MonthlyVals[2] != "S-PR-F(3)" {
			t.Errorf("Mar: want S-PR-F(3), got %q", yd.MonthlyVals[2])
		}
		if yd.MonthlyVals[3] != "F-PR(3)" {
			t.Errorf("Apr: want F-PR(3), got %q", yd.MonthlyVals[3])
		}
		if yd.MonthlyVals[4] != "31/05/90" {
			t.Errorf("Mei: want 31/05/90, got %q", yd.MonthlyVals[4])
		}
		for m := 5; m < 12; m++ {
			if yd.MonthlyVals[m] != "" {
				t.Errorf("bulan %d: want empty, got %q", m+1, yd.MonthlyVals[m])
			}
		}
	})

	// Skenario 4: Tanpa langganan -> semua U (karena dibuat 2024)
	t.Run("tanpa_langganan", func(t *testing.T) {
		row := buildMatrixRowForYears(2090, 2090, baseSnap, nil)
		yd := row.YearsData[0]
		for m := 0; m < 12; m++ {
			if yd.MonthlyVals[m] != "" {
				t.Errorf("bulan %d: want empty, got %q", m+1, yd.MonthlyVals[m])
			}
		}
	})
}



func TestBuildSubscriptionMatrixXLSX(t *testing.T) {
	rows := []SubscriptionMatrixRow{
		{
			OwnerCode:   "OWN-101",
			OwnerName:   "CuCumbah Laundry",
			OwnerPhone:  "082120006502",
			BrandOutlet: "Cucumbah Laundry / Cucumbah Laundry",
			PICInfo:     "0821-0000-001 - Lidya",
			City:        "Kota Bekasi",
			Region:      "JABAR",
			YearsData: []SubscriptionMatrixYearData{
				{
					Year:        2090,
					MonthlyVals: [12]string{"", "", "", "", "N-BC", "F-BC", "F-BC", "31/07/90", "", "", "", ""},
					MonthlyCode: [12]string{"", "", "", "", "NEW", "SUBSCRIBE", "SUBSCRIBE", "JATUH_TEMPO", "", "", "", ""},
				},
				{
					Year:        2091,
					MonthlyVals: [12]string{"", "", "", "", "", "", "", "", "", "", "", ""},
					MonthlyCode: [12]string{"", "", "", "", "", "", "", "", "", "", "", ""},
				},
			},
		},
	}
	data, err := BuildSubscriptionMatrixXLSX(2090, 2090, rows)
	if err != nil {
		t.Fatalf("expected no error building XLSX matrix, got: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty byte slice for XLSX matrix")
	}
}

func TestBuildSubscriptionImportTemplateXLSX(t *testing.T) {
	data, err := BuildSubscriptionImportTemplateXLSX()
	if err != nil {
		t.Fatalf("expected no error building import template XLSX, got: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty byte slice for import template XLSX")
	}
}
