package seeder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"backend_crm_piposmart/internal/platform/config"
	"backend_crm_piposmart/internal/platform/factory"

	_ "github.com/go-sql-driver/mysql"
)

const (
	ModeMaster = "master"
	ModeDemo   = "demo"
)

type Options struct {
	Mode      string
	Preset    string
	Seed      int64
	From      time.Time
	To        time.Time
	AsOf      time.Time
	Scale     int
	Variation float64
}

func Parse(args []string) (Options, error) {
	if len(args) == 0 {
		return Options{}, errors.New(seedUsage)
	}

	options := Options{
		Mode: args[0],
	}

	switch options.Mode {
	case ModeMaster:
		if len(args) > 1 {
			return Options{}, fmt.Errorf("seed master tidak menerima argumen tambahan\n\n%s", seedUsage)
		}
	case ModeDemo:
		today := time.Now().UTC().Truncate(24 * time.Hour)
		options.Preset = "minimal"
		options.Seed = 1
		options.From = today
		options.To = today
		options.AsOf = today
		options.Variation = 0.5
		for _, arg := range args[1:] {
			key, value, found := strings.Cut(arg, "=")
			if !found || !strings.HasPrefix(key, "--") {
				return Options{}, fmt.Errorf("argumen seed demo tidak valid: %s", arg)
			}
			switch key {
			case "--preset":
				options.Preset = strings.TrimSpace(value)
			case "--seed":
				seed, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return Options{}, fmt.Errorf("--seed harus angka: %w", err)
				}
				options.Seed = seed
			case "--from":
				from, err := time.Parse("2006-01-02", value)
				if err != nil {
					return Options{}, fmt.Errorf("--from harus format YYYY-MM-DD: %w", err)
				}
				options.From = from
			case "--to":
				to, err := time.Parse("2006-01-02", value)
				if err != nil {
					return Options{}, fmt.Errorf("--to harus format YYYY-MM-DD: %w", err)
				}
				options.To = to
			case "--as-of":
				asOf, err := time.Parse("2006-01-02", value)
				if err != nil {
					return Options{}, fmt.Errorf("--as-of harus format YYYY-MM-DD: %w", err)
				}
				options.From = asOf
				options.To = asOf
				options.AsOf = asOf
			case "--scale":
				scale, err := strconv.Atoi(value)
				if err != nil {
					return Options{}, fmt.Errorf("--scale harus angka: %w", err)
				}
				options.Scale = scale
			case "--variation":
				variation, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return Options{}, fmt.Errorf("--variation harus angka desimal 0 sampai 1: %w", err)
				}
				options.Variation = variation
			default:
				return Options{}, fmt.Errorf("argumen seed demo tidak dikenal: %s", key)
			}
		}
		if options.From.After(options.To) {
			return Options{}, errors.New("--from tidak boleh lebih besar dari --to")
		}
		if options.Variation < 0 || options.Variation > 1 {
			return Options{}, errors.New("--variation harus bernilai antara 0 sampai 1")
		}
		options.AsOf = options.To
		if options.Preset != "minimal" && options.Preset != "large" {
			return Options{}, fmt.Errorf("preset demo %q belum tersedia; gunakan minimal atau large", options.Preset)
		}
		if options.Preset == "large" {
			if options.Scale == 0 {
				options.Scale = defaultLargeSeedScale
			}
			if _, err := largeSeedOwnerCountForScale(options.Scale); err != nil {
				return Options{}, err
			}
		}
		if options.Preset != "large" && options.Scale != 0 {
			return Options{}, errors.New("--scale hanya didukung untuk seed demo --preset=large")
		}
	default:
		return Options{}, fmt.Errorf("mode seed %q tidak dikenal\n\n%s", options.Mode, seedUsage)
	}

	return options, nil
}

func Run(ctx context.Context, cfg config.Config, args []string, output io.Writer, logger *slog.Logger) error {
	options, err := Parse(args)
	if err != nil {
		return err
	}
	if err := ValidateEnvironment(cfg, options); err != nil {
		return err
	}

	db, err := sql.Open("mysql", cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("buka database seed: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database seed: %w", err)
	}

	runID, err := startSeedRun(ctx, db, cfg, options)
	if err != nil {
		return err
	}
	err = runWithTransaction(ctx, db, options)
	if finishErr := finishSeedRun(ctx, db, runID, err); finishErr != nil && err == nil {
		err = finishErr
	}
	if err != nil {
		return err
	}

	logger.Info("seed completed",
		slog.String("mode", options.Mode),
		slog.String("preset", options.Preset),
		slog.Int64("seed", options.Seed),
		slog.Int("scale", options.Scale),
		slog.String("from", options.From.Format("2006-01-02")),
		slog.String("to", options.To.Format("2006-01-02")),
		slog.Float64("variation", options.Variation),
	)
	fmt.Fprintf(output, "seed %s selesai", options.Mode)
	if options.Mode == ModeDemo {
		fmt.Fprintf(output, " (preset=%s, seed=%d, from=%s, to=%s", options.Preset, options.Seed, options.From.Format("2006-01-02"), options.To.Format("2006-01-02"))
		if options.Preset == "large" {
			fmt.Fprintf(output, ", scale=%d, variation=%.2f", options.Scale, options.Variation)
		}
		fmt.Fprintf(output, ")")
	}
	fmt.Fprintln(output)
	return nil
}

func ValidateEnvironment(cfg config.Config, options Options) error {
	if options.Mode == ModeDemo && cfg.App.IsProduction() {
		return errors.New("demo seeder ditolak pada environment production")
	}
	return nil
}

func runWithTransaction(ctx context.Context, db *sql.DB, options Options) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mulai transaksi seed: %w", err)
	}
	defer tx.Rollback()

	if err := seedMaster(ctx, tx); err != nil {
		return err
	}
	if options.Mode == ModeDemo {
		if options.Preset == "large" {
			if err := seedDemoLarge(ctx, tx, options); err != nil {
				return err
			}
		} else {
			if err := seedDemoMinimal(ctx, tx, options); err != nil {
				return err
			}
		}
		if err := seedTargetKpiRankingScenario(ctx, tx, options); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

func startSeedRun(ctx context.Context, db *sql.DB, cfg config.Config, options Options) (int64, error) {
	checksum := checksumFor(options)
	result, err := db.ExecContext(ctx, `
		INSERT INTO seed_runs (name, preset, seed_value, as_of_date, environment, checksum, status, started_at)
		VALUES (?, ?, ?, ?, ?, ?, 'RUNNING', ?)`,
		options.Mode,
		nullableString(options.Preset),
		nullableInt64(options.Seed),
		nullableDate(options.To),
		cfg.App.Environment,
		checksum,
		time.Now().UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("catat seed run: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ambil seed run id: %w", err)
	}
	return id, nil
}

func finishSeedRun(ctx context.Context, db *sql.DB, runID int64, seedErr error) error {
	status := "SUCCESS"
	var message sql.NullString
	if seedErr != nil {
		status = "FAILED"
		message = sql.NullString{String: seedErr.Error(), Valid: true}
	}

	_, err := db.ExecContext(ctx, `
		UPDATE seed_runs
		SET status = ?, finished_at = ?, error_message = ?
		WHERE id = ?`,
		status, time.Now().UTC(), message, runID,
	)
	if err != nil {
		return fmt.Errorf("update seed run: %w", err)
	}
	return nil
}

func seedMaster(ctx context.Context, tx *sql.Tx) error {
	if err := seedRoles(ctx, tx); err != nil {
		return err
	}
	if err := seedPermissions(ctx, tx); err != nil {
		return err
	}
	if err := seedRolePermissions(ctx, tx); err != nil {
		return err
	}
	if err := seedRemarkReasons(ctx, tx); err != nil {
		return err
	}
	if err := seedPartnerTypes(ctx, tx); err != nil {
		return err
	}
	if err := seedMetricCodes(ctx, tx); err != nil {
		return err
	}
	if err := seedPackagesAndPlans(ctx, tx); err != nil {
		return err
	}
	if err := seedPromotions(ctx, tx); err != nil {
		return err
	}
	if err := seedCommissionRulesDemo(ctx, tx); err != nil {
		return err
	}
	return nil
}

func seedRoles(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		code, name, description string
	}{
		{"ADMIN", "Admin", "Aktor kantor dengan akses operasional penuh."},
		{"SUPERVISOR", "Supervisor", "Kepala tim marketing yang mengelola Sales dan distribusi lead."},
		{"SALES", "Sales", "Staff sales yang melakukan call customer, follow-up, dan closing."},
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO roles (code, name, description, is_system)
			VALUES (?, ?, ?, TRUE)
			ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), is_system = TRUE`,
			row.code, row.name, row.description,
		); err != nil {
			return fmt.Errorf("seed role %s: %w", row.code, err)
		}
	}
	return nil
}

func seedPermissions(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		code, domain, action, description string
	}{
		{"users.read", "users", "read", "Melihat user CRM."},
		{"users.manage_sales", "users", "manage_sales", "Membuat dan mengelola akun Sales."},
		{"users.manage_all", "users", "manage_all", "Mengelola seluruh role user."},
		{"owners.manage", "owners", "manage", "Mengelola owner dan outlet."},
		{"leads.assign", "leads", "assign", "Mendistribusikan lead kepada Sales."},
		{"leads.work", "leads", "work", "Mengerjakan lead yang menjadi PIC."},
		{"catalog.manage", "catalog", "manage", "Mengelola paket, plan, dan promo."},
		{"reports.read_all", "reports", "read_all", "Melihat seluruh laporan."},
		{"reports.read_own", "reports", "read_own", "Melihat laporan pribadi Sales."},
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO permissions (code, domain, action, description)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				domain = VALUES(domain),
				action = VALUES(action),
				description = VALUES(description)`,
			row.code, row.domain, row.action, row.description,
		); err != nil {
			return fmt.Errorf("seed permission %s: %w", row.code, err)
		}
	}
	return nil
}

