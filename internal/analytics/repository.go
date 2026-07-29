package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) OwnerGrowthTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.created_at", false)
	return r.queryTrendCount(ctx, "owners", "o", "o.created_at", where, args, tf, "owner_count", "Owner Baru")
}

func (r *Repository) OwnerOwnershipDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.created_at", false)
	query := `
		SELECT cl.current_owner_role AS label, COUNT(DISTINCT o.id) AS total
		FROM owners o
		JOIN customer_leads cl ON cl.owner_id = o.id AND cl.deleted_at IS NULL
		WHERE ` + where + `
		GROUP BY cl.current_owner_role
		ORDER BY total DESC, label ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	series := ChartSeries{Key: "owner_count", Label: "Owner"}
	table := make([]map[string]any, 0)
	var total float64
	for rows.Next() {
		var label string
		var count float64
		if err := rows.Scan(&label, &count); err != nil {
			return queryData{}, err
		}
		series.Points = append(series.Points, ChartPoint{X: label, Y: count})
		table = append(table, map[string]any{"ownership": label, "owner_count": count})
		total += count
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: total}, rows.Err()
}

func (r *Repository) OwnerProvinceDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.created_at", false)
	return r.queryCategoryCount(ctx, `
		SELECT COALESCE(NULLIF(TRIM(o.province), ''), 'Tanpa Provinsi') AS label, COUNT(*) AS total
		FROM owners o
		WHERE `+where+`
		GROUP BY label
		ORDER BY total DESC, label ASC`, args, "province", "owner_count")
}

func (r *Repository) OwnerCityTop10(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.created_at", false)
	query := `
		SELECT COALESCE(NULLIF(TRIM(o.city), ''), 'Tanpa Kota') AS label, COUNT(*) AS total
		FROM owners o
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC
		LIMIT 10`
	return r.queryCategoryCount(ctx, query, args, "city", "owner_count")
}

func (r *Repository) OwnerSoftDeleteTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.deleted_at", true)
	where = strings.Replace(where, "o.deleted_at IS NULL", "o.deleted_at IS NOT NULL", 1)
	return r.queryTrendCount(ctx, "owners", "o", "o.deleted_at", where, args, tf, "soft_deleted_count", "Owner Soft Deleted")
}

func (r *Repository) OutletGrowthTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := outletScopedWhere(actor, req.Filters, "ot", tf, "ot.created_at")
	return r.queryTrendCount(ctx, "outlets", "ot", "ot.created_at", where, args, tf, "outlet_count", "Outlet Baru")
}

func (r *Repository) OutletPerOwnerHistogram(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := outletScopedWhere(actor, req.Filters, "ot", tf, "ot.created_at")
	query := `
		SELECT owner_total
		FROM (
			SELECT COALESCE(ot.owner_id, 0) AS owner_id, COUNT(*) AS owner_total
			FROM outlets ot
			WHERE ` + where + `
			GROUP BY COALESCE(ot.owner_id, 0)
		) summary`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	buckets := map[string]float64{"1": 0, "2": 0, "3-5": 0, "6-10": 0, "11+": 0}
	var total float64
	for rows.Next() {
		var ownerTotal int
		if err := rows.Scan(&ownerTotal); err != nil {
			return queryData{}, err
		}
		total++
		switch {
		case ownerTotal <= 1:
			buckets["1"]++
		case ownerTotal == 2:
			buckets["2"]++
		case ownerTotal <= 5:
			buckets["3-5"]++
		case ownerTotal <= 10:
			buckets["6-10"]++
		default:
			buckets["11+"]++
		}
	}
	labels := []string{"1", "2", "3-5", "6-10", "11+"}
	series := ChartSeries{Key: "owner_bucket_count", Label: "Owner"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"bucket": label, "owner_count": buckets[label]})
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: total}, rows.Err()
}

func (r *Repository) OutletSubscriptionStatusRecap(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	statuses := []string{"NEW", "BERLANGGANAN", "JATUH_TEMPO", "EXPIRED", "NOT_SUBSCRIBE"}
	seriesMap := map[string]*ChartSeries{}
	for _, status := range statuses {
		seriesMap[status] = &ChartSeries{Key: strings.ToLower(status), Label: status}
	}
	table := make([]map[string]any, 0)
	var total float64
	for _, month := range months {
		counts, monthTotal, err := r.outletSubscriptionStatusCounts(ctx, actor, req.Filters, month)
		if err != nil {
			return queryData{}, err
		}
		row := map[string]any{"period": month.Format("2006-01")}
		for _, status := range statuses {
			value := counts[status]
			seriesMap[status].Points = append(seriesMap[status].Points, ChartPoint{X: month.Format("2006-01"), Y: value})
			row[strings.ToLower(status)] = value
		}
		table = append(table, row)
		total += monthTotal
	}
	series := make([]ChartSeries, 0, len(statuses))
	for _, status := range statuses {
		series = append(series, *seriesMap[status])
	}
	return queryData{Series: series, Table: table, Value: total}, nil
}

func (r *Repository) OutletNotSubscribeTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	series := ChartSeries{Key: "not_subscribe_count", Label: "Not Subscribe"}
	table := make([]map[string]any, 0, len(months))
	var total float64
	for _, month := range months {
		counts, _, err := r.outletSubscriptionStatusCounts(ctx, actor, req.Filters, month)
		if err != nil {
			return queryData{}, err
		}
		value := counts["NOT_SUBSCRIBE"]
		series.Points = append(series.Points, ChartPoint{X: month.Format("2006-01"), Y: value})
		table = append(table, map[string]any{"period": month.Format("2006-01"), "not_subscribe_count": value})
		total += value
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: total}, nil
}

func (r *Repository) OwnerDistributionMap(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := ownerScopedWhere(actor, req.Filters, "o", tf, "o.created_at", false)
	query := `
		SELECT COALESCE(NULLIF(TRIM(o.province), ''), 'Tanpa Provinsi') AS province, COUNT(*) AS total
		FROM owners o
		WHERE ` + where + `
		GROUP BY province
		ORDER BY total DESC, province ASC`
	return r.queryRegionMap(ctx, query, args, "owner_count")
}

func (r *Repository) OutletDistributionMap(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := outletScopedWhere(actor, req.Filters, "ot", tf, "ot.created_at")
	query := `
		SELECT COALESCE(NULLIF(TRIM(ot.province), ''), 'Tanpa Provinsi') AS province, COUNT(*) AS total
		FROM outlets ot
		WHERE ` + where + `
		GROUP BY province
		ORDER BY total DESC, province ASC`
	return r.queryRegionMap(ctx, query, args, "outlet_count")
}

func (r *Repository) LeadFunnel(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := leadScopedWhere(actor, req.Filters, "cl", tf, "cl.created_at")
	query := `
		SELECT cl.stage, COUNT(*) AS total
		FROM customer_leads cl
		WHERE ` + where + `
		GROUP BY cl.stage
		ORDER BY FIELD(cl.stage, 'NEW', 'POSSIBLE', 'POTENTIAL', 'CLOSING', 'INVALID'), cl.stage`
	return r.queryCategoryCount(ctx, query, args, "stage", "lead_count")
}

func (r *Repository) LeadAgingByStage(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := leadScopedWhere(actor, req.Filters, "cl", tf, "cl.created_at")
	args = append(args, tf.End)
	query := `
		SELECT cl.stage, AVG(TIMESTAMPDIFF(DAY, cl.created_at, ?)) AS avg_days
		FROM customer_leads cl
		WHERE ` + where + `
		GROUP BY cl.stage
		ORDER BY FIELD(cl.stage, 'NEW', 'POSSIBLE', 'POTENTIAL', 'CLOSING', 'INVALID'), cl.stage`
	return r.queryCategoryCount(ctx, query, args, "stage", "avg_age_days")
}

func (r *Repository) LeadAssignmentDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := leadScopedWhere(actor, req.Filters, "cl", tf, "cl.created_at")
	query := `
		SELECT
			CASE
				WHEN cl.current_owner_role = 'ADMIN' THEN CONCAT('ADMIN - ', COALESCE(u.name, 'Tanpa Nama'))
				WHEN cl.current_owner_role = 'SUPERVISOR' THEN CONCAT('SUPERVISOR - ', COALESCE(u.name, 'Tanpa Nama'))
				WHEN cl.current_owner_role = 'SALES' THEN CONCAT('SALES - ', COALESCE(u.name, 'Tanpa Nama'))
				ELSE cl.current_owner_role
			END AS label,
			COUNT(*) AS total
		FROM customer_leads cl
		LEFT JOIN users u ON u.id = cl.current_owner_user_id
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "current_owner", "lead_count")
}

