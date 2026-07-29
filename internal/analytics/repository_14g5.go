package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"backend_crm_piposmart/internal/identity"
)

func (r *Repository) ImportBatchesPerProfile(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT ib.profile AS label, COUNT(*) AS total
		FROM import_batches ib
		WHERE ` + where + `
		GROUP BY ib.profile
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "profile", "batch_count")
}

func (r *Repository) ImportSuccessVsFailed(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN ib.status IN ('VALIDATED', 'COMMITTED') THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN ib.status IN ('VALIDATION_FAILED', 'COMMIT_FAILED') THEN 1 ELSE 0 END), 0) AS failed_count
		FROM import_batches ib
		WHERE ` + where
	var successCount, failedCount float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&successCount, &failedCount); err != nil {
		return queryData{}, err
	}
	total := successCount + failedCount
	successRate := 0.0
	if total > 0 {
		successRate = round2((successCount / total) * 100)
	}
	return queryData{
		Series: []ChartSeries{{
			Key:   "batch_count",
			Label: "Import Batch",
			Points: []ChartPoint{
				{X: "SUCCESS", Y: successCount},
				{X: "FAILED", Y: failedCount},
			},
		}},
		Table: []map[string]any{
			{"status_group": "SUCCESS", "batch_count": successCount},
			{"status_group": "FAILED", "batch_count": failedCount},
		},
		Extra: map[string]any{
			"success_rate": successRate,
		},
		Value: successRate,
	}, nil
}

func (r *Repository) InvalidRowsDistribution(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT COALESCE(ib.invalid_rows, 0) AS invalid_rows
		FROM import_batches ib
		WHERE ` + where
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	buckets := map[string]float64{
		"0":      0,
		"1-5":    0,
		"6-20":   0,
		"21-50":  0,
		"51-100": 0,
		"100+":   0,
	}
	totalInvalidRows := 0.0
	totalBatches := 0.0
	for rows.Next() {
		var invalidRows int64
		if err := rows.Scan(&invalidRows); err != nil {
			return queryData{}, err
		}
		totalBatches++
		totalInvalidRows += float64(invalidRows)
		switch {
		case invalidRows <= 0:
			buckets["0"]++
		case invalidRows <= 5:
			buckets["1-5"]++
		case invalidRows <= 20:
			buckets["6-20"]++
		case invalidRows <= 50:
			buckets["21-50"]++
		case invalidRows <= 100:
			buckets["51-100"]++
		default:
			buckets["100+"]++
		}
	}
	if err := rows.Err(); err != nil {
		return queryData{}, err
	}
	labels := []string{"0", "1-5", "6-20", "21-50", "51-100", "100+"}
	series := ChartSeries{Key: "batch_count", Label: "Batch"}
	table := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		series.Points = append(series.Points, ChartPoint{X: label, Y: buckets[label]})
		table = append(table, map[string]any{
			"invalid_row_bucket": label,
			"batch_count":        buckets[label],
		})
	}
	avgInvalidRows := 0.0
	if totalBatches > 0 {
		avgInvalidRows = round2(totalInvalidRows / totalBatches)
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"average_invalid_rows_per_batch": avgInvalidRows,
		},
		Value: avgInvalidRows,
	}, nil
}

func (r *Repository) ValidationErrorByProfile(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT
			ib.profile,
			CAST(COALESCE(SUM(ib.invalid_rows), 0) AS DOUBLE) AS invalid_row_count,
			SUM(CASE WHEN ib.status = 'VALIDATION_FAILED' THEN 1 ELSE 0 END) AS failed_batch_count
		FROM import_batches ib
		WHERE ` + where + `
		GROUP BY ib.profile
		ORDER BY invalid_row_count DESC, ib.profile ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	invalidSeries := ChartSeries{Key: "invalid_row_count", Label: "Invalid Rows"}
	failedSeries := ChartSeries{Key: "failed_batch_count", Label: "Validation Failed Batch"}
	table := make([]map[string]any, 0)
	totalInvalid := 0.0
	for rows.Next() {
		var profile string
		var invalidRows, failedBatches float64
		if err := rows.Scan(&profile, &invalidRows, &failedBatches); err != nil {
			return queryData{}, err
		}
		invalidSeries.Points = append(invalidSeries.Points, ChartPoint{X: profile, Y: invalidRows})
		failedSeries.Points = append(failedSeries.Points, ChartPoint{X: profile, Y: failedBatches})
		table = append(table, map[string]any{
			"profile":            profile,
			"invalid_row_count":  invalidRows,
			"failed_batch_count": failedBatches,
		})
		totalInvalid += invalidRows
	}
	return queryData{
		Series: []ChartSeries{invalidSeries, failedSeries},
		Table:  table,
		Value:  totalInvalid,
	}, rows.Err()
}

