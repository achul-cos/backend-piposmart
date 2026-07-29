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

const closingRevenueExpr = "(sc.base_price - sc.discount_amount + sc.additional_charge)"

type analyticsSalesInfo struct {
	ID   int64
	Name string
}

func (r *Repository) TrainingToClosingConversion(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	whereTraining, argsTraining := trainingScopedWhere(actor, req.Filters, tf, "tr.completed_at")
	whereTraining += " AND tr.status = 'COMPLETED' AND tr.completed_at IS NOT NULL"
	queryTraining := `
		SELECT ` + groupExpr("tr.completed_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM training_reports tr
		WHERE ` + whereTraining + `
		GROUP BY period
		ORDER BY period ASC`
	trainingPoints, trainingTable, trainingTotal, err := r.readTrendRows(ctx, queryTraining, argsTraining, "completed_training_count")
	if err != nil {
		return queryData{}, err
	}

	whereClosing, argsClosing := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	whereClosing += ` AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL
		AND EXISTS (
			SELECT 1
			FROM training_reports tr
			WHERE tr.deleted_at IS NULL
				AND tr.status = 'COMPLETED'
				AND tr.completed_at IS NOT NULL
				AND tr.completed_at <= sc.confirmed_at
				AND (
					(tr.lead_id IS NOT NULL AND tr.lead_id = sc.lead_id)
					OR (tr.lead_id IS NULL AND tr.owner_id = sc.owner_id AND (tr.outlet_id <=> sc.outlet_id))
				)
		)`
	queryClosing := `
		SELECT ` + groupExpr("sc.confirmed_at", tf.Granularity) + ` AS period, COUNT(*) AS total
		FROM sales_closings sc
		WHERE ` + whereClosing + `
		GROUP BY period
		ORDER BY period ASC`
	closingPoints, _, closingTotal, err := r.readTrendRows(ctx, queryClosing, argsClosing, "converted_closing_count")
	if err != nil {
		return queryData{}, err
	}

	lookup := map[string]float64{}
	for _, point := range closingPoints {
		lookup[fmt.Sprint(point.X)] = point.Y
	}

	conversionSeries := ChartSeries{Key: "conversion_rate", Label: "Conversion Rate (%)"}
	for i := range trainingTable {
		period := fmt.Sprint(trainingTable[i]["period"])
		converted := lookup[period]
		trainingTable[i]["converted_closing_count"] = converted
		trainingCount := parseFloatFromAny(trainingTable[i]["completed_training_count"])
		rate := 0.0
		if trainingCount > 0 {
			rate = round2((converted / trainingCount) * 100)
		}
		trainingTable[i]["conversion_rate"] = rate
		conversionSeries.Points = append(conversionSeries.Points, ChartPoint{X: period, Y: rate})
	}

	value := 0.0
	if trainingTotal > 0 {
		value = round2((closingTotal / trainingTotal) * 100)
	}

	return queryData{
		Series: []ChartSeries{
			{Key: "completed_training_count", Label: "Training Selesai", Points: trainingPoints},
			{Key: "converted_closing_count", Label: "Closing dari Training", Points: closingPoints},
			conversionSeries,
		},
		Table: trainingTable,
		Extra: map[string]any{
			"completed_training_total": trainingTotal,
			"converted_closing_total":  closingTotal,
			"conversion_rate":          value,
		},
		Value: value,
	}, nil
}

func (r *Repository) PackagePopularity(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')), ''), COALESCE(sp.name, 'Tanpa Paket')) AS label,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		LEFT JOIN subscription_packages sp ON sp.id = sc.package_id
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	result, err := r.queryCategoryAggregate(ctx, query, args, "package_name", metric)
	if err != nil {
		return queryData{}, err
	}
	result.Extra = map[string]any{"metric": metric}
	return result, nil
}

func (r *Repository) TenurePopularity(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT CONCAT(sc.tenure_months, ' bulan') AS label,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY sc.tenure_months
		ORDER BY sc.tenure_months ASC`
	result, err := r.queryCategoryAggregate(ctx, query, args, "tenure_label", metric)
	if err != nil {
		return queryData{}, err
	}
	result.Extra = map[string]any{"metric": metric}
	return result, nil
}

func (r *Repository) PackageTenureHeatmap(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT
			COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(sc.package_snapshot_json, '$.name')), ''), COALESCE(sp.name, 'Tanpa Paket')) AS package_name,
			sc.tenure_months,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		LEFT JOIN subscription_packages sp ON sp.id = sc.package_id
		WHERE ` + where + `
		GROUP BY package_name, sc.tenure_months
		ORDER BY package_name ASC, sc.tenure_months ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	type cell struct {
		PackageName string
		Tenure      int
		Total       float64
	}
	cells := make([]cell, 0)
	seriesMap := map[string]*ChartSeries{}
	table := make([]map[string]any, 0)
	totalValue := 0.0
	for rows.Next() {
		var packageName string
		var tenure int
		var total float64
		if err := rows.Scan(&packageName, &tenure, &total); err != nil {
			return queryData{}, err
		}
		cells = append(cells, cell{PackageName: packageName, Tenure: tenure, Total: total})
		if _, ok := seriesMap[packageName]; !ok {
			seriesMap[packageName] = &ChartSeries{Key: slugifyKey(packageName), Label: packageName}
		}
		seriesMap[packageName].Points = append(seriesMap[packageName].Points, ChartPoint{
			X: fmt.Sprintf("%d bulan", tenure),
			Y: total,
		})
		table = append(table, map[string]any{
			"package_name": packageName,
			"tenure_months": tenure,
			"tenure_label": fmt.Sprintf("%d bulan", tenure),
			"value": total,
			"metric": metric,
		})
		totalValue += total
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}

	keys := make([]string, 0, len(seriesMap))
	for key := range seriesMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	series := make([]ChartSeries, 0, len(keys))
	for _, key := range keys {
		series = append(series, *seriesMap[key])
	}

	heatmapCells := make([]map[string]any, 0, len(cells))
	for _, item := range cells {
		heatmapCells = append(heatmapCells, map[string]any{
			"package_name":  item.PackageName,
			"tenure_months": item.Tenure,
			"tenure_label":  fmt.Sprintf("%d bulan", item.Tenure),
			"value":         item.Total,
		})
	}

	return queryData{
		Series: series,
		Table:  table,
		Extra: map[string]any{
			"cells":  heatmapCells,
			"metric": metric,
		},
		Value: totalValue,
	}, nil
}

