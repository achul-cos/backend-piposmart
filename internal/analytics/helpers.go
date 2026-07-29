package analytics

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

func groupExpr(column, granularity string) string {
	switch granularity {
	case "week":
		return fmt.Sprintf("DATE_FORMAT(DATE_SUB(DATE(%s), INTERVAL WEEKDAY(%s) DAY), '%%Y-%%m-%%d')", column, column)
	case "month":
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m')", column)
	case "quarter":
		return fmt.Sprintf("CONCAT(YEAR(%s), '-Q', QUARTER(%s))", column, column)
	case "year":
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y')", column)
	default:
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", column)
	}
}

func ownerVisibilityWhere(actor identity.User, ownerColumn string) (string, []any) {
	switch actor.RoleCode {
	case "ADMIN":
		return "1 = 1", nil
	case "SUPERVISOR":
		return `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?)
		)`, []any{actor.ID, actor.ID}
	case "SALES":
		return `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = ` + ownerColumn + `
				AND cl.deleted_at IS NULL
				AND cl.current_owner_role = 'SALES'
				AND cl.current_owner_user_id = ?
		)`, []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func leadVisibilityWhere(actor identity.User, leadAlias string) (string, []any) {
	switch actor.RoleCode {
	case "ADMIN":
		return "1 = 1", nil
	case "SUPERVISOR":
		return fmt.Sprintf("(%s.current_owner_user_id = ? OR %s.supervisor_id = ?)", leadAlias, leadAlias), []any{actor.ID, actor.ID}
	case "SALES":
		return fmt.Sprintf("(%s.current_owner_role = 'SALES' AND %s.current_owner_user_id = ?)", leadAlias, leadAlias), []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func activityVisibilityWhere(actor identity.User, activityAlias string) (string, []any) {
	switch actor.RoleCode {
	case "ADMIN":
		return "1 = 1", nil
	case "SUPERVISOR":
		return fmt.Sprintf("(%s.supervisor_id = ? OR %s.sales_id IN (SELECT id FROM users WHERE deleted_at IS NULL))", activityAlias, activityAlias), []any{actor.ID}
	case "SALES":
		return fmt.Sprintf("%s.sales_id = ?", activityAlias), []any{actor.ID}
	default:
		return "1 = 0", nil
	}
}

func like(value string) string {
	return "%" + strings.TrimSpace(value) + "%"
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	values := make([]string, count)
	for i := 0; i < count; i++ {
		values[i] = "?"
	}
	return strings.Join(values, ", ")
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func buildComparison(summary ComparisonSummary, currentValue, baselineValue float64, polarity string) ComparisonSummary {
	summary.CurrentValue = round2(currentValue)
	summary.BaselineValue = round2(baselineValue)
	summary.Delta = round2(currentValue - baselineValue)
	if baselineValue != 0 {
		summary.DeltaPercent = round2(((currentValue - baselineValue) / baselineValue) * 100)
	}
	summary.PolarityRule = polarity
	switch polarity {
	case "lower_is_better":
		if currentValue < baselineValue {
			summary.Direction = "positive"
			summary.StatusValue = 1
		} else if currentValue > baselineValue {
			summary.Direction = "negative"
			summary.StatusValue = -1
		} else {
			summary.Direction = "neutral"
		}
	case "balanced_is_better":
		if summary.Delta == 0 {
			summary.Direction = "neutral"
		} else if math.Abs(summary.DeltaPercent) <= 5 {
			summary.Direction = "positive"
			summary.StatusValue = 1
		} else {
			summary.Direction = "negative"
			summary.StatusValue = -1
		}
	default:
		if currentValue > baselineValue {
			summary.Direction = "positive"
			summary.StatusValue = 1
		} else if currentValue < baselineValue {
			summary.Direction = "negative"
			summary.StatusValue = -1
		} else {
			summary.Direction = "neutral"
		}
	}
	return summary
}

func buildInsight(item DiagramCatalogItem, cmp ComparisonSummary) Insight {
	if !cmp.Enabled {
		return Insight{
			Summary:        fmt.Sprintf("%s pada periode terpilih bernilai %.2f.", item.Name, cmp.CurrentValue),
			Conclusion:     item.AnalysisGoal,
			Recommendation: "Gunakan perbandingan periode untuk melihat arah perubahan performa.",
		}
	}
	if cmp.Mode == "series_to_series" {
		return Insight{
			Summary:        fmt.Sprintf("%s membandingkan seri terpilih pada periode yang sama.", item.Name),
			Conclusion:     item.AnalysisGoal,
			Recommendation: "Bandingkan nilai antar seri pada tabel dan chart untuk menentukan area paling kuat atau paling lemah.",
		}
	}
	directionWord := "stabil"
	recommendation := "Pertahankan monitoring berkala pada diagram ini."
	switch cmp.Direction {
	case "positive":
		directionWord = "membaik"
		recommendation = "Pertahankan strategi yang berkontribusi pada perbaikan metrik ini."
	case "negative":
		directionWord = "memburuk"
		recommendation = "Lakukan investigasi pada faktor yang paling berkontribusi terhadap penurunan metrik ini."
	}
	return Insight{
		Summary:        fmt.Sprintf("%s %s %.2f%% dibanding baseline.", item.Name, directionWord, math.Abs(cmp.DeltaPercent)),
		Conclusion:     fmt.Sprintf("%s Nilai saat ini %.2f dibanding baseline %.2f.", item.AnalysisGoal, cmp.CurrentValue, cmp.BaselineValue),
		Recommendation: recommendation,
	}
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func regionStatus(delta float64, polarity string) (string, int) {
	switch polarity {
	case "lower_is_better":
		if delta < 0 {
			return "positive", 1
		}
		if delta > 0 {
			return "negative", -1
		}
	default:
		if delta > 0 {
			return "positive", 1
		}
		if delta < 0 {
			return "negative", -1
		}
	}
	return "neutral", 0
}

func copyTimeFilterForResponse(value ResolvedTimeFilter) map[string]any {
	return map[string]any{
		"mode":        value.Mode,
		"granularity": value.Granularity,
		"label":       value.Label,
	}
}

func currentNowUTC() time.Time {
	return time.Now().UTC()
}