func (r *Repository) DuplicateDetectionRate(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importRowScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	duplicateCondition := duplicateImportRowCondition("ir")
	query := `
		SELECT ` + groupExpr("ib.uploaded_at", tf.Granularity) + ` AS period,
			COUNT(*) AS total_rows,
			SUM(CASE WHEN ` + duplicateCondition + ` THEN 1 ELSE 0 END) AS duplicate_rows
		FROM import_rows ir
		JOIN import_batches ib ON ib.id = ir.batch_id
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return queryData{}, err
	}
	defer rows.Close()

	series := ChartSeries{Key: "duplicate_rate", Label: "Duplicate Rate (%)"}
	table := make([]map[string]any, 0)
	totalDuplicate := 0.0
	totalRows := 0.0
	for rows.Next() {
		var period string
		var rowCount, duplicateCount float64
		if err := rows.Scan(&period, &rowCount, &duplicateCount); err != nil {
			return queryData{}, err
		}
		rate := 0.0
		if rowCount > 0 {
			rate = round2((duplicateCount / rowCount) * 100)
		}
		series.Points = append(series.Points, ChartPoint{X: period, Y: rate})
		table = append(table, map[string]any{
			"period":         period,
			"total_rows":     rowCount,
			"duplicate_rows": duplicateCount,
			"duplicate_rate": rate,
		})
		totalDuplicate += duplicateCount
		totalRows += rowCount
	}
	overallRate := 0.0
	if totalRows > 0 {
		overallRate = round2((totalDuplicate / totalRows) * 100)
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"source_note": "Duplicate detection rate dihitung dari validation_errors atau commit_error yang mengandung indikator duplicate/duplikat/already exists.",
		},
		Value: overallRate,
	}, rows.Err()
}

func (r *Repository) ImportDurationTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	where += " AND COALESCE(ib.committed_at, ib.validated_at) IS NOT NULL"
	query := `
		SELECT ` + groupExpr("ib.uploaded_at", tf.Granularity) + ` AS period,
			AVG(TIMESTAMPDIFF(SECOND, ib.uploaded_at, COALESCE(ib.committed_at, ib.validated_at))) / 60 AS duration_minutes
		FROM import_batches ib
		WHERE ` + where + `
		GROUP BY period
		ORDER BY period ASC`
	points, table, total, err := r.readTrendRows(ctx, query, args, "duration_minutes")
	if err != nil {
		return queryData{}, err
	}
	avg := 0.0
	if len(points) > 0 {
		avg = round2(total / float64(len(points)))
	}
	return queryData{
		Series: []ChartSeries{{Key: "duration_minutes", Label: "Average Duration (Minutes)", Points: points}},
		Table:  table,
		Value:  avg,
	}, nil
}

func (r *Repository) BatchStatusFunnel(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT
			COUNT(*) AS uploaded_count,
			COALESCE(SUM(CASE WHEN ib.status IN ('VALIDATED', 'COMMITTING', 'COMMITTED', 'COMMIT_FAILED') THEN 1 ELSE 0 END), 0) AS validated_count,
			COALESCE(SUM(CASE WHEN ib.status = 'COMMITTED' THEN 1 ELSE 0 END), 0) AS committed_count,
			COALESCE(SUM(CASE WHEN ib.status IN ('VALIDATION_FAILED', 'COMMIT_FAILED') THEN 1 ELSE 0 END), 0) AS failed_count
		FROM import_batches ib
		WHERE ` + where
	var uploadedCount, validatedCount, committedCount, failedCount float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&uploadedCount, &validatedCount, &committedCount, &failedCount); err != nil {
		return queryData{}, err
	}
	commitRate := 0.0
	if uploadedCount > 0 {
		commitRate = round2((committedCount / uploadedCount) * 100)
	}
	return queryData{
		Series: []ChartSeries{{
			Key:   "batch_count",
			Label: "Import Funnel",
			Points: []ChartPoint{
				{X: "UPLOADED", Y: uploadedCount},
				{X: "VALIDATED", Y: validatedCount},
				{X: "COMMITTED", Y: committedCount},
			},
		}},
		Table: []map[string]any{
			{"stage": "UPLOADED", "batch_count": uploadedCount},
			{"stage": "VALIDATED", "batch_count": validatedCount},
			{"stage": "COMMITTED", "batch_count": committedCount},
		},
		Extra: map[string]any{
			"failed_count": failedCount,
			"commit_rate":  commitRate,
		},
		Value: commitRate,
	}, nil
}

func (r *Repository) ImportUploaderActivity(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	where, args := importBatchScopedWhere(actor, req.Filters, tf, "ib.uploaded_at")
	query := `
		SELECT COALESCE(u.name, CONCAT('User #', ib.uploaded_by_user_id)) AS label, COUNT(*) AS total
		FROM import_batches ib
		LEFT JOIN users u ON u.id = ib.uploaded_by_user_id
		WHERE ` + where + `
		GROUP BY ib.uploaded_by_user_id, u.name
		ORDER BY total DESC, label ASC`
	return r.queryCategoryCount(ctx, query, args, "uploader_name", "batch_count")
}

func (r *Repository) FileHistoryUsage(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	return queryData{
		Series: []ChartSeries{},
		Table:  []map[string]any{},
		Extra: map[string]any{
			"source_note":       "Viewer/download usage belum dipersist pada storage backend, sehingga chart ini saat ini berfungsi sebagai placeholder kontrak analytics.",
			"recommended_next":  "Tambahkan audit/event log untuk open file dan download file agar usage analytics bisa dihitung nyata.",
			"time_filter_label": tf.Label,
		},
		Value: 0,
	}, nil
}

func (r *Repository) EndToEndBusinessFunnel(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	ownerCount, err := r.metricValue(ctx, actor, req.Filters, tf, "owner_count")
	if err != nil {
		return queryData{}, err
	}
	leadCount, err := r.metricValue(ctx, actor, req.Filters, tf, "lead_count")
	if err != nil {
		return queryData{}, err
	}
	trainingCount, err := r.metricValue(ctx, actor, req.Filters, tf, "training_completed_count")
	if err != nil {
		return queryData{}, err
	}
	closingCount, err := r.metricValue(ctx, actor, req.Filters, tf, "confirmed_closing_count")
	if err != nil {
		return queryData{}, err
	}
	subscriptionActivationCount, err := r.metricValue(ctx, actor, req.Filters, tf, "subscription_activation_count")
	if err != nil {
		return queryData{}, err
	}
	ownerToLead := percent(leadCount, ownerCount)
	leadToTraining := percent(trainingCount, leadCount)
	trainingToClosing := percent(closingCount, trainingCount)
	closingToSubscription := percent(subscriptionActivationCount, closingCount)
	return queryData{
		Series: []ChartSeries{{
			Key:   "entity_count",
			Label: "Business Funnel",
			Points: []ChartPoint{
				{X: "OWNER", Y: ownerCount},
				{X: "LEAD", Y: leadCount},
				{X: "TRAINING", Y: trainingCount},
				{X: "CLOSING_CONFIRMED", Y: closingCount},
				{X: "SUBSCRIPTION_ACTIVATED", Y: subscriptionActivationCount},
			},
		}},
		Table: []map[string]any{
			{"stage": "OWNER", "entity_count": ownerCount},
			{"stage": "LEAD", "entity_count": leadCount, "conversion_rate": ownerToLead},
			{"stage": "TRAINING", "entity_count": trainingCount, "conversion_rate": leadToTraining},
			{"stage": "CLOSING_CONFIRMED", "entity_count": closingCount, "conversion_rate": trainingToClosing},
			{"stage": "SUBSCRIPTION_ACTIVATED", "entity_count": subscriptionActivationCount, "conversion_rate": closingToSubscription},
		},
		Extra: map[string]any{
			"conversion_owner_to_lead":           ownerToLead,
			"conversion_lead_to_training":        leadToTraining,
			"conversion_training_to_closing":     trainingToClosing,
			"conversion_closing_to_subscription": closingToSubscription,
		},
		Value: subscriptionActivationCount,
	}, nil
}

