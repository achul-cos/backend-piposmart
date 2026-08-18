package customer

import (
	"database/sql"
	"testing"
	"time"
)

func TestClassifyOutletSubscription_NeverSubscribed(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// Snapshot kosong tanpa subscription = TRIAL (belum pernah berlangganan), trial jauh di masa depan = BELUM JT
	// OwnerCreatedAt = zero time, trial habis jauh sebelum referenceMonth → NO PACKAGE / TELAH JATUH TEMPO
	code, label, dueCode, _, _, _, _ := classifyOutletSubscription(OutletSubscriptionSnapshot{}, referenceMonth, false, "")
	if code != OutletSubscriptionDueStatusNoPackage || label != "NO PACKAGE" {
		t.Fatalf("unexpected status: %s %s (expected NO_PACKAGE NO PACKAGE)", code, label)
	}
	if dueCode != OutletSubscriptionStatusPassedDue {
		t.Fatalf("unexpected due code: %s (expected TELAH_JATUH_TEMPO)", dueCode)
	}
}

func TestClassifyOutletSubscription_Unsubscribed_HasPastSubscription(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// Pernah berlangganan tapi sudah tidak aktif = UNSUBSCRIBE
	snapshot := OutletSubscriptionSnapshot{
		OwnerHasAnySubscription: true,
	}
	code, label, dueCode, dueLabel, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if code != OutletSubscriptionDueStatusNoPackage || label != "NO PACKAGE" {
		t.Fatalf("unexpected status: %s %s (expected NO PACKAGE)", code, label)
	}
	if dueCode != OutletSubscriptionStatusPassedDue || dueLabel != "TELAH JATUH TEMPO" {
		t.Fatalf("unexpected due status: %s %s (expected TELAH_JATUH_TEMPO TELAH JATUH TEMPO)", dueCode, dueLabel)
	}
	if remaining != nil {
		t.Fatalf("expected nil remaining days")
	}
}

func TestClassifyOutletSubscription_Expired(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 1, Valid: true},
	}
	code, _, _, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if code != OutletSubscriptionStatusInactive {
		t.Fatalf("expected expired, got %s", code)
	}
	// "remaining" is diffDays, which is negative for expired
	if remaining == nil || *remaining >= 0 {
		if remaining != nil {
			t.Fatalf("expected negative days passed, got %d", *remaining)
		} else {
			t.Fatalf("expected negative days passed, got %v", remaining)
		}
	}
}