func seedRolePermissions(ctx context.Context, tx *sql.Tx) error {
	adminPermissions := []string{
		"users.read", "users.manage_sales", "users.manage_all", "owners.manage",
		"leads.assign", "leads.work", "catalog.manage", "reports.read_all",
	}
	supervisorPermissions := []string{
		"users.read", "users.manage_sales", "owners.manage", "leads.assign",
		"leads.work", "catalog.manage", "reports.read_all",
	}
	salesPermissions := []string{"leads.work", "reports.read_own"}

	assignments := map[string][]string{
		"ADMIN":      adminPermissions,
		"SUPERVISOR": supervisorPermissions,
		"SALES":      salesPermissions,
	}
	for roleCode, permissions := range assignments {
		roleID, err := lookupID(ctx, tx, "roles", "code", roleCode)
		if err != nil {
			return err
		}
		for _, permissionCode := range permissions {
			permissionID, err := lookupID(ctx, tx, "permissions", "code", permissionCode)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT IGNORE INTO role_permissions (role_id, permission_id)
				VALUES (?, ?)`,
				roleID, permissionID,
			); err != nil {
				return fmt.Errorf("seed role permission %s:%s: %w", roleCode, permissionCode, err)
			}
		}
	}
	return nil
}

func seedRemarkReasons(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		score              int
		code, label, desc  string
		followUpDays       sql.NullInt64
		releasesAssignment bool
	}{
		{0, "INVALID", "Invalid / Tidak Potensial", "Customer menolak keras, tidak valid, atau tidak relevan.", sql.NullInt64{}, true},
		{1, "POSSIBLE", "Kemungkinan", "Masih ada peluang dan perlu follow-up ulang.", sql.NullInt64{Int64: 7, Valid: true}, false},
		{2, "POTENTIAL", "Potensial", "Customer tertarik dan bisa diarahkan ke demo/training.", sql.NullInt64{Int64: 3, Valid: true}, false},
		{3, "CLOSING", "Closing", "Customer sepakat membeli paket langganan.", sql.NullInt64{}, false},
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO remark_reasons
				(score, code, label, description, default_follow_up_days, releases_assignment, active)
			VALUES (?, ?, ?, ?, ?, ?, TRUE)
			ON DUPLICATE KEY UPDATE
				score = VALUES(score),
				label = VALUES(label),
				description = VALUES(description),
				default_follow_up_days = VALUES(default_follow_up_days),
				releases_assignment = VALUES(releases_assignment),
				active = TRUE`,
			row.score, row.code, row.label, row.desc, row.followUpDays, row.releasesAssignment,
		); err != nil {
			return fmt.Errorf("seed remark reason %s: %w", row.code, err)
		}
	}
	return nil
}

// Sprint 15a §5: 3 partner types only (down from the earlier 6 demo types) — REFERRAL,
// PARTNERSHIP, STRATEGIC, matching data_admin/Ringkasan_Komisi_Piposmart.pdf. All FIXED: the
// company applies a flat FIXED commission model generally, and the real per-plan nominal comes
// from commission_rules (seedCommissionRulesDemo), not from these flat fallback values.
func seedPartnerTypes(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		code, name, mode, description string
		value                         string
	}{
		{"REFERRAL", "Referral", "FIXED", "Mitra perujuk calon pelanggan (referral).", "0.00"},
		{"PARTNERSHIP", "Partnership", "FIXED", "Mitra kerja sama (partnership).", "0.00"},
		{"STRATEGIC", "Strategic", "FIXED", "Mitra strategis.", "0.00"},
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_types (code, name, commission_mode, commission_value, description, active)
			VALUES (?, ?, ?, ?, ?, TRUE)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				commission_mode = VALUES(commission_mode),
				commission_value = VALUES(commission_value),
				description = VALUES(description),
				active = TRUE`,
			row.code, row.name, row.mode, row.value, row.description,
		); err != nil {
			return fmt.Errorf("seed partner type %s: %w", row.code, err)
		}
	}
	// Deactivate the 6 legacy demo types (SUPPLIER/DISTRIBUTOR/AGENT/REFERRAL_PARTNER/
	// REFERRAL_REGULAR/REFERRAL_STRATEGIC) rather than delete them — a prior seed run's
	// partners may still reference them via partner_type_id FK, and deleting would break
	// re-seeding on an existing dev DB. Deactivating keeps them out of active-type listings
	// while leaving history intact.
	legacyCodes := []string{"SUPPLIER", "DISTRIBUTOR", "AGENT", "REFERRAL_PARTNER", "REFERRAL_REGULAR", "REFERRAL_STRATEGIC"}
	for _, code := range legacyCodes {
		if _, err := tx.ExecContext(ctx, `UPDATE partner_types SET active = FALSE WHERE code = ?`, code); err != nil {
			return fmt.Errorf("deactivate legacy partner type %s: %w", code, err)
		}
	}
	return nil
}

// seedCommissionRulesDemo seeds the exact FIXED commission_rules matrix from
// data_admin/Ringkasan_Komisi_Piposmart.pdf: 7 plans x 3 partner types = 21 rules. Per the mitra
// MOU, these amounts hold indefinitely until explicitly superseded — update-by-versioning
// (new EffectiveFrom + close out the old rule's EffectiveTo), never an in-place nominal edit,
// so a partner's already-earned commissions (snapshotted onto partner_commissions at calc time)
// are never retroactively changed by a later rate change.
func seedCommissionRulesDemo(ctx context.Context, tx *sql.Tx) error {
	matrix := []struct {
		planCode                         string
		referral, partnership, strategic string
	}{
		{"BASIC_12_MONTHS", "120000.00", "150000.00", "240000.00"},
		{"BUSINESS_12_MONTHS", "180000.00", "210000.00", "320000.00"},
		{"BUSINESS_18_MONTHS", "270000.00", "315000.00", "480000.00"},
		{"BUSINESS_24_MONTHS", "360000.00", "420000.00", "640000.00"},
		{"PRO_12_MONTHS", "220000.00", "250000.00", "400000.00"},
		{"PRO_18_MONTHS", "330000.00", "375000.00", "600000.00"},
		{"PRO_24_MONTHS", "440000.00", "500000.00", "800000.00"},
	}
	for _, row := range matrix {
		planID, err := lookupID(ctx, tx, "subscription_plans", "code", row.planCode)
		if err != nil {
			return fmt.Errorf("seed commission rule lookup plan %s: %w", row.planCode, err)
		}
		for typeCode, value := range map[string]string{
			"REFERRAL":    row.referral,
			"PARTNERSHIP": row.partnership,
			"STRATEGIC":   row.strategic,
		} {
			partnerTypeID, err := lookupID(ctx, tx, "partner_types", "code", typeCode)
			if err != nil {
				return fmt.Errorf("seed commission rule lookup partner type %s: %w", typeCode, err)
			}
			var existingID int64
			err = tx.QueryRowContext(ctx, `
				SELECT id FROM commission_rules
				WHERE partner_type_id = ? AND plan_id = ? AND effective_from = '2026-07-01'`,
				partnerTypeID, planID).Scan(&existingID)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO commission_rules
						(partner_type_id, plan_id, mode, value, effective_from, active)
					VALUES (?, ?, 'FIXED', ?, '2026-07-01', TRUE)`,
					partnerTypeID, planID, value,
				); err != nil {
					return fmt.Errorf("seed commission rule %s/%s: %w", typeCode, row.planCode, err)
				}
			case err != nil:
				return fmt.Errorf("seed commission rule lookup existing %s/%s: %w", typeCode, row.planCode, err)
			default:
				if _, err := tx.ExecContext(ctx, `
					UPDATE commission_rules SET mode = 'FIXED', value = ?, active = TRUE
					WHERE id = ?`, value, existingID,
				); err != nil {
					return fmt.Errorf("seed commission rule update %s/%s: %w", typeCode, row.planCode, err)
				}
			}
		}
	}
	return nil
}