func (r *Repository) PromoAdoptionRate(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT
			CASE
				WHEN sc.promotion_id IS NULL THEN 'NO_PROMO'
				WHEN sc.additional_charge > 0 THEN 'PAID_PROMO'
				ELSE 'FREE_PROMO'
			END AS label,
			COUNT(*) AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "promo_type", "closing_count")
}

func (r *Repository) AdditionalChargeAdoption(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT
			CASE WHEN sc.additional_charge > 0 THEN 'WITH_ADDITIONAL_CHARGE' ELSE 'WITHOUT_ADDITIONAL_CHARGE' END AS label,
			COUNT(*) AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "additional_charge_category", "closing_count")
}

func (r *Repository) PriceHistoryTimeline(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT
			DATE_FORMAT(a.created_at, '%Y-%m-%d') AS period,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.code')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.code')), CONCAT('PLAN-', a.entity_id)) AS plan_code,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.name')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.name')), 'Tanpa Nama') AS plan_name,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.package.name')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.package.name')), 'Tanpa Paket') AS package_name,
			CAST(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.price')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.price')), '0') AS DOUBLE) AS price_value,
			a.action,
			COALESCE(u.name, 'System') AS actor_name,
			a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.entity_type = 'catalog.plan'
			AND a.created_at >= ? AND a.created_at < ?
		ORDER BY a.created_at ASC, a.id ASC`
	rows, err := r.db.QueryContext(ctx, query, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	seriesMap := map[string]*ChartSeries{}
	table := make([]map[string]any, 0)
	totalChanges := 0.0
	for rows.Next() {
		var period, planCode, planName, packageName, action, actorName string
		var priceValue float64
		var createdAt time.Time
		if err := rows.Scan(&period, &planCode, &planName, &packageName, &priceValue, &action, &actorName, &createdAt); err != nil {
			return queryData{}, err
		}
		key := planCode
		if _, ok := seriesMap[key]; !ok {
			seriesMap[key] = &ChartSeries{Key: slugifyKey(planCode), Label: fmt.Sprintf("%s - %s", packageName, planName)}
		}
		seriesMap[key].Points = append(seriesMap[key].Points, ChartPoint{X: period, Y: priceValue})
		table = append(table, map[string]any{
			"period":       period,
			"plan_code":    planCode,
			"plan_name":    planName,
			"package_name": packageName,
			"price_value":  priceValue,
			"action":       action,
			"actor_name":   actorName,
			"changed_at":   createdAt,
		})
		totalChanges++
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: sortSeriesMap(seriesMap),
		Table:  table,
		Extra:  map[string]any{"change_count": totalChanges},
		Value:  totalChanges,
	}, nil
}

func (r *Repository) PromotionHistoryTimeline(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	if actor.RoleCode == identity.RoleSales {
		return queryData{Series: []ChartSeries{}, Table: []map[string]any{}, Value: 0}, nil
	}
	query := `
		SELECT
			DATE_FORMAT(a.created_at, '%Y-%m-%d') AS period,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.code')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.code')), CONCAT('PROMO-', a.entity_id)) AS promotion_code,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.name')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.name')), 'Tanpa Nama') AS promotion_name,
			COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.charge_type')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.charge_type')), 'FREE') AS charge_type,
			CAST(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(a.after_json, '$.additional_charge')), JSON_UNQUOTE(JSON_EXTRACT(a.before_json, '$.additional_charge')), '0') AS DOUBLE) AS additional_charge_value,
			a.action,
			COALESCE(u.name, 'System') AS actor_name,
			a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE a.entity_type = 'catalog.promotion'
			AND a.created_at >= ? AND a.created_at < ?
		ORDER BY a.created_at ASC, a.id ASC`
	rows, err := r.db.QueryContext(ctx, query, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	seriesMap := map[string]*ChartSeries{}
	table := make([]map[string]any, 0)
	totalChanges := 0.0
	for rows.Next() {
		var period, promotionCode, promotionName, chargeType, action, actorName string
		var additionalCharge float64
		var createdAt time.Time
		if err := rows.Scan(&period, &promotionCode, &promotionName, &chargeType, &additionalCharge, &action, &actorName, &createdAt); err != nil {
			return queryData{}, err
		}
		if _, ok := seriesMap[promotionCode]; !ok {
			seriesMap[promotionCode] = &ChartSeries{Key: slugifyKey(promotionCode), Label: promotionName}
		}
		seriesMap[promotionCode].Points = append(seriesMap[promotionCode].Points, ChartPoint{X: period, Y: additionalCharge})
		table = append(table, map[string]any{
			"period":                  period,
			"promotion_code":          promotionCode,
			"promotion_name":          promotionName,
			"charge_type":             chargeType,
			"additional_charge_value": additionalCharge,
			"action":                  action,
			"actor_name":              actorName,
			"changed_at":              createdAt,
		})
		totalChanges++
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: sortSeriesMap(seriesMap),
		Table:  table,
		Extra:  map[string]any{"change_count": totalChanges},
		Value:  totalChanges,
	}, nil
}

func (r *Repository) ClosingTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	timeColumn := "sc.confirmed_at"
	where, args := closingScopedWhere(actor, req.Filters, tf, timeColumn)
	switch metric {
	case "closing_count":
		timeColumn = "sc.closed_at"
		where, args = closingScopedWhere(actor, req.Filters, tf, timeColumn)
	case "confirmed_closing_count", "gross_revenue_snapshot", "average_ticket_size":
		where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	}
	query := `
		SELECT ` + groupExpr(timeColumn, tf.Granularity) + ` AS period,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, metric)
	if err != nil {
		return queryData{}, err
	}
	if metric == "average_ticket_size" && len(points) > 0 {
		total = 0
		for _, point := range points {
			total += point.Y
		}
		total = round2(total / float64(len(points)))
	}
	return queryData{
		Series: []ChartSeries{{Key: metric, Label: closingMetricLabel(metric), Points: points}},
		Table:  table,
		Extra:  map[string]any{"metric": metric},
		Value:  total,
	}, nil
}