func (r *Repository) LeadOwnershipTransferSankey(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := leadAssignmentScopedWhere(actor, req.Filters, tf)
	query := `
		SELECT COALESCE(la.from_role, 'UNKNOWN') AS from_role, la.to_role, COUNT(*) AS total
		FROM lead_assignments la
		WHERE ` + where + `
		GROUP BY COALESCE(la.from_role, 'UNKNOWN'), la.to_role
		ORDER BY total DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	nodesSet := map[string]struct{}{}
	links := make([]map[string]any, 0)
	var total float64
	for rows.Next() {
		var fromRole, toRole string
		var count float64
		if err := rows.Scan(&fromRole, &toRole, &count); err != nil {
			return queryData{}, err
		}
		nodesSet[fromRole] = struct{}{}
		nodesSet[toRole] = struct{}{}
		links = append(links, map[string]any{"source": fromRole, "target": toRole, "value": count})
		total += count
	}
	nodes := make([]map[string]any, 0, len(nodesSet))
	for key := range nodesSet {
		nodes = append(nodes, map[string]any{"key": key, "label": key})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i]["key"].(string) < nodes[j]["key"].(string) })
	return queryData{
		Table: links,
		Extra: map[string]any{"nodes": nodes, "links": links},
		Value: total,
	}, rows.Err()
}

func (r *Repository) InteractionVolumeTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := interactionScopedWhere(actor, req.Filters, tf, "ci.interaction_at")
	return r.queryTrendCountWithJoin(ctx, `customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id AND cl.deleted_at IS NULL`, "ci.interaction_at", where, args, tf, "interaction_count", "Interaksi")
}

func (r *Repository) RemarkDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := interactionScopedWhere(actor, req.Filters, tf, "ci.interaction_at")
	query := `
		SELECT COALESCE(CAST(ci.remark_score AS CHAR), 'UNKNOWN') AS label, COUNT(*) AS total
		FROM customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id AND cl.deleted_at IS NULL
		WHERE ` + where + `
		GROUP BY label
		ORDER BY label ASC`
	return r.queryCategoryCount(ctx, query, args, "remark_score", "interaction_count")
}

func (r *Repository) FollowUpCompliance(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := interactionScopedWhere(actor, req.Filters, tf, "ci.follow_up_at")
	where += " AND ci.follow_up_at IS NOT NULL"
	query := `
		SELECT ` + groupExpr("ci.follow_up_at", tf.Granularity) + ` AS period,
			COUNT(*) AS scheduled_count,
			SUM(CASE
				WHEN EXISTS (
					SELECT 1
					FROM customer_interactions ci2
					WHERE ci2.lead_id = ci.lead_id
						AND ci2.deleted_at IS NULL
						AND ci2.interaction_at >= ci.follow_up_at
				) THEN 1 ELSE 0 END
			) AS completed_count
		FROM customer_interactions ci
		LEFT JOIN customer_leads cl ON cl.id = ci.lead_id AND cl.deleted_at IS NULL
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	scheduled := ChartSeries{Key: "scheduled_count", Label: "Follow-up Due"}
	completed := ChartSeries{Key: "completed_count", Label: "Follow-up Selesai"}
	table := make([]map[string]any, 0)
	var totalScheduled, totalCompleted float64
	for rows.Next() {
		var period string
		var dueCount, doneCount float64
		if err := rows.Scan(&period, &dueCount, &doneCount); err != nil {
			return queryData{}, err
		}
		scheduled.Points = append(scheduled.Points, ChartPoint{X: period, Y: dueCount})
		completed.Points = append(completed.Points, ChartPoint{X: period, Y: doneCount})
		table = append(table, map[string]any{"period": period, "scheduled_count": dueCount, "completed_count": doneCount})
		totalScheduled += dueCount
		totalCompleted += doneCount
	}
	value := 0.0
	if totalScheduled > 0 {
		value = (totalCompleted / totalScheduled) * 100
	}
	return queryData{Series: []ChartSeries{scheduled, completed}, Table: table, Value: value}, rows.Err()
}