func seedMetricCodes(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		code, name, unit, description string
	}{
		{"CALL_CUSTOMER_COUNT", "Jumlah Call Customer", "count", "Jumlah interaksi call/chat customer."},
		{"TRAINING_COUNT", "Jumlah Training", "count", "Jumlah training/demo aplikasi."},
		{"CONFIRMED_CLOSING_COUNT", "Jumlah Closing Confirmed", "count", "Jumlah closing yang sudah terkonfirmasi."},
		{"CONFIRMED_CLOSING_AMOUNT", "Nominal Closing Confirmed", "IDR", "Nominal closing yang sudah terkonfirmasi."},
		{"PARTNER_CALL_COUNT", "Jumlah Call Mitra", "count", "Jumlah interaksi mitra."},
	}
	for _, row := range rows {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO metric_codes (code, name, unit, description, active)
			VALUES (?, ?, ?, ?, TRUE)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				unit = VALUES(unit),
				description = VALUES(description),
				active = TRUE`,
			row.code, row.name, row.unit, row.description,
		); err != nil {
			return fmt.Errorf("seed metric code %s: %w", row.code, err)
		}
	}
	return nil
}

func seedPackagesAndPlans(ctx context.Context, tx *sql.Tx) error {
	packages := []struct {
		code, name, description string
		order                   int
		baseMonthlyPrice        int
	}{
		{"BASIC", "Basic", "Paket dasar Piposmart untuk operasional laundry awal.", 1, 99000},
		{"BUSINESS", "Business", "Paket bisnis untuk laundry yang membutuhkan fitur lebih lengkap.", 2, 149000},
		{"PRO", "Pro", "Paket lanjutan untuk operasional laundry yang lebih kompleks.", 3, 199000},
	}
	tenures := []int{1, 9, 12, 18, 24}
	// pdfPrices overrides the formula price for the 7 plans the partner commission summary
	// (data_admin/Ringkasan_Komisi_Piposmart.pdf) actually prices — commission is a FIXED amount
	// tied to these exact plan prices per the mitra MOU, so the plan price must match the PDF
	// exactly for these tenures. Tenures the PDF doesn't cover (1, 9 months) keep the old formula.
	pdfPrices := map[string]int{
		"BASIC_12_MONTHS":    858000,
		"BUSINESS_12_MONTHS": 1298000,
		"BUSINESS_18_MONTHS": 1999000,
		"BUSINESS_24_MONTHS": 2596000,
		"PRO_12_MONTHS":      1688000,
		"PRO_18_MONTHS":      2688000,
		"PRO_24_MONTHS":      3368000,
	}
	for _, pkg := range packages {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subscription_packages (code, name, level_order, description, active)
			VALUES (?, ?, ?, ?, TRUE)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				level_order = VALUES(level_order),
				description = VALUES(description),
				active = TRUE,
				deleted_at = NULL`,
			pkg.code, pkg.name, pkg.order, pkg.description,
		); err != nil {
			return fmt.Errorf("seed package %s: %w", pkg.code, err)
		}

		packageID, err := lookupID(ctx, tx, "subscription_packages", "code", pkg.code)
		if err != nil {
			return err
		}
		for _, tenure := range tenures {
			price := pkg.baseMonthlyPrice * tenure
			if tenure >= 12 {
				price = price * 95 / 100
			}
			code := fmt.Sprintf("%s_%02d_MONTHS", pkg.code, tenure)
			if pdfPrice, ok := pdfPrices[code]; ok {
				price = pdfPrice
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO subscription_plans
					(package_id, code, name, tenure_months, duration_days, price, currency, effective_from, active)
				VALUES (?, ?, ?, ?, ?, ?, 'IDR', '2026-07-01', TRUE)
				ON DUPLICATE KEY UPDATE
					package_id = VALUES(package_id),
					name = VALUES(name),
					tenure_months = VALUES(tenure_months),
					duration_days = VALUES(duration_days),
					price = VALUES(price),
					currency = 'IDR',
					effective_from = VALUES(effective_from),
					active = TRUE,
					deleted_at = NULL`,
				packageID, code, fmt.Sprintf("%s %d Bulan", pkg.name, tenure), tenure, tenure*30, price,
			); err != nil {
				return fmt.Errorf("seed plan %s: %w", code, err)
			}
		}
	}
	return nil
}

func seedPromotions(ctx context.Context, tx *sql.Tx) error {
	promotions := []struct {
		code, name, promoType, chargeType, additionalCharge, description string
		priority                                                         int
		planCodes                                                        []string
		benefitType                                                      string
		durationDays                                                     sql.NullInt64
		quantity                                                         sql.NullInt64
		benefitDescription                                               string
	}{
		{
			code: "FREE_1_MONTH_BUSINESS_12", name: "Business 12 + 1 Bulan", promoType: "FREE_DURATION",
			chargeType: "FREE", additionalCharge: "0.00", priority: 10,
			description:        "Promo gratis tambahan 1 bulan untuk Business 12 bulan.",
			planCodes:          []string{"BUSINESS_12_MONTHS"},
			benefitType:        "FREE_DURATION",
			durationDays:       sql.NullInt64{Int64: 30, Valid: true},
			benefitDescription: "Tambahan 30 hari paket Business.",
		},
		{
			code: "FREE_2_MONTHS_PRO_24", name: "Pro 24 + 2 Bulan", promoType: "FREE_DURATION",
			chargeType: "FREE", additionalCharge: "0.00", priority: 10,
			description:        "Promo gratis tambahan 2 bulan untuk Pro 24 bulan.",
			planCodes:          []string{"PRO_24_MONTHS"},
			benefitType:        "FREE_DURATION",
			durationDays:       sql.NullInt64{Int64: 60, Valid: true},
			benefitDescription: "Tambahan 60 hari paket Pro.",
		},
		{
			code: "PRO_12_ANDROID_POS_BUNDLE", name: "Pro 12 Bulan Bonus Alat POS", promoType: "DEVICE_BUNDLE",
			chargeType: "PAID", additionalCharge: "1500000.00", priority: 50,
			description:        "Promo berbayar bundle POS Android dan 20 kertas thermal.",
			planCodes:          []string{"PRO_12_MONTHS"},
			benefitType:        "DEVICE",
			quantity:           sql.NullInt64{Int64: 1, Valid: true},
			benefitDescription: "POS Android + 20 kertas thermal.",
		},
		// Sprint 15a §4b — second promotion eligible for PRO_12_MONTHS, so a demo closing/order can
		// stack it together with PRO_12_ANDROID_POS_BUNDLE and exercise the new multi-promotion path
		// (sales_closing_promotions / subscription_order_promotions), not just single-promotion.
		{
			code: "FREE_ONBOARDING_PRO_12", name: "Pro 12 Bulan Sesi Onboarding Gratis", promoType: "FREE_SERVICE",
			chargeType: "FREE", additionalCharge: "0.00", priority: 20,
			description:        "Promo gratis sesi onboarding untuk Pro 12 bulan, bisa digabung dengan promo lain.",
			planCodes:          []string{"PRO_12_MONTHS"},
			benefitType:        "SERVICE",
			quantity:           sql.NullInt64{Int64: 1, Valid: true},
			benefitDescription: "1x sesi onboarding aplikasi bersama tim support.",
		},
	}

	for _, promo := range promotions {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO promotions
				(code, name, promotion_type, charge_type, additional_charge, priority, description, effective_from, active)
			VALUES (?, ?, ?, ?, ?, ?, ?, '2026-07-01', TRUE)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				promotion_type = VALUES(promotion_type),
				charge_type = VALUES(charge_type),
				additional_charge = VALUES(additional_charge),
				priority = VALUES(priority),
				description = VALUES(description),
				effective_from = VALUES(effective_from),
				active = TRUE,
				deleted_at = NULL`,
			promo.code, promo.name, promo.promoType, promo.chargeType, promo.additionalCharge, promo.priority, promo.description,
		); err != nil {
			return fmt.Errorf("seed promotion %s: %w", promo.code, err)
		}

		promotionID, err := lookupID(ctx, tx, "promotions", "code", promo.code)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM promotion_plan_eligibilities WHERE promotion_id = ?", promotionID); err != nil {
			return fmt.Errorf("reset promotion eligibility %s: %w", promo.code, err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM promotion_benefits WHERE promotion_id = ?", promotionID); err != nil {
			return fmt.Errorf("reset promotion benefit %s: %w", promo.code, err)
		}
		for _, planCode := range promo.planCodes {
			planID, err := lookupID(ctx, tx, "subscription_plans", "code", planCode)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO promotion_plan_eligibilities (promotion_id, plan_id)
				VALUES (?, ?)`,
				promotionID, planID,
			); err != nil {
				return fmt.Errorf("seed promotion eligibility %s:%s: %w", promo.code, planCode, err)
			}
		}
		var packageID sql.NullInt64
		if strings.Contains(promo.code, "BUSINESS") {
			id, err := lookupID(ctx, tx, "subscription_packages", "code", "BUSINESS")
			if err != nil {
				return err
			}
			packageID = sql.NullInt64{Int64: id, Valid: true}
		}
		if strings.Contains(promo.code, "PRO") {
			id, err := lookupID(ctx, tx, "subscription_packages", "code", "PRO")
			if err != nil {
				return err
			}
			packageID = sql.NullInt64{Int64: id, Valid: true}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO promotion_benefits
				(promotion_id, benefit_type, package_id, duration_days, quantity, description, metadata_json)
			VALUES (?, ?, ?, ?, ?, ?, JSON_OBJECT())`,
			promotionID, promo.benefitType, packageID, promo.durationDays, promo.quantity, promo.benefitDescription,
		); err != nil {
			return fmt.Errorf("seed promotion benefit %s: %w", promo.code, err)
		}
	}
	return nil
}

