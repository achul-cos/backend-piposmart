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
		planCode                                  string
		referral, partnership, strategic          string
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
	}

	if len(ownerIDs) >= 4 && len(closingIDs) >= 3 {
		aprilTopup := fake.BuildWalletTopup(10, "4500000.00")
		aprilTopup.ExternalReference = "DEMO-TOPUP-APRIL-OWNER-003"
		aprilTopup.IdempotencyKey = "demo:topup:april-owner-003"
		aprilTopup.PaidAt = time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
		aprilTopup.Note = "Demo Sprint 10: top-up April, saldo dipakai beli subscription Juli"
		if _, err := fake.CreateWalletTopup(ctx, tx, ownerIDs[2], aprilTopup); err != nil {
			return err
		}
		julyOrder := fake.BuildSubscriptionOrder(10, "PRO_12_MONTHS", "", sql.NullInt64{Int64: closingIDs[2], Valid: true})
		julyOrder.ExternalReference = "DEMO-SUB-JULY-OWNER-003"
		julyOrder.IdempotencyKey = "demo:subscription-order:april-topup-july-purchase-owner-003"
		julyOrder.PurchasedAt = time.Date(2026, 7, 10, 13, 0, 0, 0, time.UTC)
		julyOrder.SubscriptionStartDate = time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
		julyOrder.Note = "Demo Sprint 10: pembelian Juli dari top-up April dan auto reconciliation closing"
		if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerIDs[2], julyOrder); err != nil {
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

func seedPartnersDemo(ctx context.Context, tx *sql.Tx, fake *factory.Factory) error {
	supTypeID, err := lookupID(ctx, tx, "partner_types", "code", "REFERRAL")
	if err != nil {
		return err
	}
	disTypeID, err := lookupID(ctx, tx, "partner_types", "code", "PARTNERSHIP")
	if err != nil {
		return err
	}
	refTypeID, err := lookupID(ctx, tx, "partner_types", "code", "STRATEGIC")
	if err != nil {
		return err
	}

	spvID, err := lookupID(ctx, tx, "users", "code", "SPV-001")
	if err != nil {
		spvID = 2
	}
	slsID, err := lookupID(ctx, tx, "users", "code", "SLS-001")
	if err != nil {
		slsID = 3
	}
	admID, _ := lookupID(ctx, tx, "users", "code", "ADM-001")
	if admID == 0 {
		admID = 1
	}

	partnersData := []struct {
		typeID                                                int64
		code, name, phone, email, address, bankLast4, encBank string
	}{
		{supTypeID, "SUP-001", "PT Hardware Maju POS", "08123456001", "contact@posmaju.demo.id", "Jl. Industri Hardware No. 12, Jakarta", "5678", "enc_bank_account_001"},
		{disTypeID, "DIS-001", "CV Digital Software Solution", "08123456002", "sales@digitalsft.demo.id", "Jl. Teknologi No. 45, Bandung", "1234", "enc_bank_account_002"},
		{refTypeID, "REF-001", "Komunitas UMKM Kopi Indonesia", "08123456003", "info@umkmkopi.demo.id", "Jl. Pemuda No. 8, Surabaya", "9876", "enc_bank_account_003"},
	}

	partnerIDs := make(map[string]int64)
	for _, p := range partnersData {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO partners (partner_type_id, code, name, phone, email, address, bank_account_encrypted, bank_account_last4, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', NOW(), NOW())
			ON DUPLICATE KEY UPDATE partner_type_id = VALUES(partner_type_id), name = VALUES(name), phone = VALUES(phone)`,
			p.typeID, p.code, p.name, p.phone, p.email, p.address, []byte(p.encBank), p.bankLast4,
		)
		if err != nil {
			return fmt.Errorf("seed partner %s: %w", p.code, err)
		}
		id, _ := res.LastInsertId()
		if id == 0 {
			id, _ = lookupID(ctx, tx, "partners", "code", p.code)
		}
		partnerIDs[p.code] = id
	}

	if pid, ok := partnerIDs["SUP-001"]; ok && pid > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_assignments (partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
			VALUES (?, ?, ?, NOW(), TRUE, NOW(), NOW())
			ON DUPLICATE KEY UPDATE active = TRUE`,
			pid, spvID, admID,
		); err != nil {
			return fmt.Errorf("seed assignment SUP-001: %w", err)
		}
	}
	if pid, ok := partnerIDs["DIS-001"]; ok && pid > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_assignments (partner_id, user_id, assigned_by_id, assigned_at, active, created_at, updated_at)
			VALUES (?, ?, ?, NOW(), TRUE, NOW(), NOW())
			ON DUPLICATE KEY UPDATE active = TRUE`,
			pid, slsID, spvID,
		); err != nil {
			return fmt.Errorf("seed assignment DIS-001: %w", err)
		}
	}

	if pid, ok := partnerIDs["SUP-001"]; ok && pid > 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO partner_interactions (partner_id, interaction_type, interaction_at, note, created_at)
			VALUES (?, 'CALL', NOW(), 'Diskusi penawaran paket bundle POS Kasir edisi Juli 2026', NOW())`,
			pid,
		); err != nil {
			return fmt.Errorf("seed interaction SUP-001: %w", err)
		}
	}

	leadID, err := lookupFirstLeadID(ctx, tx)
	if err == nil && leadID > 0 {
		if pid, ok := partnerIDs["REF-001"]; ok && pid > 0 {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO partner_referrals (partner_id, lead_id, referral_date, notes, created_at)
				VALUES (?, ?, NOW(), 'Komunitas UMKM merujuk lead Owner Kopi Kenangan', NOW())
				ON DUPLICATE KEY UPDATE notes = VALUES(notes)`,
				pid, leadID,
			); err != nil {
				return fmt.Errorf("seed referral REF-001: %w", err)
			}
		}
	}
	return nil
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
