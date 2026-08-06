package customer

import (
	"database/sql"
	"testing"
	"time"
)

func TestClassifyOutletSubscription_NeverSubscribed(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	code, label, remaining, remainingDisplay, lastDisplay := classifyOutletSubscription(OutletSubscriptionSnapshot{}, referenceMonth, false)
	if code != OutletSubscriptionStatusNotSubscribe || label != OutletSubscriptionStatusNotSubscribe {
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
	code, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false)
	if code != OutletSubscriptionStatusExpired {
		t.Fatalf("expected expired, got %s", code)
	}
	if remaining == nil || *remaining != -46 {
		t.Fatalf("unexpected remaining days: %+v (*remaining=%d)", remaining, *remaining)
	}
}

func TestClassifyOutletSubscription_WillBeDue(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// reference month end = 2026-06-30. SubscriptionEnd = 2026-07-10 (10 days in future from refDate).
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	code, label, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false)
	if code != OutletSubscriptionStatusWillBeDue || label != "AKAN JATUH TEMPO" {
		t.Fatalf("expected akan jatuh tempo, got %s (%s)", code, label)
	}
	if remaining == nil || *remaining != -10 {
		t.Fatalf("expected remaining -10, got %+v", remaining)
	}
	if remainingDisplay != "-10 hari" {
		t.Fatalf("expected '-10 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_Due(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// reference month end = 2026-06-30. SubscriptionEnd = 2026-06-30 (diffDays = 0).
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	code, label, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false)
	if code != OutletSubscriptionStatusDue || label != "JATUH TEMPO" {
		t.Fatalf("expected jatuh tempo, got %s (%s)", code, label)
	}
	if remaining == nil || *remaining != 0 {
		t.Fatalf("expected remaining 0, got %+v", remaining)
	}
	if remainingDisplay != "0 hari" {
		t.Fatalf("expected '0 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_PassedDue(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	// reference month end = 2026-06-30. SubscriptionEnd = 2026-06-20 (diffDays = -10).
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	code, label, remaining, remainingDisplay, _ := classifyOutletSubscription(snapshot, referenceMonth, false)
	if code != OutletSubscriptionStatusPassedDue || label != "TELAH JATUH TEMPO" {
		t.Fatalf("expected telah jatuh tempo, got %s (%s)", code, label)
	}
	if remaining == nil || *remaining != 10 {
		t.Fatalf("expected remaining 10 (positive), got %+v", remaining)
	}
	if remainingDisplay != "10 hari" {
		t.Fatalf("expected '10 hari', got %s", remainingDisplay)
	}
}

func TestClassifyOutletSubscription_New(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	code, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth, false)
	if code != OutletSubscriptionStatusNew {
		t.Fatalf("expected new, got %s", code)
	}
	if remaining == nil || *remaining != 40 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}