func (r *Repository) ClosingBySales(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	if metric != "closing_count" {
		where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	}
	query := `
		SELECT COALESCE(u.name, 'Tanpa Sales') AS label,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		LEFT JOIN users u ON u.id = sc.sales_id
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	result, err := r.queryCategoryAggregate(ctx, query, args, "sales_name", metric)
	if err != nil {
		return queryData{}, err
	}
	result.Extra = map[string]any{"metric": metric}
	return result, nil
}

func (r *Repository) ClosingBySupervisor(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metric := selectedClosingMetric(req.Metrics, "confirmed_closing_count")
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	if metric != "closing_count" {
		where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	}
	query := `
		SELECT COALESCE(u.name, 'Tanpa Supervisor') AS label,
			` + closingAggregateExpr(metric) + ` AS total
		FROM sales_closings sc
		LEFT JOIN users u ON u.id = sc.supervisor_id
		WHERE ` + where + `
		GROUP BY label
		ORDER BY total DESC, label ASC`
	result, err := r.queryCategoryAggregate(ctx, query, args, "supervisor_name", metric)
	if err != nil {
		return queryData{}, err
	}
	result.Extra = map[string]any{"metric": metric}
	return result, nil
}

func (r *Repository) ClosingByPackage(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	return r.PackagePopularity(ctx, actor, req, tf)
}

func (r *Repository) ClosingByTenure(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	return r.TenurePopularity(ctx, actor, req, tf)
}

func (r *Repository) ClosingStatusDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.closed_at")
	query := `
		SELECT sc.status AS label, COUNT(*) AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY sc.status
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "status", "closing_count")
}

