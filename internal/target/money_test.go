package target

import "testing"

func TestValidateDecimal(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"whole number", "10", false},
		{"two decimals", "10.50", false},
		{"one decimal", "10.5", false},
		{"zero", "0.00", false},
		{"empty", "", true},
		{"negative", "-5.00", true},
		{"too many decimals", "10.505", true},
		{"non-numeric", "abc", true},
		{"multiple dots", "10.5.5", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDecimal(tc.value)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q, got nil", tc.value)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for %q, got %v", tc.value, err)
			}
		})
	}
}