func (r *Repository) FirstResponseLag(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := leadScopedWhere(actor, req.Filters, "cl", tf, "cl.created_at")
	query := `
		SELECT cl.id, TIMESTAMPDIFF(HOUR, cl.created_at, MIN(ci.interaction_at)) AS lag_hours
		FROM customer_leads cl
		LEFT JOIN customer_interactions ci ON ci.lead_id = cl.id AND ci.deleted_at IS NULL
		WHERE ` + where + `
		GROUP BY cl.id
		HAVING lag_hours IS NOT NULL`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	buckets := map[string]float64{
		"<1h":   0,
		"1-24h": 0,
		"1-3d":  0,
		"4-7d":  0,
		"8-30d": 0,
		"30d+":  0,
	}
	var totalLag float64
	var totalCount float64
	for rows.Next() {
		var leadID int64
		var lagHours float64
		if err := rows.Scan(&leadID, &lagHours); err != nil {
			return queryData{}, err
		}
		totalCount++
		totalLag += lagHours
		switch {
		case lagHours < 1:
			buckets["<1h"]++
		case lagHours <= 24:
			buckets["1-24h"]++
		case lagHours <= 72:
			buckets["1-3d"]++
		case lagHours <= 168:
			buckets["4-7d"]++
		case lagHours <= 720:
			buckets["8-30d"]++
		default:
			buckets["30d+"]++
		}
	}
	labels := []string{"<1h", "1-24h", "1-3d", "4-7d", "8-30d", "30d+"}
	series := ChartSeries{Key: "lead_count", Label: "Lead"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{"bucket": label, "lead_count": buckets[label]})
	}
	value := 0.0
	if totalCount > 0 {
		value = totalLag / totalCount
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: value}, rows.Err()
}