func (r *Repository) AverageTicketSizeTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT ` + groupExpr("sc.confirmed_at", tf.Granularity) + ` AS period,
			CAST(AVG(` + closingRevenueExpr + `) AS DOUBLE) AS total
		FROM sales_closings sc
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, _, err := r.readTrendRows(ctx, query, args, "average_ticket_size")
	if err != nil {
		return queryData{}, err
	}
	value := 0.0
	if len(points) > 0 {
		for _, point := range points {
			value += point.Y
		}
		value = round2(value / float64(len(points)))
	}
	return queryData{
		Series: []ChartSeries{{Key: "average_ticket_size", Label: "Average Ticket Size", Points: points}},
		Table:  table,
		Value:  value,
	}, nil
}

func (r *Repository) ClosingAmountWaterfall(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := closingScopedWhere(actor, req.Filters, tf, "sc.confirmed_at")
	where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	query := `
		SELECT
			CAST(COALESCE(SUM(sc.base_price), 0) AS DOUBLE) AS base_price_total,
			CAST(COALESCE(SUM(sc.discount_amount), 0) AS DOUBLE) AS discount_total,
			CAST(COALESCE(SUM(sc.additional_charge), 0) AS DOUBLE) AS additional_charge_total,
			CAST(COALESCE(SUM(` + closingRevenueExpr + `), 0) AS DOUBLE) AS revenue_total
		FROM sales_closings sc
		WHERE ` + where
	row := r.db.QueryRowContext(ctx, query, args...)
	var baseTotal, discountTotal, additionalChargeTotal, revenueTotal float64
	if err := row.Scan(&baseTotal, &discountTotal, &additionalChargeTotal, &revenueTotal); err != nil {
		return queryData{}, err
	}
	points := []ChartPoint{
		{X: "Base Price", Y: baseTotal},
		{X: "Discount", Y: -discountTotal},
		{X: "Additional Charge", Y: additionalChargeTotal},
		{X: "Omzet Snapshot", Y: revenueTotal},
	}
	table := []map[string]any{
		{"step": "BASE_PRICE", "label": "Base Price", "value": baseTotal},
		{"step": "DISCOUNT", "label": "Discount", "value": -discountTotal},
		{"step": "ADDITIONAL_CHARGE", "label": "Additional Charge", "value": additionalChargeTotal},
		{"step": "REVENUE_SNAPSHOT", "label": "Omzet Snapshot", "value": revenueTotal},
	}
	return queryData{
		Series: []ChartSeries{{Key: "waterfall", Label: "Closing Amount Waterfall", Points: points}},
		Table:  table,
		Extra: map[string]any{
			"base_price_total":        baseTotal,
			"discount_total":          discountTotal,
			"additional_charge_total": additionalChargeTotal,
			"revenue_total":           revenueTotal,
		},
		Value: revenueTotal,
	}, nil
}

