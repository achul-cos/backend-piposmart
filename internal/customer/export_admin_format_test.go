package customer

import "testing"

func TestToAdminOwnerOutletRowsUsesRowCodeAndTestingCategory(t *testing.T) {
	rows := []map[string]any{
		{
			"row_code":            "RB-001",
			"owner_code":          "OWN-001",
			"owner_name":          "Demo Owner",
			"owner_email":         "demo@example.com",
			"owner_phone":         "08123456789",
			"brand_name":          "Demo Brand",
			"owner_created_at":    "2026-08-08T00:00:00Z",
			"entered_by_name":     "Admin Demo",
			"outlet_name":         "Outlet A",
			"outlet_phone":        "08123456780",
			"outlet_city":         "Medan",
			"outlet_province":     "Sumatera Utara",
			"outlet_district":     "Medan Kota",
			"outlet_sub_district": "Teladan",
			"outlet_address":      "Jalan Mawar",
			"outlet_created_at":   "2026-08-08T00:00:00Z",
			"outlet_count":        int64(1),
			"is_testing_account":  true,
			"status_terbaru":      "Berlangganan",
			"akuisisi":            "Berlangganan",
			"pic":                 "Risky",
			"tanggal_dibagikan_1": "08/08/2026",
			"share_1":             "Risky",
			"kategori_nasabah_1":  "Hot Lead",
		},
	}

	items := ToAdminOwnerOutletRows(rows)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if got := items[0]["kode_baris"]; got != "RB-001" {
		t.Fatalf("kode_baris = %v, want RB-001", got)
	}
	if got := items[0]["kategori_akun"]; got != "Akun Testing" {
		t.Fatalf("kategori_akun = %v, want Akun Testing", got)
	}
	if got := items[0]["share_1"]; got != "Risky" {
		t.Fatalf("share_1 = %v, want Risky", got)
	}
	if got := items[0]["kategori_nasabah_1"]; got != "Hot Lead" {
		t.Fatalf("kategori_nasabah_1 = %v, want Hot Lead", got)
	}
}

func TestToAdminOwnerRowsKeepsFirstRowCodePerOwner(t *testing.T) {
	rows := []map[string]any{
		{
			"row_code":            "RB-001",
			"owner_code":          "OWN-001",
			"owner_name":          "Demo Owner",
			"owner_email":         "demo@example.com",
			"owner_phone":         "08123456789",
			"brand_name":          "Demo Brand",
			"owner_created_at":    "2026-08-08T00:00:00Z",
			"outlet_created_at":   "2026-08-08T00:00:00Z",
			"outlet_city":         "Medan",
			"outlet_province":     "Sumatera Utara",
			"outlet_district":     "Medan Kota",
			"outlet_sub_district": "Teladan",
			"outlet_address":      "Jalan Mawar",
			"outlet_count":        int64(2),
			"owner_balance":       "50000",
		},
		{
			"row_code":            "RB-002",
			"owner_code":          "OWN-001",
			"owner_name":          "Demo Owner",
			"owner_email":         "demo@example.com",
			"owner_phone":         "08123456789",
			"brand_name":          "Demo Brand",
			"owner_created_at":    "2026-08-08T00:00:00Z",
			"outlet_created_at":   "2026-08-09T00:00:00Z",
			"outlet_city":         "Medan",
			"outlet_province":     "Sumatera Utara",
			"outlet_district":     "Medan Kota",
			"outlet_sub_district": "Teladan",
			"outlet_address":      "Jalan Mawar 2",
			"outlet_count":        int64(2),
			"owner_balance":       "50000",
		},
	}

	items := ToAdminOwnerRows(rows)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if got := items[0]["kode_baris"]; got != "RB-001" {
		t.Fatalf("kode_baris = %v, want RB-001", got)
	}
}
