package partner

import "testing"

func TestParseMoneyToCents(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "whole number", value: "150000", want: 15000000},
		{name: "two decimals", value: "1000.50", want: 100050},
		{name: "one decimal padded", value: "1000.5", want: 100050},
		{name: "zero", value: "0", want: 0},
		{name: "empty", value: "", wantErr: true},
		{name: "negative", value: "-100", wantErr: true},
		{name: "too many decimals", value: "100.123", wantErr: true},
		{name: "not a number", value: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMoneyToCents(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatCents(t *testing.T) {
	tests := []struct {
		value int64
		want  string
	}{
		{value: 100050, want: "1000.50"},
		{value: 0, want: "0.00"},
		{value: -550, want: "-5.50"},
	}
	for _, tt := range tests {
		if got := formatCents(tt.value); got != tt.want {
			t.Errorf("formatCents(%d) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestParseCommissionRate(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "zero", value: "0", want: 0},
		{name: "whole percent", value: "5", want: 500},
		{name: "two decimals", value: "5.25", want: 525},
		{name: "max", value: "100", want: 10000},
		{name: "max with decimals", value: "100.00", want: 10000},
		{name: "over max", value: "100.01", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "empty", value: "", wantErr: true},
		{name: "garbage", value: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommissionRate(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCalculateCommissionCents(t *testing.T) {
	tests := []struct {
		name           string
		baseCents      int64
		rateHundredths int64
		want           int64
	}{
		{name: "5.25% of Rp1,000.00", baseCents: 100000, rateHundredths: 525, want: 5250},
		{name: "zero rate", baseCents: 100000, rateHundredths: 0, want: 0},
		{name: "100% rate", baseCents: 250000, rateHundredths: 10000, want: 250000},
		{name: "rounds half up", baseCents: 333, rateHundredths: 50, want: 2}, // 333 * 0.5% = 1.665 -> rounds to 2
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCommissionCents(tt.baseCents, tt.rateHundredths)
			if got != tt.want {
				t.Errorf("calculateCommissionCents(%d, %d) = %d, want %d", tt.baseCents, tt.rateHundredths, got, tt.want)
			}
		})
	}
}

func TestValidateCommissionValue(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		value   string
		wantErr bool
	}{
		{name: "percentage valid", mode: CommissionModePercentage, value: "5.00"},
		{name: "percentage over 100", mode: CommissionModePercentage, value: "150.00", wantErr: true},
		{name: "fixed valid", mode: CommissionModeFixed, value: "150000.00"},
		{name: "fixed negative", mode: CommissionModeFixed, value: "-1.00", wantErr: true},
		{name: "unknown mode", mode: "BOGUS", value: "5.00", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommissionValue(tt.mode, tt.value)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCalculateCommissionAmountCents(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		value     string
		baseCents int64
		want      int64
		wantErr   bool
	}{
		{name: "percentage 5% of Rp1,000,000.00", mode: CommissionModePercentage, value: "5.00", baseCents: 100000000, want: 5000000},
		{name: "fixed flat regardless of base", mode: CommissionModeFixed, value: "150000.00", baseCents: 100000, want: 15000000},
		{name: "fixed flat on zero base still pays flat", mode: CommissionModeFixed, value: "150000.00", baseCents: 0, want: 15000000},
		{name: "unknown mode errors", mode: "BOGUS", value: "5.00", baseCents: 100000, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateCommissionAmountCents(tt.mode, tt.value, tt.baseCents)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%d)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