func (r *Repository) RevenueClosingActiveSubscriptionBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	topupSeries := ChartSeries{Key: "topup_revenue", Label: "Topup Revenue"}
	closingSeries := ChartSeries{Key: "closing_revenue_snapshot", Label: "Closing Revenue Snapshot"}
	activeSeries := ChartSeries{Key: "active_subscription_count", Label: "Active Subscription"}
	table := make([]map[string]any, 0, len(months))
	totalClosingRevenue := 0.0
	for _, month := range months {
		start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		period := start.Format("2006-01")
		monthTF := ResolvedTimeFilter{Mode: "month_range", Granularity: "month", Start: start, End: end, Label: period}

		topupRevenue, err := r.metricValue(ctx, actor, req.Filters, monthTF, "topup_revenue")
		if err != nil {
			return queryData{}, err
		}
		closingRevenue, err := r.metricValue(ctx, actor, req.Filters, monthTF, "closing_revenue_snapshot")
		if err != nil {
			return queryData{}, err
		}
		activeSubscriptions, err := r.metricValue(ctx, actor, req.Filters, monthTF, "active_subscription_count")
		if err != nil {
			return queryData{}, err
		}
		topupSeries.Points = append(topupSeries.Points, ChartPoint{X: period, Y: topupRevenue})
		closingSeries.Points = append(closingSeries.Points, ChartPoint{X: period, Y: closingRevenue})
		activeSeries.Points = append(activeSeries.Points, ChartPoint{X: period, Y: activeSubscriptions})
		table = append(table, map[string]any{
			"period":                    period,
			"topup_revenue":             topupRevenue,
			"closing_revenue_snapshot":  closingRevenue,
			"active_subscription_count": activeSubscriptions,
		})
		totalClosingRevenue += closingRevenue
	}
	return queryData{
		Series: []ChartSeries{topupSeries, closingSeries, activeSeries},
		Table:  table,
		Extra: map[string]any{
			"source_note": "Topup revenue dan closing revenue sengaja dipisahkan agar tidak terjadi double counting pada pembacaan bisnis.",
		},
		Value: totalClosingRevenue,
	}, nil
}

func (r *Repository) MonthlyOperatingReviewBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{
		"owner_count",
		"outlet_count",
		"lead_count",
		"confirmed_closing_count",
		"topup_revenue",
		"active_subscription_count",
	})
	table, series, total, err := r.metricBoard(ctx, actor, req, tf, metrics)
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"board_type": "monthly_operating_review",
		},
		Value: total,
	}, nil
}

func (r *Repository) NorthStarKPITrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{
		"owner_count",
		"confirmed_closing_count",
		"topup_revenue",
		"active_subscription_count",
	})
	series := make([]ChartSeries, 0, len(metrics))
	tableByPeriod := map[string]map[string]any{}
	total := 0.0
	for _, metric := range metrics {
		itemSeries, itemTable, value, err := r.metricTrend(ctx, actor, req.Filters, tf, metric)
		if err != nil {
			return queryData{}, err
		}
		series = append(series, itemSeries)
		total += value
		for _, row := range itemTable {
			period := fmt.Sprint(row["period"])
			entry, ok := tableByPeriod[period]
			if !ok {
				entry = map[string]any{"period": period}
				tableByPeriod[period] = entry
			}
			entry[metric] = row[metric]
		}
	}
	periods := make([]string, 0, len(tableByPeriod))
	for period := range tableByPeriod {
		periods = append(periods, period)
	}
	sort.Strings(periods)
	table := make([]map[string]any, 0, len(periods))
	for _, period := range periods {
		table = append(table, tableByPeriod[period])
	}
	return queryData{
		Series: series,
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) DataQualityScoreByModule(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	importScore, importMeta, err := r.importQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	leadScore, leadMeta, err := r.leadQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	salesScore, salesMeta, err := r.salesQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	walletScore, walletMeta, err := r.walletQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	subscriptionScore, subscriptionMeta, err := r.subscriptionQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	partnerScore, partnerMeta, err := r.partnerQualityScore(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}

	type scoreRow struct {
		module string
		score  float64
		meta   map[string]any
	}
	rows := []scoreRow{
		{module: "imports", score: importScore, meta: importMeta},
		{module: "leads", score: leadScore, meta: leadMeta},
		{module: "sales", score: salesScore, meta: salesMeta},
		{module: "wallets", score: walletScore, meta: walletMeta},
		{module: "subscriptions", score: subscriptionScore, meta: subscriptionMeta},
		{module: "partners", score: partnerScore, meta: partnerMeta},
	}
	series := ChartSeries{Key: "quality_score", Label: "Quality Score"}
	table := make([]map[string]any, 0, len(rows))
	total := 0.0
	formulas := map[string]any{}
	for _, row := range rows {
		series.Points = append(series.Points, ChartPoint{X: strings.ToUpper(row.module), Y: row.score})
		item := map[string]any{
			"module":        row.module,
			"quality_score": row.score,
		}
		for key, value := range row.meta {
			item[key] = value
		}
		table = append(table, item)
		formulas[row.module] = row.meta
		total += row.score
	}
	avg := 0.0
	if len(rows) > 0 {
		avg = round2(total / float64(len(rows)))
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Extra: map[string]any{
			"formula_note": "Score 0-100 ini adalah heuristic operational quality score dari data backend saat ini, bukan skor data quality enterprise yang final.",
			"formula_meta": formulas,
		},
		Value: avg,
	}, nil
}

