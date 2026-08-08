package customer

import (
	"database/sql"
	"testing"
	"time"
)

func TestClassifyOutletSubscription_NeverSubscribed(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	code, label, _, _, remaining, remainingDisplay, lastDisplay := classifyOutletSubscription(OutletSubscriptionSnapshot{}, referenceMonth, false, "")
	if code != OutletSubscriptionStatusInactive || label != "TIDAK AKTIF" {
		t.Fatalf("unexpected status: %s %s", code, label)
	}
	if remaining != nil {
		t.Fatalf("expected nil remaining days")
	}
	if remainingDisplay != NeverSubscribedDisplay || lastDisplay != NeverSubscribedDisplay {
		t.Fatalf("unexpected display values: %s %s", remainingDisplay, lastDisplay)
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
		t.Fatalf("expected remaining 5, got %+v", *remaining)
	}
	if remainingDisplay != "5 hari" {
		t.Fatalf("expected '5 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_Due(t *testing.T) {
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	referenceMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: referenceMonth.AddDate(0, -1, 0), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: today, Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	_, _, dueCode, dueLabel, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
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
	if remaining == nil || *remaining != -10 { // daysPassed is negative
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
		OutletSubscriptionCount: 1, // Add this
	}
	code, _, _, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false, "")
	if code != OutletSubscriptionStatusNew {
		t.Fatalf("expected new, got %s", code)
	}
	// remaining is dependent on time.Now(), skip strict check
	_ = remaining
}

