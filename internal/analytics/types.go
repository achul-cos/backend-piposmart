package analytics

import "time"

type DiagramCatalogItem struct {
	Module             string   `json:"module"`
	Key                string   `json:"key"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Function           string   `json:"function"`
	Purpose            string   `json:"purpose"`
	HowToRead          string   `json:"how_to_read"`
	AnalysisGoal       string   `json:"analysis_goal"`
	SupportedMetrics   []string `json:"supported_metrics,omitempty"`
	SupportedFilters   []string `json:"supported_filters,omitempty"`
	SupportedCompare   []string `json:"supported_compare_modes,omitempty"`
	QueryEndpoint      string   `json:"query_endpoint"`
	ExportEndpoint     string   `json:"export_endpoint,omitempty"`
	PolarityRule       string   `json:"polarity_rule"`
	ExportAvailable    bool     `json:"export_available"`
	ComparisonEnabled  bool     `json:"comparison_enabled"`
}

type TimeFilterRequest struct {
	Mode        string `json:"mode"`
	DateFrom    string `json:"date_from"`
	DateTo      string `json:"date_to"`
	MonthFrom   string `json:"month_from"`
	MonthTo     string `json:"month_to"`
	YearFrom    *int   `json:"year_from"`
	YearTo      *int   `json:"year_to"`
	Granularity string `json:"granularity"`
}

type ComparisonSeriesRequest struct {
	Field string `json:"field"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type ComparisonRequest struct {
	Enabled            bool                      `json:"enabled"`
	Mode               string                    `json:"mode"`
	BaselineTimeFilter *TimeFilterRequest        `json:"baseline_time_filter"`
	CompareSeries      []ComparisonSeriesRequest `json:"compare_series"`
}

type FilterRequest struct {
	Province     []string `json:"province"`
	City         []string `json:"city"`
	SalesIDs     []int64  `json:"sales_id"`
	SupervisorIDs []int64 `json:"supervisor_id"`
	OwnerIDs     []int64  `json:"owner_id"`
	OutletIDs    []int64  `json:"outlet_id"`
	Status       []string `json:"status"`
}

type QueryOptions struct {
	Limit                 int    `json:"limit"`
	Sort                  string `json:"sort"`
	IncludeTable          bool   `json:"include_table"`
	IncludeSummary        bool   `json:"include_summary"`
	IncludePreviousPoints bool   `json:"include_previous_points"`
}

type QueryRequest struct {
	TimeFilter TimeFilterRequest `json:"time_filter"`
	Comparison ComparisonRequest `json:"comparison"`
	Metrics    []string          `json:"metrics"`
	Dimensions []string          `json:"dimensions"`
	Filters    FilterRequest     `json:"filters"`
	Options    QueryOptions      `json:"options"`
}

type ResolvedTimeFilter struct {
	Mode        string    `json:"mode"`
	Granularity string    `json:"granularity"`
	Start       time.Time `json:"-"`
	End         time.Time `json:"-"`
	Label       string    `json:"label"`
}

type DiagramMetadata struct {
	Key          string `json:"key"`
	Module       string `json:"module"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Function     string `json:"function"`
	Purpose      string `json:"purpose"`
	HowToRead    string `json:"how_to_read"`
	AnalysisGoal string `json:"analysis_goal"`
}

type ComparisonSummary struct {
	Enabled       bool    `json:"enabled"`
	Mode          string  `json:"mode,omitempty"`
	BaselineLabel string  `json:"baseline_label,omitempty"`
	CurrentValue  float64 `json:"current_value,omitempty"`
	BaselineValue float64 `json:"baseline_value,omitempty"`
	Delta         float64 `json:"delta,omitempty"`
	DeltaPercent  float64 `json:"delta_percent,omitempty"`
	Direction     string  `json:"direction,omitempty"`
	PolarityRule  string  `json:"polarity_rule,omitempty"`
	StatusValue   int     `json:"status_value,omitempty"`
}

type ChartPoint struct {
	X any     `json:"x"`
	Y float64 `json:"y"`
}

type ChartSeries struct {
	Key    string       `json:"key"`
	Label  string       `json:"label"`
	Points []ChartPoint `json:"points"`
}

type Insight struct {
	Summary        string `json:"summary,omitempty"`
	Conclusion     string `json:"conclusion,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type QueryResult struct {
	Diagram    DiagramMetadata        `json:"diagram"`
	TimeFilter map[string]any         `json:"time_filter"`
	Comparison ComparisonSummary      `json:"comparison"`
	Series     []ChartSeries          `json:"series,omitempty"`
	Table      []map[string]any       `json:"table,omitempty"`
	Extra      map[string]any         `json:"extra,omitempty"`
	Insight    Insight                `json:"insight,omitempty"`
}

type queryData struct {
	Series []ChartSeries
	Table  []map[string]any
	Extra  map[string]any
	Value  float64
}
