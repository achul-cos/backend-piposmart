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

func TestNewPartnerCommissionResponse(t *testing.T) {
	now := time.Now()
	approvedAt := now.Add(time.Hour)

	pc := partner.PartnerCommission{
		ID:               1,
		Code:             "COM-20260724-000001-000001",
		PartnerID:        2,
		PartnerCode:      sql.NullString{String: "REF-001", Valid: true},
		PartnerName:      sql.NullString{String: "Mitra Referral 001", Valid: true},
		ReferralID:       3,
		ClosingID:        4,
		ClosingCode:      sql.NullString{String: "CLS-20260724-000004-000001", Valid: true},
		CommissionMode:   partner.CommissionModePercentage,
		CommissionValue:  "5.00",
		BaseAmount:       "1000000.00",
		CommissionAmount: "50000.00",
		Currency:         "IDR",
		Status:           partner.CommissionStatusPending,
		Note:             sql.NullString{},
		ApprovedByUserID: sql.NullInt64{},
		ApprovedAt:       sql.NullTime{},
		PaidByUserID:     sql.NullInt64{},
		PaidAt:           sql.NullTime{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := partner.NewPartnerCommissionResponse(pc)
	if resp.PartnerCode == nil || *resp.PartnerCode != "REF-001" {
		t.Errorf("expected partner code REF-001, got %v", resp.PartnerCode)
	}
	if resp.CommissionMode != partner.CommissionModePercentage || resp.CommissionValue != "5.00" {
		t.Errorf("expected mode=PERCENTAGE value=5.00, got mode=%s value=%s", resp.CommissionMode, resp.CommissionValue)
	}
	if resp.ApprovedBy != nil {
		t.Errorf("expected nil ApprovedBy for unapproved commission, got %v", resp.ApprovedBy)
	}
	if resp.Status != partner.CommissionStatusPending {
		t.Errorf("expected status PENDING, got %s", resp.Status)
	}

	pc.ApprovedByUserID = sql.NullInt64{Int64: 9, Valid: true}
	pc.ApprovedByName = sql.NullString{String: "Admin Utama", Valid: true}
	pc.ApprovedAt = sql.NullTime{Time: approvedAt, Valid: true}
	pc.Status = partner.CommissionStatusApproved

	resp = partner.NewPartnerCommissionResponse(pc)
	if resp.ApprovedBy == nil || resp.ApprovedBy.ID != 9 || resp.ApprovedBy.Name != "Admin Utama" {
		t.Errorf("expected ApprovedBy {9, Admin Utama}, got %v", resp.ApprovedBy)
	}
	if resp.ApprovedAt == nil || !resp.ApprovedAt.Equal(approvedAt) {
		t.Errorf("expected ApprovedAt %v, got %v", approvedAt, resp.ApprovedAt)
	}
}

func TestNewPartnerTypeResponse(t *testing.T) {
	pt := partner.PartnerType{
		ID:              1,
		Code:            "AGENT",
		Name:            "Agent Regional",
		CommissionMode:  partner.CommissionModeFixed,
		CommissionValue: "150000.00",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	resp := partner.NewPartnerTypeResponse(pt)
	if resp.CommissionMode != partner.CommissionModeFixed {
		t.Errorf("expected commission mode FIXED, got %s", resp.CommissionMode)
	}
	if resp.CommissionValue != "150000.00" {
		t.Errorf("expected commission value 150000.00, got %s", resp.CommissionValue)
	}
}

func TestNewCommissionRuleResponse(t *testing.T) {
	now := time.Now()

	// PERCENTAGE/FIXED rule: Value present, Tiers empty.
	r := partner.CommissionRule{
		ID:            1,
		PartnerTypeID: 2,
		PackageID:     sql.NullInt64{Int64: 3, Valid: true},
		PackageCode:   sql.NullString{String: "PRO", Valid: true},
		PackageName:   sql.NullString{String: "Pro Package", Valid: true},
		Mode:          partner.CommissionModePercentage,
		Value:         sql.NullString{String: "7.00", Valid: true},
		EffectiveFrom: now,
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	resp := partner.NewCommissionRuleResponse(r)
	if resp.PackageID == nil || *resp.PackageID != 3 {
		t.Errorf("expected package_id 3, got %v", resp.PackageID)
	}
	if resp.Value == nil || *resp.Value != "7.00" {
		t.Errorf("expected value 7.00, got %v", resp.Value)
	}
	if len(resp.Tiers) != 0 {
		t.Errorf("expected no tiers for PERCENTAGE rule, got %d", len(resp.Tiers))
	}

	// TIER rule: Value absent, Tiers populated.
	r.Mode = partner.CommissionModeTier
	r.Value = sql.NullString{}
	r.Tiers = []partner.CommissionTier{
		{ID: 10, CommissionRuleID: 1, TierOrder: 1, MinClosings: 1, MaxClosings: sql.NullInt64{Int64: 3, Valid: true}, Mode: partner.CommissionModePercentage, Value: "2.00"},
		{ID: 11, CommissionRuleID: 1, TierOrder: 2, MinClosings: 4, Mode: partner.CommissionModePercentage, Value: "5.00"},
	}
	resp = partner.NewCommissionRuleResponse(r)
	if resp.Value != nil {
		t.Errorf("expected nil value for TIER rule, got %v", *resp.Value)
	}
	if len(resp.Tiers) != 2 {
		t.Fatalf("expected 2 tiers, got %d", len(resp.Tiers))
	}
	if resp.Tiers[0].MaxClosings == nil || *resp.Tiers[0].MaxClosings != 3 {
		t.Errorf("expected tier 1 max_closings 3, got %v", resp.Tiers[0].MaxClosings)
	}
	if resp.Tiers[1].MaxClosings != nil {
		t.Errorf("expected tier 2 (top tier) max_closings nil, got %v", *resp.Tiers[1].MaxClosings)
	}
}

func TestNewPartnerPayoutResponse(t *testing.T) {
	now := time.Now()

	p := partner.PartnerPayout{
		ID:          1,
		Code:        "PAYOUT-20260724-000002-000001",
		PartnerID:   2,
		PartnerCode: sql.NullString{String: "PTR-002", Valid: true},
		PartnerName: sql.NullString{String: "Mitra Dua", Valid: true},
		TotalAmount: "150000.00",
		Currency:    "IDR",
		Status:      partner.PayoutStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	resp := partner.NewPartnerPayoutResponse(p)
	if resp.Status != partner.PayoutStatusPending {
		t.Errorf("expected status PENDING, got %s", resp.Status)
	}
	if resp.PaidBy != nil {
		t.Errorf("expected nil PaidBy for a PENDING payout, got %v", resp.PaidBy)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected no items, got %d", len(resp.Items))
	}

	paidAt := now.Add(time.Hour)
	p.Status = partner.PayoutStatusPaid
	p.PaidByUserID = sql.NullInt64{Int64: 5, Valid: true}
	p.PaidByName = sql.NullString{String: "Admin Utama", Valid: true}
	p.PaidAt = sql.NullTime{Time: paidAt, Valid: true}
	p.Items = []partner.PartnerPayoutItem{
		{ID: 1, PayoutID: 1, CommissionID: 9, CommissionCode: sql.NullString{String: "COM-001", Valid: true}, Amount: "50000.00", CreatedAt: now},
		{ID: 2, PayoutID: 1, CommissionID: 10, CommissionCode: sql.NullString{String: "COM-002", Valid: true}, Amount: "100000.00", CreatedAt: now},
	}

	resp = partner.NewPartnerPayoutResponse(p)
	if resp.PaidBy == nil || resp.PaidBy.ID != 5 || resp.PaidBy.Name != "Admin Utama" {
		t.Errorf("expected PaidBy {5, Admin Utama}, got %v", resp.PaidBy)
	}
	if resp.PaidAt == nil || !resp.PaidAt.Equal(paidAt) {
		t.Errorf("expected PaidAt %v, got %v", paidAt, resp.PaidAt)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].CommissionCode == nil || *resp.Items[0].CommissionCode != "COM-001" {
		t.Errorf("expected first item commission_code COM-001, got %v", resp.Items[0].CommissionCode)
	}
	if resp.Items[0].ReleasedAt != nil {
		t.Errorf("expected nil ReleasedAt for an active item, got %v", resp.Items[0].ReleasedAt)
	}
}

func ptr(s string) *string {
	return &s
}