func (r *Repository) TargetVsActual(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	periodYear, periodMonth := analyticsPeriodFrom(tf)
	metricCode := selectedTargetMetricCode(req.Metrics)
	salesItems, err := r.listVisibleSalesForAnalytics(ctx, actor, req.Filters.SalesIDs)
	if err != nil {
		return queryData{}, err
	}
	salesIDs := salesIDsFromItems(salesItems)
	targets, err := r.targetValuesByMetric(ctx, metricCode, periodYear, periodMonth, salesIDs)
	if err != nil {
		return queryData{}, err
	}
	actuals, err := r.actualValuesByMetric(ctx, metricCode, salesIDs, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}

	targetSeries := ChartSeries{Key: "target_value", Label: "Target"}
	actualSeries := ChartSeries{Key: "actual_value", Label: "Actual"}
	table := make([]map[string]any, 0, len(salesItems))
	actualTotal := 0.0
	targetTotal := 0.0
	for _, sales := range salesItems {
		target := round2(targets[sales.ID])
		actual := round2(actuals[sales.ID])
		gap := round2(actual - target)
		achievement := 0.0
		if target > 0 {
			achievement = round2((actual / target) * 100)
		}
		targetSeries.Points = append(targetSeries.Points, ChartPoint{X: sales.Name, Y: target})
		actualSeries.Points = append(actualSeries.Points, ChartPoint{X: sales.Name, Y: actual})
		table = append(table, map[string]any{
			"sales_id":         sales.ID,
			"sales_name":       sales.Name,
			"metric_code":      metricCode,
			"period_year":      periodYear,
			"period_month":     periodMonth,
			"target_value":     target,
			"actual_value":     actual,
			"gap_value":        gap,
			"achievement_pct":  achievement,
		})
		targetTotal += target
		actualTotal += actual
	}
	return queryData{
		Series: []ChartSeries{targetSeries, actualSeries},
		Table:  table,
		Extra: map[string]any{
			"metric_code":  metricCode,
			"period_year":  periodYear,
			"period_month": periodMonth,
			"target_total": round2(targetTotal),
			"actual_total": round2(actualTotal),
		},
		Value: actualTotal,
	}, nil
}

func (r *Repository) TargetBurnup(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	periodYear, periodMonth := analyticsPeriodFrom(tf)
	metricCode := selectedTargetMetricCode(req.Metrics)
	salesItems, err := r.listVisibleSalesForAnalytics(ctx, actor, req.Filters.SalesIDs)
	if err != nil {
		return queryData{}, err
	}
	salesIDs := salesIDsFromItems(salesItems)
	targets, err := r.targetValuesByMetric(ctx, metricCode, periodYear, periodMonth, salesIDs)
	if err != nil {
		return queryData{}, err
	}
	targetTotal := 0.0
	for _, value := range targets {
		targetTotal += value
	}
	dailyActuals, err := r.dailyActualValuesByMetric(ctx, metricCode, salesIDs, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}

	actualSeries := ChartSeries{Key: "actual_cumulative", Label: "Actual Kumulatif"}
	targetSeries := ChartSeries{Key: "target_cumulative", Label: "Target Kumulatif"}
	table := make([]map[string]any, 0)

	dayCount := int(tf.End.Sub(tf.Start).Hours() / 24)
	if dayCount <= 0 {
		dayCount = 1
	}
	runningActual := 0.0
	for i := 0; i < dayCount; i++ {
		day := tf.Start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		runningActual += dailyActuals[key]
		targetValue := 0.0
		if targetTotal > 0 {
			targetValue = round2((targetTotal / float64(dayCount)) * float64(i+1))
		}
		actualSeries.Points = append(actualSeries.Points, ChartPoint{X: key, Y: round2(runningActual)})
		targetSeries.Points = append(targetSeries.Points, ChartPoint{X: key, Y: targetValue})
		table = append(table, map[string]any{
			"period":            key,
			"actual_cumulative": round2(runningActual),
			"target_cumulative": targetValue,
			"daily_actual":      round2(dailyActuals[key]),
			"metric_code":       metricCode,
		})
	}

	return queryData{
		Series: []ChartSeries{actualSeries, targetSeries},
		Table:  table,
		Extra: map[string]any{
			"metric_code":  metricCode,
			"period_year":  periodYear,
			"period_month": periodMonth,
			"target_total": round2(targetTotal),
		},
		Value: round2(runningActual),
	}, nil
}

