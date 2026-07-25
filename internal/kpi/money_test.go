package kpi

import "testing"

func TestParsePercent(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{"whole", "40", 4000, false},
		{"two decimals", "40.50", 4050, false},
		{"one decimal", "40.5", 4050, false},
		{"zero", "0", 0, false},
		{"exactly 100", "100.00", 10000, false},
		{"over 100", "100.01", 0, true},
		{"negative", "-1", 0, true},
		{"empty", "", 0, true},
		{"non-numeric", "abc", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePercent(tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.value, err)
			}
			if got != tc.want {
				t.Fatalf("parsePercent(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}
