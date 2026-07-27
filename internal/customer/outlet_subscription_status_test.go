package customer

import (
	"database/sql"
	"testing"
	"time"
)

func TestClassifyOutletSubscription_NeverSubscribed(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	code, label, remaining, remainingDisplay, lastDisplay := classifyOutletSubscription(OutletSubscriptionSnapshot{}, referenceMonth)
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
	code, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusExpired {
		t.Fatalf("expected expired, got %s", code)
	}
	if remaining == nil || *remaining != -46 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}

func TestClassifyOutletSubscription_NotSubscribeLongExpired(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 3, Valid: true},
	}
	code, _, _, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusNotSubscribe {
		t.Fatalf("expected not subscribe, got %s", code)
	}
}

func TestClassifyOutletSubscription_DueThisMonth(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 4, Valid: true},
	}
	code, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusDueThisMonth {
		t.Fatalf("expected jatuh tempo, got %s", code)
	}
	if remaining == nil || *remaining != -10 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}

func TestClassifyOutletSubscription_New(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.June, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 2, Valid: true},
	}
	code, _, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusNew {
		t.Fatalf("expected new, got %s", code)
	}
	if remaining == nil || *remaining != 40 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}

func TestClassifyOutletSubscription_OneMonthSpecialLabel(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 1, Valid: true},
	}
	code, label, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusSubscribed || label != OutletSubscriptionLabelOneMonth {
		t.Fatalf("unexpected status: %s %s", code, label)
	}
	if remaining == nil || *remaining != 0 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}

func TestClassifyOutletSubscription_Subscribed(t *testing.T) {
	referenceMonth := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	snapshot := OutletSubscriptionSnapshot{
		SubscriptionStart: sql.NullTime{Time: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		SubscriptionEnd:   sql.NullTime{Time: time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		TenureMonths:      sql.NullInt64{Int64: 12, Valid: true},
	}
	code, label, remaining, _, _ := classifyOutletSubscription(snapshot, referenceMonth)
	if code != OutletSubscriptionStatusSubscribed || label != OutletSubscriptionStatusSubscribed {
		t.Fatalf("unexpected status: %s %s", code, label)
	}
	if remaining == nil || *remaining != 184 {
		t.Fatalf("unexpected remaining days: %+v", remaining)
	}
}