func (r *Repository) CustomMultiSeriesTrend(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{
		"owner_count",
		"outlet_count",
		"lead_count",
	})
	series := make([]ChartSeries, 0, len(metrics))
	tableByPeriod := map[string]map[string]any{}
	total := 0.0
	for _, metric := range metrics {
		itemSeries, itemTable, value, err := r.metricTrend(ctx, actor, req.Filters, tf, metric)
		if err != nil {
			return queryData{}, err
		}
		series = append(series, itemSeries)
		total += value
		for _, row := range itemTable {
			period := fmt.Sprint(row["period"])
			entry, ok := tableByPeriod[period]
			if !ok {
				entry = map[string]any{"period": period}
				tableByPeriod[period] = entry
			}
			entry[metric] = row[metric]
		}
	}
	periods := make([]string, 0, len(tableByPeriod))
	for period := range tableByPeriod {
		periods = append(periods, period)
	}
	sort.Strings(periods)
	table := make([]map[string]any, 0, len(periods))
	for _, period := range periods {
		table = append(table, tableByPeriod[period])
	}
	return queryData{
		Series: series,
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) MetricComparisonBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{
		"owner_count",
		"outlet_count",
		"lead_count",
		"confirmed_closing_count",
	})
	table, series, total, err := r.metricBoard(ctx, actor, req, tf, metrics)
	if err != nil {
		return queryData{}, err
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Value:  total,
	}, nil
}

func (r *Repository) RegionComparisonBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{"owner_count"})
	comparisons := req.Comparison.CompareSeries
	if len(comparisons) == 0 {
		defaultRegions, err := r.defaultRegionComparisonSeries(ctx, actor, req.Filters, tf)
		if err != nil {
			return queryData{}, err
		}
		comparisons = defaultRegions
	}
	seriesMap := map[string]*ChartSeries{}
	table := make([]map[string]any, 0, len(comparisons))
	total := 0.0
	for _, metric := range metrics {
		seriesMap[metric] = &ChartSeries{Key: metric, Label: analyticsMetricLabel(metric)}
	}
	for _, item := range comparisons {
		row := map[string]any{
			"field": item.Field,
			"label": item.Label,
			"value": item.Value,
		}
		for _, metric := range metrics {
			value, err := r.metricValueByRegion(ctx, actor, req.Filters, tf, metric, item.Field, item.Value)
			if err != nil {
				return queryData{}, err
			}
			seriesMap[metric].Points = append(seriesMap[metric].Points, ChartPoint{X: item.Label, Y: value})
			row[metric] = value
			total += value
		}
		table = append(table, row)
	}
	return queryData{
		Series: sortSeriesMap(seriesMap),
		Table:  table,
		Extra: map[string]any{
			"comparison_mode": "series_to_series",
		},
		Value: total,
	}, nil
}

func (r *Repository) SubscriptionCohortRetention(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	months := monthRange(tf)
	maxOffset := len(months) - 1
	if maxOffset > 11 {
		maxOffset = 11
	}
	table := make([]map[string]any, 0)
	heatmap := make([]map[string]any, 0)
	totalRetention := 0.0
	totalCells := 0.0
	for _, cohortMonth := range months {
		cohortStart := time.Date(cohortMonth.Year(), cohortMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
		cohortEnd := cohortStart.AddDate(0, 1, 0)
		cohortSize, err := r.subscriptionCohortSize(ctx, actor, req.Filters, cohortStart, cohortEnd)
		if err != nil {
			return queryData{}, err
		}
		if cohortSize <= 0 {
			continue
		}
		for offset := 0; offset <= maxOffset; offset++ {
			observationStart := cohortStart.AddDate(0, offset, 0)
			if observationStart.After(tf.End) {
				break
			}
			observationEnd := observationStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
			retainedCount, err := r.subscriptionCohortRetainedCount(ctx, actor, req.Filters, cohortStart, cohortEnd, observationEnd)
			if err != nil {
				return queryData{}, err
			}
			retentionRate := percent(retainedCount, cohortSize)
			table = append(table, map[string]any{
				"cohort_month":      cohortStart.Format("2006-01"),
				"offset_month":      offset,
				"observation_month": observationStart.Format("2006-01"),
				"cohort_size":       cohortSize,
				"retained_count":    retainedCount,
				"retention_rate":    retentionRate,
			})
			heatmap = append(heatmap, map[string]any{
				"x":     offset,
				"y":     cohortStart.Format("2006-01"),
				"value": retentionRate,
			})
			totalRetention += retentionRate
			totalCells++
		}
	}
	avgRetention := 0.0
	if totalCells > 0 {
		avgRetention = round2(totalRetention / totalCells)
	}
	return queryData{
		Table: table,
		Extra: map[string]any{
			"heatmap": heatmap,
		},
		Value: avgRetention,
	}, nil
}

func (r *Repository) ForecastSummaryBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	baseDate := tf.End.Add(-time.Nanosecond)
	expiry30, expiry60, expiry90, err := r.expiryForecastCounts(ctx, actor, req.Filters, baseDate)
	if err != nil {
		return queryData{}, err
	}
	openIssues, err := r.openReconciliationIssueCount(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	unpaidCommissionCount, err := r.unpaidCommissionCount(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	importFailedCount, err := r.importFailedBatchCount(ctx, actor, req.Filters, tf)
	if err != nil {
		return queryData{}, err
	}
	riskScore := clamp100(round2(expiry30 + (openIssues * 2) + unpaidCommissionCount + importFailedCount))
	return queryData{
		Series: []ChartSeries{{
			Key:   "forecast_value",
			Label: "Forecast Summary",
			Points: []ChartPoint{
				{X: "EXPIRY_30_DAYS", Y: expiry30},
				{X: "EXPIRY_31_60_DAYS", Y: expiry60},
				{X: "EXPIRY_61_90_DAYS", Y: expiry90},
				{X: "OPEN_RECON_ISSUES", Y: openIssues},
				{X: "UNPAID_COMMISSIONS", Y: unpaidCommissionCount},
				{X: "FAILED_IMPORT_BATCHES", Y: importFailedCount},
			},
		}},
		Table: []map[string]any{
			{"metric": "EXPIRY_30_DAYS", "value": expiry30},
			{"metric": "EXPIRY_31_60_DAYS", "value": expiry60},
			{"metric": "EXPIRY_61_90_DAYS", "value": expiry90},
			{"metric": "OPEN_RECON_ISSUES", "value": openIssues},
			{"metric": "UNPAID_COMMISSIONS", "value": unpaidCommissionCount},
			{"metric": "FAILED_IMPORT_BATCHES", "value": importFailedCount},
		},
		Extra: map[string]any{
			"risk_score": riskScore,
			"risk_note":  "Semakin tinggi risk_score, semakin tinggi backlog operasional yang perlu diantisipasi.",
		},
		Value: riskScore,
	}, nil
}

func (r *Repository) ComparisonImpactSummary(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter) (queryData, error) {
	metrics := analyticsMetricList(req.Metrics, []string{
		"owner_count",
		"confirmed_closing_count",
		"topup_revenue",
	})
	comparison, baseline, err := resolveComparison(tf, req.Comparison, currentNowUTC())
	if err != nil {
		return queryData{}, err
	}
	series := ChartSeries{Key: "delta_percent", Label: "Delta Percent (%)"}
	table := make([]map[string]any, 0, len(metrics))
	totalDelta := 0.0
	for _, metric := range metrics {
		currentValue, err := r.metricValue(ctx, actor, req.Filters, tf, metric)
		if err != nil {
			return queryData{}, err
		}
		baselineValue := 0.0
		if baseline != nil {
			baselineValue, err = r.metricValue(ctx, actor, req.Filters, *baseline, metric)
			if err != nil {
				return queryData{}, err
			}
		}
		delta := round2(currentValue - baselineValue)
		deltaPct := percentDelta(currentValue, baselineValue)
		direction, statusValue := regionStatus(delta, analyticsMetricPolarity(metric))
		series.Points = append(series.Points, ChartPoint{X: analyticsMetricLabel(metric), Y: deltaPct})
		table = append(table, map[string]any{
			"metric":         metric,
			"metric_label":   analyticsMetricLabel(metric),
			"current_value":  currentValue,
			"baseline_value": baselineValue,
			"delta":          delta,
			"delta_percent":  deltaPct,
			"direction":      direction,
			"status_value":   statusValue,
			"baseline_label": comparison.BaselineLabel,
		})
		totalDelta += deltaPct
	}
	avgDelta := 0.0
	if len(metrics) > 0 {
		avgDelta = round2(totalDelta / float64(len(metrics)))
	}
	return queryData{
		Series: []ChartSeries{series},
		Table:  table,
		Value:  avgDelta,
	}, nil
}

func importBatchScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"1 = 1"}
	args := []any{}
	switch actor.RoleCode {
	case identity.RoleAdmin:
		where = append(where, "1 = 1")
	default:
		where = append(where, "ib.uploaded_by_user_id = ?")
		args = append(args, actor.ID)
	}
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, "ib.status", filters.Status)
	return strings.Join(where, " AND "), args
}