func (r *Repository) TrainingScheduledVsCompleted(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	whereScheduled, argsScheduled := trainingScopedWhere(actor, req.Filters, tf, "tr.scheduled_at")
	queryScheduled := `
		SELECT ` + groupExpr("tr.scheduled_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM training_reports tr
		WHERE ` + whereScheduled + `
		GROUP BY period
		ORDER BY period ASC`
	scheduledPoints, scheduledTable, scheduledTotal, err := r.readTrendRows(ctx, queryScheduled, argsScheduled, "scheduled_count")
	if err != nil {
		return queryData{}, err
	}
	whereCompleted, argsCompleted := trainingScopedWhere(actor, req.Filters, tf, "tr.completed_at")
	whereCompleted += " AND tr.completed_at IS NOT NULL"
	queryCompleted := `
		SELECT ` + groupExpr("tr.completed_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM training_reports tr
		WHERE ` + whereCompleted + `
		GROUP BY period
		ORDER BY period ASC`
	completedPoints, _, completedTotal, err := r.readTrendRows(ctx, queryCompleted, argsCompleted, "completed_count")
	if err != nil {
		return queryData{}, err
	}
	combined := mergeTrendTables(scheduledTable, completedPoints)
	return queryData{
		Series: []ChartSeries{
			{Key: "scheduled_count", Label: "Training Terjadwal", Points: scheduledPoints},
			{Key: "completed_count", Label: "Training Selesai", Points: completedPoints},
		},
		Table: combined,
		Value: completedTotal - scheduledTotal,
	}, nil
}

func (r *Repository) queryTrendCount(ctx context.Context, table, alias, timeColumn, where string, args []any, tf ResolvedTimeFilter, key, label string) (queryData, error) {
	return r.queryTrendCountWithJoin(ctx, table+" "+alias, timeColumn, where, args, tf, key, label)
}

func (r *Repository) queryTrendCountWithJoin(ctx context.Context, fromClause, timeColumn, where string, args []any, tf ResolvedTimeFilter, key, label string) (queryData, error) {
	query := `
		SELECT ` + groupExpr(timeColumn, tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM ` + fromClause + `
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, key)
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{{Key: key, Label: label, Points: points}},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) readTrendRows(ctx context.Context, query string, args []any, valueKey string) ([]ChartPoint, []map[string]any, float64, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, 0, err
	}
	defer rows.Close()
	points := make([]ChartPoint, 0)
	table := make([]map[string]any, 0)
	var total float64
	for rows.Next() {
		var period string
		var count float64
		if err := rows.Scan(&period, &count); err != nil {
			return nil, nil, 0, err
		}
		points = append(points, ChartPoint{X: period, Y: count})
		table = append(table, map[string]any{"period": period, valueKey: count})
		total += count
	}
	return points, table, total, rows.Err()
}