func seedDemoMinimal(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)
	users := []factory.User{
		fake.BuildUser("ADMIN", 1),
		fake.BuildUser("SUPERVISOR", 1),
		fake.BuildUser("SALES", 1),
		fake.BuildUser("SALES", 2),
	}
	for _, user := range users {
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return err
		}
	}

	leadIDs := []int64{}
	ownerIDs := []int64{}
	for ownerIndex := 1; ownerIndex <= 4; ownerIndex++ {
		owner := fake.BuildOwner(ownerIndex)
		ownerID, err := fake.CreateOwner(ctx, tx, owner)
		if err != nil {
			return err
		}
		ownerIDs = append(ownerIDs, ownerID)

		outletCount := 1
		if ownerIndex == 2 {
			outletCount = 3
		}
		var firstOutletID int64
		for outletIndex := 1; outletIndex <= outletCount; outletIndex++ {
			outlet := fake.BuildOutlet(owner.Code, outletIndex, owner)
			outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
			if err != nil {
				return err
			}
			if outletIndex == 1 {
				firstOutletID = outletID
			}
		}

		salesEmail := users[2].Email
		if ownerIndex%2 == 0 {
			salesEmail = users[3].Email
		}
		lead := fake.BuildLead(owner.Code, 1, salesEmail)
		leadID, err := fake.CreateLead(ctx, tx, ownerID, firstOutletID, lead)
		if err != nil {
			return err
		}
		leadIDs = append(leadIDs, leadID)
		remarkScore := ownerIndex - 1
		interaction := fake.BuildInteraction(ownerIndex, remarkScore)
		if _, err := fake.CreateInteraction(ctx, tx, leadID, interaction); err != nil {
			return err
		}
		if remarkScore == 2 {
			training := fake.BuildTrainingReport(ownerIndex, false)
			if _, err := fake.CreateTrainingReport(ctx, tx, leadID, training); err != nil {
				return err
			}
		}
	}

	closingScenarios := []factory.Closing{
		fake.BuildClosing(1, "BASIC_01_MONTHS", "", "PENDING_RECONCILIATION"),
		fake.BuildClosing(2, "BUSINESS_12_MONTHS", "FREE_1_MONTH_BUSINESS_12", "CONFIRMED"),
		fake.BuildClosing(3, "PRO_12_MONTHS", "PRO_12_ANDROID_POS_BUNDLE", "PENDING_RECONCILIATION"),
	}
	// Sprint 15a §4b — stack a second promotion on the Pro 12 closing to demo multi-promotion
	// (sales_closing_promotions); the linked subscription order below inherits both automatically.
	closingScenarios[2].PromotionCodes = []string{"PRO_12_ANDROID_POS_BUNDLE", "FREE_ONBOARDING_PRO_12"}
	closingIDs := []int64{}
	for index, closing := range closingScenarios {
		if index >= len(leadIDs) {
			break
		}
		closingID, err := fake.CreateClosing(ctx, tx, leadIDs[index], closing)
		if err != nil {
			return err
		}
		closingIDs = append(closingIDs, closingID)
	}

	if len(ownerIDs) >= 2 {
		usedTopup := fake.BuildWalletTopup(1, "2000000.00")
		usedTopup.ExternalReference = "DEMO-TOPUP-USED-OWNER-001"
		usedTopup.IdempotencyKey = "demo:topup:used-owner-001"
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[0], usedTopup); err != nil {
			return err
		}
		usedDebit := fake.BuildWalletDebit(1, "500000.00")
		usedDebit.ExternalReference = "DEMO-DEBIT-USED-OWNER-001"
		usedDebit.IdempotencyKey = "demo:debit:used-owner-001"
		if _, err := fake.CreateWalletDebit(ctx, tx, ownerIDs[0], usedDebit); err != nil {
			return err
		}

		unusedTopup := fake.BuildWalletTopup(2, "1250000.00")
		unusedTopup.ExternalReference = "DEMO-TOPUP-UNUSED-OWNER-002"
		unusedTopup.IdempotencyKey = "demo:topup:unused-owner-002"
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[1], unusedTopup); err != nil {
			return err
		}

		// Sprint 15a §2/§3 — Top Up lifecycle variety (PENDING/REJECTED) and the Transfer they can
		// be matched against, so the demo dataset actually exercises the new lifecycle/Transfer UI
		// instead of only ever showing the terminal ACCEPTED state.
		pendingTopup := fake.BuildWalletTopup(3, "750000.00")
		pendingTopup.Status = "PENDING"
		pendingTopup.ExternalReference = "DEMO-TOPUP-PENDING-OWNER-001"
		pendingTopup.IdempotencyKey = "demo:topup:pending-owner-001"
		pendingTopup.Note = "Demo Sprint 15a: top-up menunggu verifikasi transfer"
		pendingPaymentID, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[0], pendingTopup)
		if err != nil {
			return err
		}
		suggestedTransfer := fake.BuildTransfer(3, "750000.00", "SUGGESTED")
		suggestedTransfer.TransferDate = pendingTopup.PaidAt
		suggestedTransfer.Note = "Demo Sprint 15a: transfer disarankan cocok dengan top-up PENDING"
		suggestedTransfer.MatchedWalletPaymentID = sql.NullInt64{Int64: pendingPaymentID, Valid: true}
		suggestedTransfer.ExternalReference = "DEMO-TRF-SUGGESTED-OWNER-001"
		if _, err := fake.CreateTransfer(ctx, tx, ownerIDs[0], suggestedTransfer); err != nil {
			return err
		}

		rejectedTopup := fake.BuildWalletTopup(4, "300000.00")
		rejectedTopup.Status = "REJECTED"
		rejectedTopup.RejectNote = "Demo Sprint 15a: ditolak, nominal transfer tidak sesuai"
		rejectedTopup.ExternalReference = "DEMO-TOPUP-REJECTED-OWNER-002"
		rejectedTopup.IdempotencyKey = "demo:topup:rejected-owner-002"
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[1], rejectedTopup); err != nil {
			return err
		}
		rejectedMatchTransfer := fake.BuildTransfer(4, "300000.00", "REJECTED_MATCH")
		rejectedMatchTransfer.TransferDate = rejectedTopup.PaidAt
		rejectedMatchTransfer.Note = "Demo Sprint 15a: kecocokan ditolak admin, nominal tidak sama"
		rejectedMatchTransfer.ExternalReference = "DEMO-TRF-REJECTED-OWNER-002"
		if _, err := fake.CreateTransfer(ctx, tx, ownerIDs[1], rejectedMatchTransfer); err != nil {
			return err
		}

		unmatchedTransfer := fake.BuildTransfer(5, "500000.00", "UNMATCHED")
		unmatchedTransfer.Note = "Demo Sprint 15a: transfer baru masuk, belum ada saran kecocokan"
		unmatchedTransfer.ExternalReference = "DEMO-TRF-UNMATCHED-OWNER-004"
		if _, err := fake.CreateTransfer(ctx, tx, ownerIDs[3], unmatchedTransfer); err != nil {
			return err
		}
	}

	if len(ownerIDs) >= 4 && len(closingIDs) >= 3 {
		aprilTopup := fake.BuildWalletTopup(10, "4500000.00")
		aprilTopup.ExternalReference = "DEMO-TOPUP-APRIL-OWNER-003"
		aprilTopup.IdempotencyKey = "demo:topup:april-owner-003"
		aprilTopup.PaidAt = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
		aprilTopup.Note = "Demo Sprint 10: top-up April, saldo dipakai beli subscription Juli"
		aprilPaymentID, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[2], aprilTopup)
		if err != nil {
			return err
		}
		julyOrder := fake.BuildSubscriptionOrder(10, "PRO_12_MONTHS", "", sql.NullInt64{Int64: closingIDs[2], Valid: true})
		julyOrder.ExternalReference = "DEMO-SUB-JULY-OWNER-003"
		julyOrder.IdempotencyKey = "demo:subscription-order:april-topup-july-purchase-owner-003"
		julyOrder.PurchasedAt = time.Date(2026, 7, 10, 13, 0, 0, 0, time.UTC)
		julyOrder.SubscriptionStartDate = time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
		julyOrder.Note = "Demo Sprint 10: pembelian Juli dari top-up April dan auto reconciliation closing"
		julyOrderID, err := fake.CreateSubscriptionOrder(ctx, tx, ownerIDs[2], julyOrder)
		if err != nil {
			return err
		}
		if err := seedPartialConfirmDemo(ctx, tx, julyOrderID); err != nil {
			return err
		}

		matchedTransfer := fake.BuildTransfer(10, "4500000.00", "MATCHED")
		matchedTransfer.TransferDate = aprilTopup.PaidAt
		matchedTransfer.Note = "Demo Sprint 15a: transfer sudah dikonfirmasi cocok dengan top-up April"
		matchedTransfer.MatchedWalletPaymentID = sql.NullInt64{Int64: aprilPaymentID, Valid: true}
		matchedTransfer.ExternalReference = "DEMO-TRF-MATCHED-OWNER-003"
		if _, err := fake.CreateTransfer(ctx, tx, ownerIDs[2], matchedTransfer); err != nil {
			return err
		}

		hangingTopup := fake.BuildWalletTopup(11, "500000.00")
		hangingTopup.ExternalReference = "DEMO-TOPUP-HANGING-OWNER-004"
		hangingTopup.IdempotencyKey = "demo:topup:hanging-owner-004"
		hangingTopup.PaidAt = time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
		hangingTopup.Note = "Demo Sprint 10: top-up April untuk hanging subscription order"
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[3], hangingTopup); err != nil {
			return err
		}
		hangingOrder := fake.BuildSubscriptionOrder(11, "BASIC_01_MONTHS", "", sql.NullInt64{})
		hangingOrder.ExternalReference = "DEMO-SUB-HANGING-OWNER-004"
		hangingOrder.IdempotencyKey = "demo:subscription-order:hanging-owner-004"
		hangingOrder.PurchasedAt = time.Date(2026, 7, 12, 13, 0, 0, 0, time.UTC)
		hangingOrder.SubscriptionStartDate = time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
		hangingOrder.Note = "Demo Sprint 10: pembelian tanpa closing untuk reconciliation issue queue"
		if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerIDs[3], hangingOrder); err != nil {
			return err
		}
	}
	if err := seedPartnersDemo(ctx, tx, fake); err != nil {
		return err
	}
	return nil
}