func (r *Repository) KpiLeaderboard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	periodYear, periodMonth := analyticsPeriodFrom(tf)
	where := []string{"skr.period_year = ?", "skr.period_month = ?"}
	args := []any{periodYear, periodMonth}
	switch actor.RoleCode {
	case identity.RoleAdmin, identity.RoleSupervisor:
	case identity.RoleSales:
		where = append(where, "skr.sales_id = ?")
		args = append(args, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	if len(req.Filters.SalesIDs) > 0 {
		where = append(where, "skr.sales_id IN ("+placeholders(len(req.Filters.SalesIDs))+")")
		for _, id := range req.Filters.SalesIDs {
			args = append(args, id)
		}
	}
	query := `
		SELECT skr.sales_id, COALESCE(u.name, 'Tanpa Sales') AS sales_name, skr.total_score, skr.rank_position, skr.classification
		FROM sales_kpi_results skr
		LEFT JOIN users u ON u.id = skr.sales_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY skr.rank_position ASC, skr.total_score DESC, sales_name ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	series := ChartSeries{Key: "total_score", Label: "KPI Score"}
	table := make([]map[string]any, 0)
	total := 0.0
	for rows.Next() {
		var salesID int64
		var salesName, classification string
		var totalScore float64
		var rank sql.NullInt64
		if err := rows.Scan(&salesID, &salesName, &totalScore, &rank, &classification); err != nil {
			return queryData{}, err
		}
		series.Points = append(series.Points, ChartPoint{X: salesName, Y: totalScore})
		table = append(table, map[string]any{
			"sales_id":        salesID,
			"sales_name":      salesName,
			"total_score":     totalScore,
			"rank_position":   nullableInt(rank),
			"classification":  classification,
			"period_year":     periodYear,
			"period_month":    periodMonth,
		})
		total += totalScore
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"period_year":  periodYear,
			"period_month": periodMonth,
		},
		Value: total,
	}, rows.Err()
}

func (r *Repository) ActivityVsClosingScatter(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	salesItems, err := r.listVisibleSalesForAnalytics(ctx, actor, req.Filters.SalesIDs)
	if err != nil {
		return queryData{}, err
	}
	salesIDs := salesIDsFromItems(salesItems)
	activityMap, err := r.actualValuesByMetric(ctx, "CALL_CUSTOMER_COUNT", salesIDs, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	closingMap, err := r.actualValuesByMetric(ctx, "CONFIRMED_CLOSING_COUNT", salesIDs, tf.Start, tf.End)
	if err != nil {
		return queryData{}, err
	}
	series := ChartSeries{Key: "activity_vs_closing", Label: "Activity vs Closing"}
	table := make([]map[string]any, 0, len(salesItems))
	totalClosing := 0.0
	for _, sales := range salesItems {
		activity := round2(activityMap[sales.ID])
		closing := round2(closingMap[sales.ID])
		conversion := 0.0
		if activity > 0 {
			conversion = round2((closing / activity) * 100)
		}
		series.Points = append(series.Points, ChartPoint{
			X: map[string]any{
				"sales_id":         sales.ID,
				"sales_name":       sales.Name,
				"interaction_count": activity,
			},
			Y: closing,
		})
		table = append(table, map[string]any{
			"sales_id":          sales.ID,
			"sales_name":        sales.Name,
			"interaction_count": activity,
			"closing_count":     closing,
			"conversion_rate":   conversion,
		})
		totalClosing += closing
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"x_metric": "CALL_CUSTOMER_COUNT",
			"y_metric": "CONFIRMED_CLOSING_COUNT",
		},
		Value: totalClosing,
	}, nil
}

func selectedClosingMetric(metrics []string, fallback string) string {
	allowed := map[string]string{
		"closing_count":            "closing_count",
		"confirmed_closing_count":  "confirmed_closing_count",
		"gross_revenue_snapshot":   "gross_revenue_snapshot",
		"average_ticket_size":      "average_ticket_size",
	}
	for _, metric := range metrics {
		key := strings.ToLower(strings.TrimSpace(metric))
		if value, ok := allowed[key]; ok {
			return value
		}
	}
	return fallback
}

func closingMetricLabel(metric string) string {
	switch metric {
	case "closing_count":
		return "Closing Count"
	case "gross_revenue_snapshot":
		return "Omzet Snapshot"
	case "average_ticket_size":
		return "Average Ticket Size"
	default:
		return "Confirmed Closing Count"
	}
}

func closingAggregateExpr(metric string) string {
	switch metric {
	case "gross_revenue_snapshot":
		return "CAST(COALESCE(SUM(" + closingRevenueExpr + "), 0) AS DOUBLE)"
	case "average_ticket_size":
		return "CAST(COALESCE(AVG(" + closingRevenueExpr + "), 0) AS DOUBLE)"
	default:
		return "COUNT(*)"
	}
}

func analyticsPeriodFrom(tf ResolvedTimeFilter) (int, int) {
	end := tf.End.Add(-time.Nanosecond)
	return end.Year(), int(end.Month())
}

func selectedTargetMetricCode(metrics []string) string {
	allowed := map[string]string{
		"confirmed_closing_count":  "CONFIRMED_CLOSING_COUNT",
		"closing_count":            "CONFIRMED_CLOSING_COUNT",
		"confirmed_closing_amount": "CONFIRMED_CLOSING_AMOUNT",
		"gross_revenue_snapshot":   "CONFIRMED_CLOSING_AMOUNT",
		"call_customer_count":      "CALL_CUSTOMER_COUNT",
		"training_count":           "TRAINING_COUNT",
	}
	for _, metric := range metrics {
		key := strings.ToLower(strings.TrimSpace(metric))
		if value, ok := allowed[key]; ok {
			return value
		}
		key = strings.ToUpper(strings.TrimSpace(metric))
		switch key {
		case "CONFIRMED_CLOSING_COUNT", "CONFIRMED_CLOSING_AMOUNT", "CALL_CUSTOMER_COUNT", "TRAINING_COUNT":
			return key
		}
	}
	return "CONFIRMED_CLOSING_COUNT"
}

func (r *Repository) listVisibleSalesForAnalytics(ctx context.Context, actor identity.User, filterSalesIDs []int64) ([]analyticsSalesInfo, error) {
	where := []string{"r.code = 'SALES'", "u.status = 'ACTIVE'", "u.deleted_at IS NULL"}
	args := []any{}
	switch actor.RoleCode {
	case identity.RoleAdmin, identity.RoleSupervisor:
	case identity.RoleSales:
		where = append(where, "u.id = ?")
		args = append(args, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	if len(filterSalesIDs) > 0 {
		where = append(where, "u.id IN ("+placeholders(len(filterSalesIDs))+")")
		for _, id := range filterSalesIDs {
			args = append(args, id)
		}
	}
	query := `
		SELECT u.id, u.name
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY u.name ASC, u.id ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]analyticsSalesInfo, 0)
	for rows.Next() {
		var item analyticsSalesInfo
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func salesIDsFromItems(items []analyticsSalesInfo) []int64 {
	result := make([]int64, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

func (r *Repository) targetValuesByMetric(ctx context.Context, metricCode string, periodYear, periodMonth int, salesIDs []int64) (map[int64]float64, error) {
	values := make(map[int64]float64, len(salesIDs))
	for _, id := range salesIDs {
		values[id] = 0
	}
	if len(salesIDs) == 0 {
		return values, nil
	}
	query := `
		SELECT st.sales_id, CAST(st.target_value AS DOUBLE) AS target_value
		FROM sales_targets st
		JOIN metric_codes mc ON mc.id = st.metric_code_id
		WHERE mc.code = ?
			AND st.period_year = ?
			AND st.period_month = ?
			AND st.sales_id IN (` + placeholders(len(salesIDs)) + `)`
	args := []any{metricCode, periodYear, periodMonth}
	for _, id := range salesIDs {
		args = append(args, id)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var salesID int64
		var value float64
		if err := rows.Scan(&salesID, &value); err != nil {
			return nil, err
		}
		values[salesID] = value
	}
	return values, rows.Err()
}

func (r *Repository) actualValuesByMetric(ctx context.Context, metricCode string, salesIDs []int64, from, to time.Time) (map[int64]float64, error) {
	values := make(map[int64]float64, len(salesIDs))
	for _, id := range salesIDs {
		values[id] = 0
	}
	if len(salesIDs) == 0 {
		return values, nil
	}
	query, timeColumn, aggregateExpr, err := analyticsMetricQuery(metricCode)
	if err != nil {
		return nil, err
	}
	query = `
		SELECT sales_id, ` + aggregateExpr + ` AS total
		FROM ` + query + `
		WHERE sales_id IN (` + placeholders(len(salesIDs)) + `)
			AND ` + timeColumn + ` >= ? AND ` + timeColumn + ` < ?
		GROUP BY sales_id`
	args := make([]any, 0, len(salesIDs)+2)
	for _, id := range salesIDs {
		args = append(args, id)
	}
	args = append(args, from, to)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var salesID int64
		var value float64
		if err := rows.Scan(&salesID, &value); err != nil {
			return nil, err
		}
		values[salesID] = value
	}
	return values, rows.Err()
}

func (r *Repository) dailyActualValuesByMetric(ctx context.Context, metricCode string, salesIDs []int64, from, to time.Time) (map[string]float64, error) {
	values := map[string]float64{}
	if len(salesIDs) == 0 {
		return values, nil
	}
	query, timeColumn, aggregateExpr, err := analyticsMetricQuery(metricCode)
	if err != nil {
		return nil, err
	}
	query = `
		SELECT DATE_FORMAT(` + timeColumn + `, '%Y-%m-%d') AS period, ` + aggregateExpr + ` AS total
		FROM ` + query + `
		WHERE sales_id IN (` + placeholders(len(salesIDs)) + `)
			AND ` + timeColumn + ` >= ? AND ` + timeColumn + ` < ?
		GROUP BY DATE_FORMAT(` + timeColumn + `, '%Y-%m-%d')
		ORDER BY period ASC`
	args := make([]any, 0, len(salesIDs)+2)
	for _, id := range salesIDs {
		args = append(args, id)
	}
	args = append(args, from, to)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var period string
		var value float64
		if err := rows.Scan(&period, &value); err != nil {
			return nil, err
		}
		values[period] = value
	}
	return values, rows.Err()
}

func analyticsMetricQuery(metricCode string) (fromClause, timeColumn, aggregateExpr string, err error) {
	switch metricCode {
	case "CONFIRMED_CLOSING_COUNT":
		return "sales_closings", "confirmed_at", "COUNT(*)", nil
	case "CONFIRMED_CLOSING_AMOUNT":
		return "sales_closings", "confirmed_at", "CAST(COALESCE(SUM(base_price - discount_amount + additional_charge), 0) AS DOUBLE)", nil
	case "CALL_CUSTOMER_COUNT":
		return "customer_interactions", "interaction_at", "COUNT(*)", nil
	case "TRAINING_COUNT":
		return "training_reports", "completed_at", "COUNT(*)", nil
	default:
		return "", "", "", fmt.Errorf("metric analytics tidak didukung: %s", metricCode)
	}
}

func closingScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"sc.deleted_at IS NULL"}
	args := []any{}
	switch actor.RoleCode {
	case identity.RoleAdmin:
		where = append(where, "1 = 1")
	case identity.RoleSupervisor:
		where = append(where, "(sc.supervisor_id = ? OR EXISTS (SELECT 1 FROM customer_leads cl WHERE cl.id = sc.lead_id AND cl.deleted_at IS NULL AND cl.supervisor_id = ?))")
		args = append(args, actor.ID, actor.ID)
	case identity.RoleSales:
		where = append(where, "sc.sales_id = ?")
		args = append(args, actor.ID)
	default:
		where = append(where, "1 = 0")
	}
	if timeColumn != "" {
		where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
		args = append(args, tf.Start, tf.End)
	}
	appendStringFilter(&where, &args, "sc.status", filters.Status)
	appendIntFilter(&where, &args, "sc.sales_id", filters.SalesIDs)
	appendIntFilter(&where, &args, "sc.supervisor_id", filters.SupervisorIDs)
	appendIntFilter(&where, &args, "sc.owner_id", filters.OwnerIDs)
	appendIntFilter(&where, &args, "sc.outlet_id", filters.OutletIDs)
	return strings.Join(where, " AND "), args
}

func (r *Repository) queryCategoryAggregate(ctx context.Context, query string, args []any, dimensionKey, metricKey string) (queryData, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()
	series := ChartSeries{Key: metricKey, Label: closingMetricLabel(metricKey)}
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
	return queryData{Series: []ChartSeries{series}, Table: table, Value: round2(total)}, rows.Err()
}

func sortSeriesMap(seriesMap map[string]*ChartSeries) []ChartSeries {
	keys := make([]string, 0, len(seriesMap))
	for key := range seriesMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	series := make([]ChartSeries, 0, len(keys))
	for _, key := range keys {
		series = append(series, *seriesMap[key])
	}
	return series
}

func slugifyKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func nullableInt(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func normalizeHistoryMetricValue(raw string) float64 {
	value, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return value
}