func importRowScopedWhere(actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, timeColumn string) (string, []any) {
	where := []string{"1 = 1"}
	args := []any{}
	switch actor.RoleCode {
	case identity.RoleAdmin:
		where = append(where, "1 = 1")
	default:
		where = append(where, "ib.uploaded_by_user_id = ?")
		args = append(args, actor.ID)
	}
	where = append(where, timeColumn+" >= ?", timeColumn+" < ?")
	args = append(args, tf.Start, tf.End)
	appendStringFilter(&where, &args, "ir.status", filters.Status)
	return strings.Join(where, " AND "), args
}

func duplicateImportRowCondition(alias string) string {
	return "(LOWER(COALESCE(CAST(" + alias + ".validation_errors AS CHAR(2000)), '')) LIKE '%duplik%'" +
		" OR LOWER(COALESCE(CAST(" + alias + ".validation_errors AS CHAR(2000)), '')) LIKE '%duplicate%'" +
		" OR LOWER(COALESCE(" + alias + ".commit_error, '')) LIKE '%duplik%'" +
		" OR LOWER(COALESCE(" + alias + ".commit_error, '')) LIKE '%duplicate%'" +
		" OR LOWER(COALESCE(" + alias + ".commit_error, '')) LIKE '%already exists%'" +
		" OR LOWER(COALESCE(" + alias + ".commit_error, '')) LIKE '%sudah terdaftar%')"
}

func analyticsMetricList(metrics []string, fallback []string) []string {
	clean := make([]string, 0, len(metrics))
	seen := map[string]struct{}{}
	source := metrics
	if len(source) == 0 {
		source = fallback
	}
	for _, item := range source {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, key)
	}
	if len(clean) == 0 {
		return append([]string{}, fallback...)
	}
	return clean
}

func analyticsMetricLabel(metric string) string {
	switch metric {
	case "owner_count":
		return "Owner Baru"
	case "outlet_count":
		return "Outlet Baru"
	case "lead_count":
		return "Lead Baru"
	case "training_completed_count":
		return "Training Selesai"
	case "confirmed_closing_count":
		return "Closing Confirmed"
	case "closing_revenue_snapshot":
		return "Omzet Closing Snapshot"
	case "topup_revenue":
		return "Revenue Topup"
	case "topup_transaction_count":
		return "Jumlah Topup"
	case "active_subscription_count":
		return "Subscription Aktif"
	case "subscription_activation_count":
		return "Subscription Activated"
	case "import_batch_count":
		return "Import Batch"
	case "import_failed_batch_count":
		return "Import Failed Batch"
	case "import_committed_batch_count":
		return "Import Committed Batch"
	default:
		return strings.ReplaceAll(strings.Title(strings.ReplaceAll(metric, "_", " ")), "Id", "ID")
	}
}

func analyticsMetricPolarity(metric string) string {
	switch metric {
	case "import_failed_batch_count":
		return "lower_is_better"
	default:
		return "higher_is_better"
	}
}