// seedPartialConfirmDemo converts an already-auto-reconciled demo order (CreateSubscriptionOrder
// always inserts a plain CONFIRMED reconciliation, matching+consistent final_amount) into a
// PARTIAL_CONFIRM demo scenario — Sprint 15a §4's admin correction path, where the admin dashboard's
// real final amount differs slightly from what Sales entered. Mirrors production's
// confirmOrderAndClosingWithAmount: admin_final_amount overrides sales_closings.final_amount, order
// status stays RECONCILED (PARTIAL_CONFIRM only changes the reconciliation, not the order status).
func seedPartialConfirmDemo(ctx context.Context, tx *sql.Tx, orderID int64) error {
	var reconciliationID, closingID int64
	var finalAmount string
	var tenureMonths int
	err := tx.QueryRowContext(ctx, `
		SELECT sr.id, sr.closing_id, CAST(sc.final_amount AS CHAR), sc.tenure_months
		FROM subscription_reconciliations sr
		JOIN sales_closings sc ON sc.id = sr.closing_id
		WHERE sr.order_id = ? AND sr.deleted_at IS NULL
		LIMIT 1`, orderID).Scan(&reconciliationID, &closingID, &finalAmount, &tenureMonths)
	if err != nil {
		return fmt.Errorf("seed partial confirm demo lookup order=%d: %w", orderID, err)
	}

	finalAmountCents, err := parseDecimalToCents(finalAmount)
	if err != nil {
		return err
	}
	// Admin dashboard's real number is Rp 50.000 lower than what Sales originally entered.
	adminFinalAmount := formatCentsToDecimal(finalAmountCents - 5000)

	if _, err := tx.ExecContext(ctx, `
		UPDATE subscription_reconciliations
		SET status = 'PARTIAL_CONFIRM', admin_final_amount = ?, admin_tenure_months = ?,
			note = ?
		WHERE id = ?`,
		adminFinalAmount, tenureMonths,
		"Demo Sprint 15a: dikoreksi admin, nominal dashboard berbeda tipis dari input Sales.",
		reconciliationID,
	); err != nil {
		return fmt.Errorf("seed partial confirm demo update reconciliation order=%d: %w", orderID, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE sales_closings SET final_amount = ? WHERE id = ?`,
		adminFinalAmount, closingID,
	); err != nil {
		return fmt.Errorf("seed partial confirm demo update closing order=%d: %w", orderID, err)
	}
	return nil
}

func seedPartnersDemo(ctx context.Context, tx *sql.Tx, fake *factory.Factory) error {
	type partnerSeedInfo struct {
		PartnerID     int64
		PartnerTypeID int64
		PartnerCode   string
	}
	type referralSeedInfo struct {
		PartnerID     int64
		PartnerTypeID int64
		PartnerCode   string
		ReferralID    int64
		LeadID        int64
	}

	partnerTypeIDs := map[string]int64{}
	for _, code := range []string{"REFERRAL", "PARTNERSHIP", "STRATEGIC"} {
		typeID, err := lookupID(ctx, tx, "partner_types", "code", code)
		if err != nil {
			return err
		}
		partnerTypeIDs[code] = typeID
	}

	adminID, err := lookupID(ctx, tx, "users", "code", "ADM-001")
	if err != nil || adminID == 0 {
		adminID = 1
	}
	supervisorIDs, err := lookupUserIDsByRole(ctx, tx, "SUPERVISOR")
	if err != nil {
		return err
	}
	salesIDs, err := lookupUserIDsByRole(ctx, tx, "SALES")
	if err != nil {
		return err
	}
	if len(supervisorIDs) == 0 {
		supervisorIDs = []int64{adminID}
	}
	if len(salesIDs) == 0 {
		salesIDs = []int64{adminID}
	}

	leadCount, err := countTableRows(ctx, tx, "SELECT COUNT(*) FROM customer_leads WHERE deleted_at IS NULL")
	if err != nil {
		return err
	}
	targetPartners := 18
	if leadCount > 18 {
		targetPartners = minInt(160, maxInt(18, leadCount/40))
	}

	confirmedClosings, err := ensurePartnerDemoConfirmedClosings(ctx, tx, fake, 5)
	if err != nil {
		return err
	}

	leadIDs, err := lookupLeadIDs(ctx, tx, maxInt(targetPartners*2, len(confirmedClosings)+5))
	if err != nil {
		return err
	}
	if len(leadIDs) == 0 {
		return nil
	}

	baseAt := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	typeCycle := []string{"REFERRAL", "PARTNERSHIP", "STRATEGIC"}
	typeLabel := map[string]string{
		"REFERRAL":    "Referral",
		"PARTNERSHIP": "Partnership",
		"STRATEGIC":   "Strategic",
	}

	partners := make([]partnerSeedInfo, 0, targetPartners)
	referralsByLead := make(map[int64]referralSeedInfo)
	referralCursor := 0

	for i := 1; i <= targetPartners; i++ {
		typeCode := typeCycle[(i-1)%len(typeCycle)]
		partner := fake.BuildPartner(typeCode, i)
		partner.Name = fmt.Sprintf("Mitra %s Demo %03d", typeLabel[typeCode], i)
		partner.Address = fmt.Sprintf("Jl. Mitra Demo No. %d, Indonesia", i)
		partner.BankAccount = fmt.Sprintf("52701234%04d", 1000+i)
		if i%7 == 0 {
			partner.Status = "INACTIVE"
		} else {
			partner.Status = "ACTIVE"
		}

		result, err := tx.ExecContext(ctx, `
			INSERT INTO partners
				(partner_type_id, code, name, phone, email, address, bank_account_encrypted, bank_account_last4, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
			ON DUPLICATE KEY UPDATE
				partner_type_id = VALUES(partner_type_id),
				name = VALUES(name),
				phone = VALUES(phone),
				email = VALUES(email),
				address = VALUES(address),
				bank_account_encrypted = VALUES(bank_account_encrypted),
				bank_account_last4 = VALUES(bank_account_last4),
				status = VALUES(status),
				updated_at = NOW()`,
			partnerTypeIDs[typeCode],
			partner.Code,
			partner.Name,
			partner.Phone,
			partner.Email,
			partner.Address,
			[]byte("enc_"+partner.BankAccount),
			partner.BankAccount[len(partner.BankAccount)-4:],
			partner.Status,
		)
		if err != nil {
			return fmt.Errorf("seed partner %s: %w", partner.Code, err)
		}
		partnerID, _ := result.LastInsertId()
		if partnerID == 0 {
			partnerID, err = lookupID(ctx, tx, "partners", "code", partner.Code)
			if err != nil {
				return err
			}
		}

		if err := resetPartnerDemoChildren(ctx, tx, partnerID); err != nil {
			return err
		}

		partners = append(partners, partnerSeedInfo{
			PartnerID:     partnerID,
			PartnerTypeID: partnerTypeIDs[typeCode],
			PartnerCode:   partner.Code,
		})

		assignedAt := baseAt.AddDate(0, 0, i)
		if i%4 == 0 {
			historicalUserID := salesIDs[i%len(salesIDs)]
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO partner_assignments
					(partner_id, user_id, assigned_by_id, assigned_at, unassigned_at, active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, FALSE, NOW(), NOW())`,
				partnerID, historicalUserID, adminID, assignedAt.AddDate(0, 0, -10), assignedAt.AddDate(0, 0, -2),
			); err != nil {
				return fmt.Errorf("seed historical partner assignment %s: %w", partner.Code, err)
			}
		}
		switch {
		case i%6 == 0:
			// intentionally leave without active PIC to show unassigned state
		case i%5 == 0:
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO partner_assignments
					(partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
				VALUES (?, ?, ?, ?, TRUE, NOW(), NOW())`,
				partnerID, supervisorIDs[(i-1)%len(supervisorIDs)], adminID, assignedAt,
			); err != nil {
				return fmt.Errorf("seed supervisor partner assignment %s: %w", partner.Code, err)
			}
		default:
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO partner_assignments
					(partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
				VALUES (?, ?, ?, ?, TRUE, NOW(), NOW())`,
				partnerID, salesIDs[(i-1)%len(salesIDs)], supervisorIDs[(i-1)%len(supervisorIDs)], assignedAt,
			); err != nil {
				return fmt.Errorf("seed sales partner assignment %s: %w", partner.Code, err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_interactions (partner_id, interaction_type, interaction_at, note, created_at)
			VALUES (?, 'CALL', ?, ?, NOW())`,
			partnerID,
			assignedAt.Add(2*time.Hour),
			fmt.Sprintf("Call mitra demo %s untuk penawaran dan silaturahmi", partner.Code),
		); err != nil {
			return fmt.Errorf("seed partner interaction CALL %s: %w", partner.Code, err)
		}
		if i%2 == 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO partner_interactions (partner_id, interaction_type, interaction_at, note, created_at)
				VALUES (?, 'CHAT', ?, ?, NOW())`,
				partnerID,
				assignedAt.Add(5*time.Hour),
				fmt.Sprintf("Chat follow up mitra demo %s terkait prospek owner laundry", partner.Code),
			); err != nil {
				return fmt.Errorf("seed partner interaction CHAT %s: %w", partner.Code, err)
			}
		}

		referralPerPartner := 1
		if i%3 == 0 {
			referralPerPartner = 2
		}
		for r := 0; r < referralPerPartner && referralCursor < len(leadIDs); r++ {
			leadID := leadIDs[referralCursor]
			referralCursor++
			result, err := tx.ExecContext(ctx, `
				INSERT INTO partner_referrals (partner_id, lead_id, referral_date, notes, created_at)
				VALUES (?, ?, ?, ?, NOW())`,
				partnerID,
				leadID,
				assignedAt.AddDate(0, 0, r+1),
				fmt.Sprintf("Referral demo %s untuk lead %d", partner.Code, leadID),
			)
			if err != nil {
				return fmt.Errorf("seed partner referral %s lead=%d: %w", partner.Code, leadID, err)
			}
			referralID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("seed partner referral id %s: %w", partner.Code, err)
			}
			if _, exists := referralsByLead[leadID]; !exists {
				referralsByLead[leadID] = referralSeedInfo{
					PartnerID:     partnerID,
					PartnerTypeID: partnerTypeIDs[typeCode],
					PartnerCode:   partner.Code,
					ReferralID:    referralID,
					LeadID:        leadID,
				}
			}
		}
	}

	commissionScenarios := []struct {
		Status          string
		BuildPending    bool
		BuildPaid       bool
		BuildCancelledP bool
	}{
		{Status: "PENDING"},
		{Status: "APPROVED", BuildPending: true},
		{Status: "PAID", BuildPaid: true},
		{Status: "CANCELLED"},
		{Status: "APPROVED", BuildCancelledP: true},
	}

	for idx, scenario := range commissionScenarios {
		if idx >= len(confirmedClosings) {
			break
		}
		closingInfo := confirmedClosings[idx]
		referral, ok := referralsByLead[closingInfo.LeadID]
		if !ok {
			continue
		}
		ruleID, commissionMode, commissionValue, err := lookupPartnerCommissionSnapshot(ctx, tx, referral.PartnerTypeID, closingInfo.PlanID)
		if err != nil {
			return err
		}
		commissionAmount, err := calculatePartnerCommissionAmount(closingInfo.FinalAmount, commissionMode, commissionValue)
		if err != nil {
			return err
		}
		approvedAt := sql.NullTime{}
		paidAt := sql.NullTime{}
		approvedBy := sql.NullInt64{}
		paidBy := sql.NullInt64{}
		if scenario.Status == "APPROVED" || scenario.Status == "PAID" {
			approvedAt = sql.NullTime{Time: closingInfo.ConfirmedAt.Add(24 * time.Hour), Valid: true}
			approvedBy = sql.NullInt64{Int64: adminID, Valid: true}
		}
		if scenario.Status == "PAID" {
			paidAt = sql.NullTime{Time: closingInfo.ConfirmedAt.Add(72 * time.Hour), Valid: true}
			paidBy = sql.NullInt64{Int64: adminID, Valid: true}
		}
		code := fmt.Sprintf("DEMO-COM-%03d", idx+1)
		result, err := tx.ExecContext(ctx, `
			INSERT INTO partner_commissions
				(code, partner_id, referral_id, closing_id, commission_mode, commission_value, commission_rule_id,
				 base_amount, commission_amount, currency, status, note, approved_by_user_id, approved_at, paid_by_user_id, paid_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			code,
			referral.PartnerID,
			referral.ReferralID,
			closingInfo.ClosingID,
			commissionMode,
			commissionValue,
			ruleID,
			closingInfo.FinalAmount,
			commissionAmount,
			closingInfo.Currency,
			scenario.Status,
			fmt.Sprintf("Demo komisi %s untuk mitra %s", scenario.Status, referral.PartnerCode),
			approvedBy,
			approvedAt,
			paidBy,
			paidAt,
		)
		if err != nil {
			return fmt.Errorf("seed partner commission %s: %w", code, err)
		}
		commissionID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		switch {
		case scenario.BuildPending:
			if err := seedPartnerPayoutDemo(ctx, tx, partnerPayoutSeedInput{
				Code:             fmt.Sprintf("DEMO-PAYOUT-PENDING-%03d", idx+1),
				PartnerID:        referral.PartnerID,
				CommissionID:     commissionID,
				Amount:           commissionAmount,
				Status:           "PENDING",
				PreparedByUserID: adminID,
			}); err != nil {
				return err
			}
		case scenario.BuildPaid:
			if err := seedPartnerPayoutDemo(ctx, tx, partnerPayoutSeedInput{
				Code:             fmt.Sprintf("DEMO-PAYOUT-PAID-%03d", idx+1),
				PartnerID:        referral.PartnerID,
				CommissionID:     commissionID,
				Amount:           commissionAmount,
				Status:           "PAID",
				PreparedByUserID: adminID,
				PaidByUserID:     adminID,
				PaidAt:           sql.NullTime{Time: closingInfo.ConfirmedAt.Add(96 * time.Hour), Valid: true},
			}); err != nil {
				return err
			}
		case scenario.BuildCancelledP:
			if err := seedPartnerPayoutDemo(ctx, tx, partnerPayoutSeedInput{
				Code:             fmt.Sprintf("DEMO-PAYOUT-CANCELLED-%03d", idx+1),
				PartnerID:        referral.PartnerID,
				CommissionID:     commissionID,
				Amount:           commissionAmount,
				Status:           "CANCELLED",
				PreparedByUserID: adminID,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

type partnerClosingSeedInfo struct {
	LeadID      int64
	ClosingID   int64
	PlanID      int64
	FinalAmount string
	Currency    string
	ConfirmedAt time.Time
}

type partnerPayoutSeedInput struct {
	Code             string
	PartnerID        int64
	CommissionID     int64
	Amount           string
	Status           string
	PreparedByUserID int64
	PaidByUserID     int64
	PaidAt           sql.NullTime
}

func resetPartnerDemoChildren(ctx context.Context, tx *sql.Tx, partnerID int64) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE ppi FROM partner_payout_items ppi
		INNER JOIN partner_payouts pp ON pp.id = ppi.payout_id
		WHERE pp.partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner payout items partner=%d: %w", partnerID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM partner_payouts WHERE partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner payouts partner=%d: %w", partnerID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM partner_commissions WHERE partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner commissions partner=%d: %w", partnerID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM partner_referrals WHERE partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner referrals partner=%d: %w", partnerID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM partner_interactions WHERE partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner interactions partner=%d: %w", partnerID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM partner_assignments WHERE partner_id = ?`, partnerID); err != nil {
		return fmt.Errorf("reset partner assignments partner=%d: %w", partnerID, err)
	}
	return nil
}

func seedPartnerPayoutDemo(ctx context.Context, tx *sql.Tx, input partnerPayoutSeedInput) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO partner_payouts
			(code, partner_id, total_amount, currency, status, note, prepared_by_user_id, paid_by_user_id, paid_at, created_at, updated_at)
		VALUES (?, ?, ?, 'IDR', ?, ?, ?, ?, ?, NOW(), NOW())`,
		input.Code,
		input.PartnerID,
		input.Amount,
		input.Status,
		fmt.Sprintf("Demo payout %s", input.Status),
		input.PreparedByUserID,
		nullableInt64(input.PaidByUserID),
		input.PaidAt,
	)
	if err != nil {
		return fmt.Errorf("seed partner payout %s: %w", input.Code, err)
	}
	payoutID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	releasedAt := sql.NullTime{}
	if input.Status == "CANCELLED" {
		releasedAt = sql.NullTime{Time: time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO partner_payout_items (payout_id, commission_id, amount, released_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())`,
		payoutID,
		input.CommissionID,
		input.Amount,
		releasedAt,
	); err != nil {
		return fmt.Errorf("seed partner payout item %s: %w", input.Code, err)
	}
	return nil
}

