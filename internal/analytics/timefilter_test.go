package analytics

import (
	"testing"
	"time"
)

func TestResolveTimeFilterDateRange(t *testing.T) {
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	filter, err := resolveTimeFilter(TimeFilterRequest{
		Mode:        "date_range",
		DateFrom:    "2026-07-01",
		DateTo:      "2026-07-29",
		Granularity: "day",
	}, now)
	if err != nil {
		t.Fatalf("resolveTimeFilter error = %v", err)
	}
	if filter.Start.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("unexpected start %s", filter.Start.Format(time.RFC3339))
	}
	if filter.End.Format("2006-01-02") != "2026-07-30" {
		t.Fatalf("unexpected end %s", filter.End.Format(time.RFC3339))
	}
}

func TestResolveComparisonPreviousMonth(t *testing.T) {
	current := ResolvedTimeFilter{
		Mode:        "month_range",
		Granularity: "month",
		Start:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		End:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Label:       "Jul 2026",
	}
	summary, baseline, err := resolveComparison(current, ComparisonRequest{
		Enabled: true,
		Mode:    "previous_month",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("resolveComparison error = %v", err)
	}
	if !summary.Enabled {
		t.Fatal("expected comparison enabled")
	}
	if baseline == nil {
		t.Fatal("expected baseline")
	}
	if baseline.Start.Format("2006-01-02") != "2026-06-01" {
		t.Fatalf("unexpected baseline start %s", baseline.Start.Format(time.RFC3339))
	}
	if baseline.End.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("unexpected baseline end %s", baseline.End.Format(time.RFC3339))
	}
}

func TestFindDiagram(t *testing.T) {
	item, ok := FindDiagram("owners", "growth-trend")
	if !ok {
		t.Fatal("expected diagram to exist")
	}
	if item.Name == "" || item.QueryEndpoint == "" {
		t.Fatalf("expected metadata to be populated: %+v", item)
	}
}

func TestFindDiagramSprint14g2(t *testing.T) {
	item, ok := FindDiagram("closings", "closing-trend")
	if !ok {
		t.Fatal("expected sprint 14g2 diagram to exist")
	}
	if item.Name == "" || len(item.SupportedMetrics) == 0 {
		t.Fatalf("expected 14g2 metadata to be populated: %+v", item)
	}
}

func TestFindDiagramSprint14g3(t *testing.T) {
	item, ok := FindDiagram("wallets", "topup-revenue-trend")
	if !ok {
		t.Fatal("expected sprint 14g3 diagram to exist")
	}
	if item.Name == "" || item.QueryEndpoint == "" {
		t.Fatalf("expected 14g3 metadata to be populated: %+v", item)
	}
}

func TestFindDiagramSprint14g4(t *testing.T) {
	item, ok := FindDiagram("partners", "partner-growth-trend")
	if !ok {
		t.Fatal("expected sprint 14g4 diagram to exist")
	}
	if item.Name == "" || item.QueryEndpoint == "" {
		t.Fatalf("expected 14g4 metadata to be populated: %+v", item)
	}
}

func TestFindDiagramSprint14g5(t *testing.T) {
	item, ok := FindDiagram("executive", "north-star-kpi-trend")
	if !ok {
		t.Fatal("expected sprint 14g5 diagram to exist")
	}
	if item.Name == "" || len(item.SupportedMetrics) == 0 {
		t.Fatalf("expected 14g5 metadata to be populated: %+v", item)
	}
}