func (r *Repository) metricTrend(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, metric string) (ChartSeries, []map[string]any, float64, error) {
	switch metric {
	case "owner_count":
		where, args := ownerScopedWhere(actor, filters, "o", tf, "o.created_at", false)
		data, err := r.queryTrendCount(ctx, "owners", "o", "o.created_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "outlet_count":
		where, args := outletScopedWhere(actor, filters, "ot", tf, "ot.created_at")
		data, err := r.queryTrendCount(ctx, "outlets", "ot", "ot.created_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "lead_count":
		where, args := leadScopedWhere(actor, filters, "cl", tf, "cl.created_at")
		data, err := r.queryTrendCount(ctx, "customer_leads", "cl", "cl.created_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "training_completed_count":
		where, args := trainingScopedWhere(actor, filters, tf, "tr.completed_at")
		where += " AND tr.status = 'COMPLETED' AND tr.completed_at IS NOT NULL"
		data, err := r.queryTrendCount(ctx, "training_reports", "tr", "tr.completed_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "confirmed_closing_count":
		where, args := closingScopedWhere(actor, filters, tf, "sc.confirmed_at")
		where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
		data, err := r.queryTrendCount(ctx, "sales_closings", "sc", "sc.confirmed_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "closing_revenue_snapshot":
		where, args := closingScopedWhere(actor, filters, tf, "sc.confirmed_at")
		where += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
		query := `
			SELECT ` + groupExpr("sc.confirmed_at", tf.Granularity) + ` AS period, CAST(COALESCE(SUM(` + closingRevenueExpr + `), 0) AS DOUBLE) AS total
			FROM sales_closings sc
			WHERE ` + where + `
			GROUP BY period
			ORDER BY period ASC`
		points, table, total, err := r.readTrendRows(ctx, query, args, metric)
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return ChartSeries{Key: metric, Label: analyticsMetricLabel(metric), Points: points}, table, total, nil
	case "topup_revenue":
		where, args := walletPaymentScopedWhere(actor, filters, tf, "wp.paid_at")
		where += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
		query := `
			SELECT ` + groupExpr("wp.paid_at", tf.Granularity) + ` AS period, CAST(COALESCE(SUM(wp.amount), 0) AS DOUBLE) AS total
			FROM wallet_payments wp
			WHERE ` + where + `
			GROUP BY period
			ORDER BY period ASC`
		points, table, total, err := r.readTrendRows(ctx, query, args, metric)
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return ChartSeries{Key: metric, Label: analyticsMetricLabel(metric), Points: points}, table, total, nil
	case "topup_transaction_count":
		where, args := walletPaymentScopedWhere(actor, filters, tf, "wp.paid_at")
		where += " AND wp.status = 'PAID' AND wp.paid_at IS NOT NULL"
		data, err := r.queryTrendCount(ctx, "wallet_payments", "wp", "wp.paid_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "active_subscription_count":
		months := monthRange(tf)
		series := ChartSeries{Key: metric, Label: analyticsMetricLabel(metric)}
		table := make([]map[string]any, 0, len(months))
		total := 0.0
		for _, month := range months {
			monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
			monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
			count, err := r.countActiveSubscriptionsAtDate(ctx, actor, filters, monthEnd)
			if err != nil {
				return ChartSeries{}, nil, 0, err
			}
			period := monthStart.Format("2006-01")
			series.Points = append(series.Points, ChartPoint{X: period, Y: count})
			table = append(table, map[string]any{"period": period, metric: count})
			total += count
		}
		return series, table, total, nil
	case "import_batch_count":
		where, args := importBatchScopedWhere(actor, filters, tf, "ib.uploaded_at")
		data, err := r.queryTrendCount(ctx, "import_batches", "ib", "ib.uploaded_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "import_failed_batch_count":
		where, args := importBatchScopedWhere(actor, filters, tf, "ib.uploaded_at")
		where += " AND ib.status IN ('VALIDATION_FAILED', 'COMMIT_FAILED')"
		data, err := r.queryTrendCount(ctx, "import_batches", "ib", "ib.uploaded_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "import_committed_batch_count":
		where, args := importBatchScopedWhere(actor, filters, tf, "ib.uploaded_at")
		where += " AND ib.status = 'COMMITTED'"
		data, err := r.queryTrendCount(ctx, "import_batches", "ib", "ib.uploaded_at", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	case "subscription_activation_count":
		where, args := subscriptionScopedWhere(actor, filters, tf, "s.active_from")
		data, err := r.queryTrendCount(ctx, "subscriptions", "s", "s.active_from", where, args, tf, metric, analyticsMetricLabel(metric))
		if err != nil {
			return ChartSeries{}, nil, 0, err
		}
		return data.Series[0], data.Table, data.Value, nil
	default:
		return ChartSeries{}, nil, 0, fmt.Errorf("metric analytics tidak didukung: %s", metric)
	}
}

func (r *Repository) metricValue(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, metric string) (float64, error) {
	switch metric {
	case "active_subscription_count":
		return r.countActiveSubscriptionsAtDate(ctx, actor, filters, tf.End.Add(-time.Nanosecond))
	case "owner_count", "outlet_count", "lead_count", "training_completed_count", "confirmed_closing_count", "closing_revenue_snapshot", "topup_revenue", "topup_transaction_count", "import_batch_count", "import_failed_batch_count", "import_committed_batch_count", "subscription_activation_count":
		_, _, value, err := r.metricTrend(ctx, actor, filters, tf, metric)
		return value, err
	default:
		return 0, fmt.Errorf("metric analytics tidak didukung: %s", metric)
	}
}

func (r *Repository) metricBoard(ctx context.Context, actor identity.User, req QueryRequest, tf ResolvedTimeFilter, metrics []string) ([]map[string]any, ChartSeries, float64, error) {
	comparison, baseline, err := resolveComparison(tf, req.Comparison, currentNowUTC())
	if err != nil {
		return nil, ChartSeries{}, 0, err
	}
	series := ChartSeries{Key: "current_value", Label: "Current Value"}
	table := make([]map[string]any, 0, len(metrics))
	total := 0.0
	for _, metric := range metrics {
		currentValue, err := r.metricValue(ctx, actor, req.Filters, tf, metric)
		if err != nil {
			return nil, ChartSeries{}, 0, err
		}
		baselineValue := 0.0
		if baseline != nil {
			baselineValue, err = r.metricValue(ctx, actor, req.Filters, *baseline, metric)
			if err != nil {
				return nil, ChartSeries{}, 0, err
			}
		}
		delta := round2(currentValue - baselineValue)
		deltaPct := percentDelta(currentValue, baselineValue)
		direction, statusValue := regionStatus(delta, analyticsMetricPolarity(metric))
		series.Points = append(series.Points, ChartPoint{X: analyticsMetricLabel(metric), Y: currentValue})
		table = append(table, map[string]any{
			"metric":         metric,
			"metric_label":   analyticsMetricLabel(metric),
			"current_value":  currentValue,
			"baseline_value": baselineValue,
			"delta":          delta,
			"delta_percent":  deltaPct,
			"direction":      direction,
			"status_value":   statusValue,
			"baseline_label": comparison.BaselineLabel,
		})
		total += currentValue
	}
	return table, series, total, nil
}

func (r *Repository) metricValueByRegion(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter, metric, field, value string) (float64, error) {
	field = strings.ToLower(strings.TrimSpace(field))
	switch field {
	case "province", "city":
	default:
		return 0, fmt.Errorf("field compare region tidak didukung: %s", field)
	}
	switch metric {
	case "owner_count":
		where, args := ownerScopedWhere(actor, filters, "o", tf, "o.created_at", false)
		where += " AND o." + field + " = ?"
		args = append(args, value)
		return r.scalarFloat(ctx, `SELECT COUNT(*) FROM owners o WHERE `+where, args...)
	case "outlet_count":
		where, args := outletScopedWhere(actor, filters, "ot", tf, "ot.created_at")
		where += " AND ot." + field + " = ?"
		args = append(args, value)
		return r.scalarFloat(ctx, `SELECT COUNT(*) FROM outlets ot WHERE `+where, args...)
	case "lead_count":
		where, args := leadScopedWhere(actor, filters, "cl", tf, "cl.created_at")
		query := `SELECT COUNT(*) FROM customer_leads cl LEFT JOIN owners o ON o.id = cl.owner_id WHERE ` + where + ` AND o.` + field + ` = ?`
		args = append(args, value)
		return r.scalarFloat(ctx, query, args...)
	case "confirmed_closing_count":
		where, args := closingScopedWhere(actor, filters, tf, "sc.confirmed_at")
		query := `SELECT COUNT(*) FROM sales_closings sc LEFT JOIN owners o ON o.id = sc.owner_id WHERE ` + where + ` AND sc.status = 'CONFIRMED' AND o.` + field + ` = ?`
		args = append(args, value)
		return r.scalarFloat(ctx, query, args...)
	case "topup_revenue":
		where, args := walletPaymentScopedWhere(actor, filters, tf, "wp.paid_at")
		query := `SELECT CAST(COALESCE(SUM(wp.amount), 0) AS DOUBLE) FROM wallet_payments wp LEFT JOIN owners o ON o.id = wp.owner_id WHERE ` + where + ` AND wp.status = 'PAID' AND o.` + field + ` = ?`
		args = append(args, value)
		return r.scalarFloat(ctx, query, args...)
	default:
		return 0, fmt.Errorf("metric region compare tidak didukung: %s", metric)
	}
}

func (r *Repository) defaultRegionComparisonSeries(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) ([]ComparisonSeriesRequest, error) {
	where, args := ownerScopedWhere(actor, filters, "o", tf, "o.created_at", false)
	query := `
		SELECT COALESCE(NULLIF(TRIM(o.province), ''), 'Tanpa Provinsi') AS province, COUNT(*) AS total
		FROM owners o
		WHERE ` + where + `
		GROUP BY province
		ORDER BY total DESC, province ASC
		LIMIT 5`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ComparisonSeriesRequest, 0)
	for rows.Next() {
		var province string
		var total float64
		if err := rows.Scan(&province, &total); err != nil {
			return nil, err
		}
		items = append(items, ComparisonSeriesRequest{
			Field: "province",
			Label: province,
			Value: province,
		})
	}
	return items, rows.Err()
}

func (r *Repository) subscriptionCohortSize(ctx context.Context, actor identity.User, filters FilterRequest, cohortStart, cohortEnd time.Time) (float64, error) {
	query := `
		SELECT COUNT(DISTINCT CONCAT(COALESCE(s.owner_id, 0), ':', COALESCE(s.outlet_id, 0)))
		FROM subscriptions s
		WHERE s.deleted_at IS NULL
			AND s.active_from >= ? AND s.active_from < ?
			AND ` + ownerVisibilityCondition(actor, "s.owner_id", filters)
	args := []any{cohortStart.Format("2006-01-02"), cohortEnd.Format("2006-01-02")}
	args = append(args, ownerVisibilityArgs(actor, filters)...)
	return r.scalarFloat(ctx, query, args...)
}

func (r *Repository) subscriptionCohortRetainedCount(ctx context.Context, actor identity.User, filters FilterRequest, cohortStart, cohortEnd, observationEnd time.Time) (float64, error) {
	query := `
		SELECT COUNT(DISTINCT CONCAT(COALESCE(s0.owner_id, 0), ':', COALESCE(s0.outlet_id, 0)))
		FROM subscriptions s0
		WHERE s0.deleted_at IS NULL
			AND s0.active_from >= ? AND s0.active_from < ?
			AND ` + ownerVisibilityCondition(actor, "s0.owner_id", filters) + `
			AND EXISTS (
				SELECT 1
				FROM subscriptions s1
				WHERE s1.deleted_at IS NULL
					AND s1.owner_id <=> s0.owner_id
					AND s1.outlet_id <=> s0.outlet_id
					AND s1.active_from <= ?
					AND s1.active_until >= ?
			)`
	args := []any{
		cohortStart.Format("2006-01-02"),
		cohortEnd.Format("2006-01-02"),
	}
	args = append(args, ownerVisibilityArgs(actor, filters)...)
	args = append(args, observationEnd.Format("2006-01-02"), observationEnd.Format("2006-01-02"))
	return r.scalarFloat(ctx, query, args...)
}

func (r *Repository) scalarFloat(ctx context.Context, query string, args ...any) (float64, error) {
	var value sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&value); err != nil {
		return 0, err
	}
	if !value.Valid {
		return 0, nil
	}
	return round2(value.Float64), nil
}

func percent(current, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return round2((current / total) * 100)
}

func percentDelta(current, baseline float64) float64 {
	if baseline == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return round2(((current - baseline) / baseline) * 100)
}

func clamp100(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (r *Repository) importQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	where, args := importBatchScopedWhere(actor, filters, tf, "ib.uploaded_at")
	query := `
		SELECT
			COUNT(*) AS total_batch_count,
			COALESCE(SUM(CASE WHEN ib.status IN ('VALIDATION_FAILED', 'COMMIT_FAILED') THEN 1 ELSE 0 END), 0) AS failed_batch_count,
			CAST(COALESCE(SUM(ib.invalid_rows), 0) AS DOUBLE) AS invalid_row_count,
			CAST(COALESCE(SUM(ib.total_rows), 0) AS DOUBLE) AS total_row_count
		FROM import_batches ib
		WHERE ` + where
	var totalBatches, failedBatches, invalidRows, totalRows float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&totalBatches, &failedBatches, &invalidRows, &totalRows); err != nil {
		return 0, nil, err
	}
	failedRate := percent(failedBatches, totalBatches)
	invalidRate := percent(invalidRows, totalRows)
	score := clamp100(round2(100 - (failedRate * 0.6) - (invalidRate * 0.4)))
	return score, map[string]any{
		"failed_batch_rate": failedRate,
		"invalid_row_rate":  invalidRate,
	}, nil
}

func (r *Repository) leadQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	totalLeads, err := r.metricValue(ctx, actor, filters, tf, "lead_count")
	if err != nil {
		return 0, nil, err
	}
	where, args := leadScopedWhere(actor, filters, "cl", tf, "cl.created_at")
	where += " AND cl.stage = 'INVALID'"
	invalidLeads, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM customer_leads cl WHERE `+where, args...)
	if err != nil {
		return 0, nil, err
	}
	invalidRate := percent(invalidLeads, totalLeads)
	score := clamp100(round2(100 - invalidRate))
	return score, map[string]any{
		"invalid_lead_rate": invalidRate,
	}, nil
}

func (r *Repository) salesQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	where, args := closingScopedWhere(actor, filters, tf, "sc.closed_at")
	totalClosing, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM sales_closings sc WHERE `+where, args...)
	if err != nil {
		return 0, nil, err
	}
	whereConfirmed, argsConfirmed := closingScopedWhere(actor, filters, tf, "sc.confirmed_at")
	whereConfirmed += " AND sc.status = 'CONFIRMED' AND sc.confirmed_at IS NOT NULL"
	confirmedClosing, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM sales_closings sc WHERE `+whereConfirmed, argsConfirmed...)
	if err != nil {
		return 0, nil, err
	}
	confirmRate := percent(confirmedClosing, totalClosing)
	return confirmRate, map[string]any{
		"confirmed_closing_rate": confirmRate,
	}, nil
}

func (r *Repository) walletQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	where, args := walletPaymentScopedWhere(actor, filters, tf, "wp.paid_at")
	totalPayment, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM wallet_payments wp WHERE `+where, args...)
	if err != nil {
		return 0, nil, err
	}
	paidWhere, paidArgs := walletPaymentScopedWhere(actor, filters, tf, "wp.paid_at")
	paidWhere += " AND wp.status = 'PAID'"
	paidPayment, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM wallet_payments wp WHERE `+paidWhere, paidArgs...)
	if err != nil {
		return 0, nil, err
	}
	paidRate := percent(paidPayment, totalPayment)
	return paidRate, map[string]any{
		"paid_payment_rate": paidRate,
	}, nil
}

func (r *Repository) subscriptionQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	whereRecon, argsRecon := reconciliationScopedWhere(actor, filters, tf, "sr.created_at")
	totalRecon, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM subscription_reconciliations sr WHERE `+whereRecon, argsRecon...)
	if err != nil {
		return 0, nil, err
	}
	openIssues, err := r.openReconciliationIssueCount(ctx, actor, filters, tf)
	if err != nil {
		return 0, nil, err
	}
	issueRate := percent(openIssues, totalRecon+openIssues)
	score := clamp100(round2(100 - issueRate))
	return score, map[string]any{
		"open_issue_rate": issueRate,
	}, nil
}

func (r *Repository) partnerQualityScore(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, map[string]any, error) {
	where, args := partnerCommissionScopedWhere(actor, filters, tf, "pc.created_at")
	totalCommission, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM partner_commissions pc WHERE `+where, args...)
	if err != nil {
		return 0, nil, err
	}
	paidWhere, paidArgs := partnerCommissionScopedWhere(actor, filters, tf, "pc.paid_at")
	paidWhere += " AND pc.status = 'PAID' AND pc.paid_at IS NOT NULL"
	paidCommission, err := r.scalarFloat(ctx, `SELECT COUNT(*) FROM partner_commissions pc WHERE `+paidWhere, paidArgs...)
	if err != nil {
		return 0, nil, err
	}
	paidRate := percent(paidCommission, totalCommission)
	return paidRate, map[string]any{
		"paid_commission_rate": paidRate,
	}, nil
}

func (r *Repository) openReconciliationIssueCount(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, error) {
	where, args := reconciliationIssueScopedWhere(actor, filters, tf, "ri.detected_at")
	where += " AND ri.status = 'OPEN'"
	return r.scalarFloat(ctx, `SELECT COUNT(*) FROM reconciliation_issues ri WHERE `+where, args...)
}

func (r *Repository) unpaidCommissionCount(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, error) {
	where, args := partnerCommissionScopedWhere(actor, filters, tf, "pc.created_at")
	where += " AND pc.status IN ('PENDING', 'APPROVED')"
	return r.scalarFloat(ctx, `SELECT COUNT(*) FROM partner_commissions pc WHERE `+where, args...)
}

func (r *Repository) importFailedBatchCount(ctx context.Context, actor identity.User, filters FilterRequest, tf ResolvedTimeFilter) (float64, error) {
	where, args := importBatchScopedWhere(actor, filters, tf, "ib.uploaded_at")
	where += " AND ib.status IN ('VALIDATION_FAILED', 'COMMIT_FAILED')"
	return r.scalarFloat(ctx, `SELECT COUNT(*) FROM import_batches ib WHERE `+where, args...)
}