func ensurePartnerDemoConfirmedClosings(ctx context.Context, tx *sql.Tx, fake *factory.Factory, minimum int) ([]partnerClosingSeedInfo, error) {
	items, err := lookupConfirmedClosingsForPartnerSeed(ctx, tx, minimum)
	if err != nil {
		return nil, err
	}
	if len(items) >= minimum {
		return items, nil
	}

	salesEmails, err := lookupUserEmailsByRole(ctx, tx, "SALES")
	if err != nil {
		return nil, err
	}
	if len(salesEmails) == 0 {
		return items, nil
	}

	startIndex := 90000 + len(items) + 1
	for len(items) < minimum {
		index := startIndex + len(items)
		owner := fake.BuildOwner(index)
		owner.Name = fmt.Sprintf("Owner Mitra Demo %03d", index)
		owner.BrandName = fmt.Sprintf("Laundry Mitra Demo %03d", index)
		owner.Email = fmt.Sprintf("owner.mitra.demo.%03d@example.test", index)
		ownerID, err := fake.CreateOwner(ctx, tx, owner)
		if err != nil {
			return nil, err
		}
		outlet := fake.BuildOutlet(owner.Code, 1, owner)
		outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
		if err != nil {
			return nil, err
		}
		lead := fake.BuildLead(owner.Code, 1, salesEmails[len(items)%len(salesEmails)])
		lead.Stage = "POTENTIAL"
		lead.SourceType = "PARTNER_DEMO"
		lead.SourceReference = fmt.Sprintf("partner-seed-%06d", index)
		leadID, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead)
		if err != nil {
			return nil, err
		}
		interaction := fake.BuildInteraction(index, 2)
		interaction.Note = fmt.Sprintf("Interaksi mitra demo %06d", index)
		if _, err := fake.CreateInteraction(ctx, tx, leadID, interaction); err != nil {
			return nil, err
		}
		closing := fake.BuildClosing(index, "BUSINESS_12_MONTHS", "FREE_1_MONTH_BUSINESS_12", "CONFIRMED")
		closing.Note = fmt.Sprintf("Closing mitra demo %06d", index)
		closingID, err := fake.CreateClosing(ctx, tx, leadID, closing)
		if err != nil {
			return nil, err
		}
		order := fake.BuildSubscriptionOrder(index, closing.PlanCode, closing.PromotionCode, sql.NullInt64{Int64: closingID, Valid: true})
		order.ExternalReference = fmt.Sprintf("DEMO-MITRA-SUB-%06d", index)
		order.IdempotencyKey = fmt.Sprintf("demo:mitra:subscription:%06d", index)
		topupAmount, err := fake.ResolveSubscriptionOrderTopupAmount(ctx, tx, ownerID, order, "2500000.00")
		if err != nil {
			return nil, err
		}
		topup := fake.BuildWalletTopup(index, topupAmount)
		topup.ExternalReference = fmt.Sprintf("DEMO-MITRA-TOPUP-%06d", index)
		topup.IdempotencyKey = fmt.Sprintf("demo:mitra:topup:%06d", index)
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerID, topup); err != nil {
			return nil, err
		}
		if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerID, order); err != nil {
			return nil, err
		}
		items, err = lookupConfirmedClosingsForPartnerSeed(ctx, tx, minimum)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func lookupConfirmedClosingsForPartnerSeed(ctx context.Context, tx *sql.Tx, limit int) ([]partnerClosingSeedInfo, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT sc.lead_id, sc.id, sc.plan_id, CAST(sc.final_amount AS CHAR), sc.currency, COALESCE(sc.confirmed_at, sc.closed_at)
		FROM sales_closings sc
		WHERE sc.deleted_at IS NULL
		  AND sc.status = 'CONFIRMED'
		ORDER BY COALESCE(sc.confirmed_at, sc.closed_at) ASC, sc.id ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("lookup confirmed closings for partner seed: %w", err)
	}
	defer rows.Close()

	var items []partnerClosingSeedInfo
	for rows.Next() {
		var item partnerClosingSeedInfo
		if err := rows.Scan(&item.LeadID, &item.ClosingID, &item.PlanID, &item.FinalAmount, &item.Currency, &item.ConfirmedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func lookupLeadIDs(ctx context.Context, tx *sql.Tx, limit int) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM customer_leads
		WHERE deleted_at IS NULL
		ORDER BY id
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("lookup lead ids: %w", err)
	}
	defer rows.Close()

	items := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

func lookupUserIDsByRole(ctx context.Context, tx *sql.Tx, roleCode string) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT u.id
		FROM users u
		INNER JOIN roles r ON r.id = u.role_id
		WHERE r.code = ? AND u.deleted_at IS NULL
		ORDER BY u.id`, roleCode)
	if err != nil {
		return nil, fmt.Errorf("lookup user ids role=%s: %w", roleCode, err)
	}
	defer rows.Close()

	var items []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

func lookupUserEmailsByRole(ctx context.Context, tx *sql.Tx, roleCode string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT u.email
		FROM users u
		INNER JOIN roles r ON r.id = u.role_id
		WHERE r.code = ? AND u.deleted_at IS NULL
		ORDER BY u.id`, roleCode)
	if err != nil {
		return nil, fmt.Errorf("lookup user emails role=%s: %w", roleCode, err)
	}
	defer rows.Close()

	var items []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		items = append(items, email)
	}
	return items, rows.Err()
}