func TestClassifyOutletSubscription_WillBeDue(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// For false isSpecificDate and different month, refDate is month end (June 30)
	refDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	endDate := refDate.AddDate(0, 0, 5)
	start := refDate.AddDate(0, 0, -40) // Bypass 'NEW' status

	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: start, Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: endDate, Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	_, _, dueCode, dueLabel, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if dueCode != OutletSubscriptionStatusWillBeDue || dueLabel != "AKAN JATUH TEMPO" {
		t.Fatalf("expected akan jatuh tempo, got %s (%s)", dueCode, dueLabel)
	}
	if remaining == nil || *remaining != 5 {
		t.Fatalf("expected remaining 5, got %+v", remaining)
	}
	if remainingDisplay != "5 hari" {
		t.Fatalf("expected '5 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_Due(t *testing.T) {
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	referenceMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)

	// 0 hari tersisa: active_until = hari ini = JATUH TEMPO
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: referenceMonth.AddDate(0, -1, 0), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: today, Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	_, _, dueCode, dueLabel, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	// 0 hari tersisa = JATUH TEMPO (jatuh tempo hari ini)
	if dueCode != OutletSubscriptionStatusDue || dueLabel != "JATUH TEMPO" {
		t.Fatalf("expected jatuh tempo, got %s (%s)", dueCode, dueLabel)
	}
	if remaining == nil || *remaining != 0 {
		t.Fatalf("expected remaining 0, got %+v", remaining)
	}
	if remainingDisplay != "0 hari" {
		t.Fatalf("expected '0 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_PassedDue(t *testing.T) {
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	referenceMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)

	endDate := today.AddDate(0, 0, -10)

	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: referenceMonth.AddDate(0, -2, 0), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: endDate, Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	_, _, dueCode, dueLabel, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if dueCode != OutletSubscriptionStatusPassedDue || dueLabel != "TELAH JATUH TEMPO" {
		t.Fatalf("expected telah jatuh tempo, got %s (%s)", dueCode, dueLabel)
	}
	if remaining == nil || *remaining != -10 {
		t.Fatalf("expected remaining -10, got %+v", *remaining)
	}
	if remainingDisplay != "-10 hari" {
		t.Fatalf("expected '-10 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_New(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart:       sql.NullTime{Time: time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:         sql.NullTime{Time: time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:            sql.NullInt64{Int64: 2, Valid: true},
		OutletSubscriptionCount: 1,
	}
	code, _, _, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if code != OutletSubscriptionStatusNew {
		t.Fatalf("expected new, got %s", code)
	}
	_ = remaining
}

func TestClassifyOutletSubscription_Trial_BelumJatuhTempo(t *testing.T) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Created 2 days ago, so 12 days remaining (> 7)
	createdAt := now.AddDate(0, 0, -2)
	snapshot := OutletSubscriptionSnapshot{
		OwnerCreatedAt:          createdAt,
		OwnerHasAnySubscription: false,
	}
	code, label, dueCode, dueLabel, remaining, _, _ := classifyOutletSubscription(snapshot, now, false, "")
	if code != OutletSubscriptionStatusTrial || dueCode != "BELUM_JATUH_TEMPO" {
		t.Fatalf("expected TRIAL and BELUM_JATUH_TEMPO, got %s and %s", code, dueCode)
	}
	if label != "TRIAL" || dueLabel != "BELUM JATUH TEMPO" {
		t.Fatalf("expected label TRIAL and BELUM JATUH TEMPO, got %s and %s", label, dueLabel)
	}
	if remaining == nil || *remaining != 12 {
		t.Fatalf("expected 12 days remaining, got %v", remaining)
	}
}

func TestClassifyOutletSubscription_Trial_AkanJatuhTempo(t *testing.T) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Created 10 days ago, so 4 days remaining (<= 7 and > 0)
	createdAt := now.AddDate(0, 0, -10)
	snapshot := OutletSubscriptionSnapshot{
		OwnerCreatedAt:          createdAt,
		OwnerHasAnySubscription: false,
	}
	code, label, dueCode, dueLabel, remaining, _, _ := classifyOutletSubscription(snapshot, now, false, "")
	if code != OutletSubscriptionStatusTrial || dueCode != OutletSubscriptionStatusWillBeDue {
		t.Fatalf("expected TRIAL and AKAN_JATUH_TEMPO, got %s and %s", code, dueCode)
	}
	if label != "TRIAL" || dueLabel != "AKAN JATUH TEMPO" {
		t.Fatalf("expected label TRIAL and AKAN JATUH TEMPO, got %s and %s", label, dueLabel)
	}
	if remaining == nil || *remaining != 4 {
		t.Fatalf("expected 4 days remaining, got %v", remaining)
	}
}

func TestClassifyOutletSubscription_Trial_Expired(t *testing.T) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Created 15 days ago → trial expired → NO PACKAGE / TELAH JATUH TEMPO
	createdAt := now.AddDate(0, 0, -15)
	snapshot := OutletSubscriptionSnapshot{
		OwnerCreatedAt:          createdAt,
		OwnerHasAnySubscription: false,
	}
	code, label, dueCode, dueLabel, _, _, _ := classifyOutletSubscription(snapshot, now, false, "")
	if code != OutletSubscriptionDueStatusNoPackage || label != "NO PACKAGE" {
		t.Fatalf("expected NO_PACKAGE, got %s %s", code, label)
	}
	if dueCode != OutletSubscriptionStatusPassedDue || dueLabel != "TELAH JATUH TEMPO" {
		t.Fatalf("expected TELAH JATUH TEMPO, got %s %s", dueCode, dueLabel)
	}
}

func TestClassifyOutletSubscription_Trial_ZeroDays_NoPackage(t *testing.T) {
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Created exactly 14 days ago → trial expires TODAY → NO PACKAGE / JATUH TEMPO
	createdAt := now.AddDate(0, 0, -14)
	snapshot := OutletSubscriptionSnapshot{
		OwnerCreatedAt:          createdAt,
		OwnerHasAnySubscription: false,
	}
	code, label, dueCode, dueLabel, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, now, false, "")
	if code != OutletSubscriptionDueStatusNoPackage || label != "NO PACKAGE" {
		t.Fatalf("expected NO_PACKAGE at 0 days trial, got %s %s", code, label)
	}
	if dueCode != OutletSubscriptionStatusDue || dueLabel != "JATUH TEMPO" {
		t.Fatalf("expected JATUH TEMPO at 0 days trial, got %s %s", dueCode, dueLabel)
	}
	if remaining == nil || *remaining != 0 {
		t.Fatalf("expected remaining 0, got %v", remaining)
	}
	if remainingDisplay != "0 hari" {
		t.Fatalf("expected '0 hari', got %s", remainingDisplay)
	}
}

func TestMultiSelectOutletSubscriptionStatusFilterCondition(t *testing.T) {
	normalized := normalizeOutletSubscriptionStatusCode("TRIAL, NEW, invalid")
	if normalized != "TRIAL,NEW" {
		t.Fatalf("expected 'TRIAL,NEW', got '%s'", normalized)
	}

	refMonth := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	cond, args := outletSubscriptionStatusFilterCondition("TRIAL,NEW", refMonth, false, "")
	if cond == "" {
		t.Fatalf("expected non-empty SQL condition for TRIAL,NEW")
	}
	if len(args) == 0 {
		t.Fatalf("expected non-empty args for TRIAL,NEW")
	}
}