func (r *Repository) queryCategoryCount(ctx context.Context, query string, args []any, dimensionKey, metricKey string) (queryData, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	series := ChartSeries{Key: metricKey, Label: strings.ReplaceAll(strings.Title(strings.ReplaceAll(metricKey, "_", " ")), "Id", "ID")}
	table := make([]map[string]any, 0)
	var total float64
	for rows.Next() {
		var label string
		var value float64
		if err := rows.Scan(&label, &value); err != nil {
			return queryData{}, err
		}
		series.Points = append(series.Points, ChartPoint{X: label, Y: value})
		table = append(table, map[string]any{dimensionKey: label, metricKey: value})
		total += value
	}
	return queryData{Series: []ChartSeries{series}, Table: table, Value: total}, rows.Err()
}

func (r *Repository) queryRegionMap(ctx context.Context, query string, args []any, valueKey string) (queryData, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	regions := make([]map[string]any, 0)
	var total float64
	for rows.Next() {
		var province string
		var count float64
		if err := rows.Scan(&province, &count); err != nil {
			return queryData{}, err
		}
		regions = append(regions, map[string]any{
			"province_code": slugProvinceCode(province),
			"province_name": province,
			valueKey:        count,
		})
		total += count
	}
	return queryData{
		Table: regions,
		Extra: map[string]any{"regions": regions},
		Value: total,
	}, rows.Err()
}

func ownerScopedWhere(actor identity.User, filters FilterRequest, alias string, tf ResolvedTimeFilter, timeColumn string, includeDeleted bool) (string, []any) {
	where := []string{}
	args := []any{}
	if includeDeleted {
		where = append(where, alias+".deleted_at IS NOT NULL")
	} else {
		where = append(where, alias+".deleted_at IS NULL")
	}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, alias+".id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, alias+".province", filters.Province)
	appendStringFilter(&where, &args, alias+".city", filters.City)
	appendIntFilter(&where, &args, alias+".id", filters.OwnerIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, alias+".id")
	return strings.Join(where, " AND "), args
}