func countTableRows(ctx context.Context, tx *sql.Tx, query string, args ...any) (int, error) {
	var total int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func lookupPartnerCommissionSnapshot(ctx context.Context, tx *sql.Tx, partnerTypeID, planID int64) (sql.NullInt64, string, string, error) {
	var ruleID int64
	var mode string
	var value sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, mode, CAST(value AS CHAR)
		FROM commission_rules
		WHERE partner_type_id = ?
		  AND active = TRUE
		  AND (plan_id = ? OR plan_id IS NULL)
		ORDER BY CASE WHEN plan_id = ? THEN 0 ELSE 1 END, effective_from DESC, id DESC
		LIMIT 1`, partnerTypeID, planID, planID).Scan(&ruleID, &mode, &value)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx, `
			SELECT commission_mode, CAST(commission_value AS CHAR)
			FROM partner_types
			WHERE id = ?`, partnerTypeID).Scan(&mode, &value)
		if err != nil {
			return sql.NullInt64{}, "", "", fmt.Errorf("lookup fallback partner type commission id=%d: %w", partnerTypeID, err)
		}
		return sql.NullInt64{}, mode, value.String, nil
	case err != nil:
		return sql.NullInt64{}, "", "", fmt.Errorf("lookup commission rule partner_type=%d plan=%d: %w", partnerTypeID, planID, err)
	default:
		return sql.NullInt64{Int64: ruleID, Valid: true}, mode, value.String, nil
	}
}

func calculatePartnerCommissionAmount(baseAmount, mode, value string) (string, error) {
	baseCents, err := parseDecimalToCents(baseAmount)
	if err != nil {
		return "", err
	}
	valueCents, err := parseDecimalToCents(value)
	if err != nil {
		return "", err
	}
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "FIXED":
		return value, nil
	case "PERCENTAGE":
		amountCents := (baseCents * valueCents) / 10000
		return formatCentsToDecimal(amountCents), nil
	default:
		return "", fmt.Errorf("mode komisi seed tidak didukung: %s", mode)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func lookupFirstLeadID(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM customer_leads
		WHERE deleted_at IS NULL
		ORDER BY id
		LIMIT 1`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("lookup first customer lead: %w", err)
	}
	return id, nil
}

