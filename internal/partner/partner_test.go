package partner_test

import (
	"database/sql"
	"testing"
	"time"

	"backend_crm_piposmart/internal/partner"
	"backend_crm_piposmart/internal/platform/config"

	_ "github.com/go-sql-driver/mysql"
)

func TestMaskedAccountResponse(t *testing.T) {
	tests := []struct {
		name     string
		last4    sql.NullString
		expected *string
	}{
		{
			name:     "Valid last4",
			last4:    sql.NullString{String: "1234", Valid: true},
			expected: ptr("****1234"),
		},
		{
			name:     "Invalid last4",
			last4:    sql.NullString{Valid: false},
			expected: nil,
		},
		{
			name:     "Empty string last4",
			last4:    sql.NullString{String: "", Valid: true},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := partner.Partner{
				ID:               1,
				PartnerTypeID:    1,
				Code:             "PTR-001",
				Name:             "Partner Test",
				BankAccountLast4: tt.last4,
				Status:           partner.StatusActive,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			resp := partner.NewPartnerResponse(p)
			if tt.expected == nil {
				if resp.BankAccountMasked != nil {
					t.Errorf("expected nil BankAccountMasked, got %v", *resp.BankAccountMasked)
				}
			} else {
				if resp.BankAccountMasked == nil || *resp.BankAccountMasked != *tt.expected {
					t.Errorf("expected %v, got %v", *tt.expected, resp.BankAccountMasked)
				}
			}
		})
	}
}

func TestEncryptionServiceInitialization(t *testing.T) {
	cfg := config.Config{}
	svc := partner.NewService(nil, cfg)
	if svc == nil {
		t.Fatal("expected service to be initialized")
	}
}

func ptr(s string) *string {
	return &s
}