func outletScopedWhere(actor identity.User, filters FilterRequest, alias string, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{alias + ".deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, alias+".owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, alias+".province", filters.Province)
	appendStringFilter(&where, &args, alias+".city", filters.City)
	appendIntFilter(&where, &args, alias+".owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, alias+".id", filters.OutletIDs)
	appendOwnerLeadScopedFilters(&where, &args, filters, alias+".owner_id")
	return strings.Join(where, " AND "), args
}

func leadScopedWhere(actor identity.User, filters FilterRequest, alias string, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{alias + ".deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := leadVisibilityWhere(actor, alias)
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, alias+".stage", filters.Status)
	appendIntFilter(&where, &args, alias+".owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, alias+".outlet_id", filters.OutletIDs)
	appendIntFilter(&where, &args, alias+".supervisor_id", filters.SupervisorIDs)
	appendIntFilter(&where, &args, alias+".active_sales_id", filters.SalesIDs)
	return strings.Join(where, " AND "), args
}

func interactionScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"ci.deleted_at IS NULL"}
	args := []any{}
	switch actor.RoleCode {
	case "ADMIN":
		where = append(where, "1 = 1")
	case "SUPERVISOR":
		where = append(where, "(ci.supervisor_id = ? OR cl.current_owner_user_id = ? OR cl.supervisor_id = ?)")
		args = append(args, actor.ID, actor.ID, actor.ID)
	case "SALES":
		where = append(where, "(ci.sales_id = ? OR (cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?))")
		args = append(args, actor.ID, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendIntFilter(&where, &args, "ci.sales_id", filters.SalesIDs)
	appendIntFilter(&where, &args, "ci.supervisor_id", filters.SupervisorIDs)
	appendIntFilter(&where, &args, "ci.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "ci.outlet_id", filters.OutletIDs)
	return strings.Join(where, " AND "), args
}

func trainingScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"tr.deleted_at IS NULL"}
	args := []any{}
	switch actor.RoleCode {
	case "ADMIN":
		where = append(where, "1 = 1")
	case "SUPERVISOR":
		where = append(where, "tr.supervisor_id = ?")
		args = append(args, actor.ID)
	case "SALES":
		where = append(where, "tr.sales_id = ?")
		args = append(args, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendIntFilter(&where, &args, "tr.sales_id", filters.SalesIDs)
	appendIntFilter(&where, &args, "tr.supervisor_id", filters.SupervisorIDs)
	appendIntFilter(&where, &args, "tr.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "tr.outlet_id", filters.OutletIDs)
	appendStringFilter(&where, &args, "tr.status", filters.Status)
	return strings.Join(where, " AND "), args
}

func leadAssignmentScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (string, []any) {
	where := []string{"la.deleted_at IS NULL", "la.created_at >= ?", "la.created_at < ?"}
	args := []any{tf.Start, tf.End}
	switch actor.RoleCode {
	case "ADMIN":
		where = append(where, "1 = 1")
	case "SUPERVISOR":
		where = append(where, "(la.supervisor_id = ? OR EXISTS (SELECT 1 FROM customer_leads cl WHERE cl.id = la.lead_id AND cl.deleted_at IS NULL AND (cl.current_owner_user_id = ? OR cl.supervisor_id = ?)))")
		args = append(args, actor.ID, actor.ID, actor.ID)
	case "SALES":
		where = append(where, "EXISTS (SELECT 1 FROM customer_leads cl WHERE cl.id = la.lead_id AND cl.deleted_at IS NULL AND cl.current_owner_role = 'SALES' AND cl.current_owner_user_id = ?)")
		args = append(args, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	appendIntFilter(&where, &args, "la.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "la.supervisor_id", filters.SupervisorIDs)
	return strings.Join(where, " AND "), args
}

func appendStringFilter(where *[]string, args *[]any, column string, values []string) {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			clean = append(clean, strings.TrimSpace(value))
		}
	}
	if len(clean) == 0 {
		return
	}
	*where = append(*where, column+" IN ("+placeholders(len(clean))+")")
	for _, item := range clean {
		*args = append(*args, item)
	}
}

func appendIntFilter(where *[]string, args *[]any, column string, values []int64) {
	if len(values) == 0 {
		return
	}
	*where = append(*where, column+" IN ("+placeholders(len(values))+")")
	for _, item := range values {
		*args = append(*args, item)
	}
}

func appendOwnerLeadScopedFilters(where *[]string, args *[]any, filters FilterRequest, ownerColumn string) {
	if len(filters.SupervisorIDs) > 0 {
		*where = append(*where, `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = `+ownerColumn+`
				AND cl.deleted_at IS NULL
				AND cl.supervisor_id IN (`+placeholders(len(filters.SupervisorIDs))+`)
		)`)
		for _, value := range filters.SupervisorIDs {
			*args = append(*args, value)
		}
	}
	if len(filters.SalesIDs) > 0 {
		*where = append(*where, `EXISTS (
			SELECT 1 FROM customer_leads cl
			WHERE cl.owner_id = `+ownerColumn+`
				AND cl.deleted_at IS NULL
				AND cl.active_sales_id IN (`+placeholders(len(filters.SalesIDs))+`)
		)`)
		for _, value := range filters.SalesIDs {
			*args = append(*args, value)
		}
	}
}

func monthRange(tf ResolvedTimeFilter) []time.Time {
	start := time.Date(tf.Start.Year(), tf.Start.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(tf.End.Add(-time.Nanosecond).Year(), tf.End.Add(-time.Nanosecond).Month(), 1, 0, 0, 0, 0, time.UTC)
	items := make([]time.Time, 0)
	for current := start; !current.After(end); current = current.AddDate(0, 1, 0) {
		items = append(items, current)
	}
	return items
}

func (r *Repository) outletSubscriptionStatusCounts(ctx context.Context, actor identity.User, filters FilterRequest, month time.Time) (map[string]float64, float64, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	threshold := monthEnd.AddDate(0, 0, -60)
	recentStart := monthEnd.AddDate(0, 0, -30)
	year := month.Year()
	monthNum := int(month.Month())

	where := []string{"ot.deleted_at IS NULL"}
	args := []any{}
	visibility, visibilityArgs := ownerVisibilityWhere(actor, "ot.owner_id")
	where = append(where, visibility)
	args = append(args, visibilityArgs...)
	appendStringFilter(&where, &args, "ot.province", filters.Province)
	appendStringFilter(&where, &args, "ot.city", filters.City)
	appendIntFilter(&where, &args, "ot.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "ot.id", filters.OutletIDs)

	query := `
		WITH latest_subscriptions AS (
			SELECT
				s.outlet_id,
				s.active_from,
				s.active_until,
				spl.tenure_months,
				ROW_NUMBER() OVER (PARTITION BY s.outlet_id ORDER BY s.active_from DESC, s.id DESC) AS rn
			FROM subscriptions s
			LEFT JOIN subscription_plans spl ON spl.id = s.plan_id
			WHERE s.deleted_at IS NULL AND s.active_from <= ?
		)
		SELECT
			CASE
				WHEN ls.active_until IS NULL OR ls.active_from IS NULL OR ls.active_until < ? THEN 'NOT_SUBSCRIBE'
				WHEN ls.active_until >= ? AND ls.active_until < ? THEN 'EXPIRED'
				WHEN ls.active_from IS NOT NULL AND ls.active_until IS NOT NULL AND ls.active_until >= ? AND ls.active_from >= ? AND NOT (COALESCE(ls.tenure_months, 0) = 1 AND YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ?) THEN 'NEW'
				WHEN ls.active_until IS NOT NULL AND YEAR(ls.active_until) = ? AND MONTH(ls.active_until) = ? AND COALESCE(ls.tenure_months, 0) <> 1 THEN 'JATUH_TEMPO'
				ELSE 'BERLANGGANAN'
			END AS status_code,
			COUNT(*) AS total
		FROM outlets ot
		LEFT JOIN latest_subscriptions ls ON ls.outlet_id = ot.id AND ls.rn = 1
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY status_code`
	fullArgs := []any{monthEnd}
	fullArgs = append(fullArgs, threshold, threshold, monthStart, monthStart, recentStart, year, monthNum, year, monthNum)
	fullArgs = append(fullArgs, args...)
	rows, err := r.db.QueryContext(ctx, query, fullArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	counts := map[string]float64{
		"NEW":           0,
		"BERLANGGANAN":  0,
		"JATUH_TEMPO":   0,
		"EXPIRED":       0,
		"NOT_SUBSCRIBE": 0,
	}
	total := 0.0
	for rows.Next() {
		var status string
		var count float64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, 0, err
		}
		counts[status] = count
		total += count
	}
	return counts, total, rows.Err()
}

func mergeTrendTables(primary []map[string]any, comparison []ChartPoint) []map[string]any {
	lookup := map[string]float64{}
	for _, point := range comparison {
		lookup[fmt.Sprint(point.X)] = point.Y
	}
	for _, row := range primary {
		key := fmt.Sprint(row["period"])
		row["completed_count"] = lookup[key]
	}
	return primary
}

func slugProvinceCode(province string) string {
	normalized := strings.ToUpper(strings.TrimSpace(province))
	if normalized == "" || normalized == "TANPA PROVINSI" {
		return "ID-UNKNOWN"
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '-' || r == '/'
	})
	code := "ID-"
	for _, part := range parts {
		if part == "" {
			continue
		}
		code += string(part[0])
	}
	return code
}

func parseFloatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}