func lookupID(ctx context.Context, tx *sql.Tx, table, column, value string) (int64, error) {
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s = ?", table, column)
	var id int64
	if err := tx.QueryRowContext(ctx, query, value).Scan(&id); err != nil {
		return 0, fmt.Errorf("lookup %s.%s=%s: %w", table, column, value, err)
	}
	return id, nil
}

func checksumFor(options Options) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s|%s|%d|%.4f", options.Mode, options.Preset, options.Seed, options.From.Format("2006-01-02"), options.To.Format("2006-01-02"), options.Scale, options.Variation)))
	return hex.EncodeToString(sum[:])
}

func parseDecimalToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse decimal seed %q: %w", value, err)
	}
	var cents int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		cents, err = strconv.ParseInt(fraction[:2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse decimal seed %q: %w", value, err)
		}
	}
	return whole*100 + cents, nil
}

func formatCentsToDecimal(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func nullableString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableInt64(value int64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func nullableDate(value time.Time) sql.NullString {
	if value.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: value.Format("2006-01-02"), Valid: true}
}

const seedUsage = `CRM Piposmart seed

Usage:
  crm seed master
  crm seed demo --preset=minimal --seed=20260723 --from=2026-07-01 --to=2026-07-01
  crm seed demo --preset=large --seed=20260723 --from=2026-01-01 --to=2026-07-01 --scale=10 --variation=0.5`
