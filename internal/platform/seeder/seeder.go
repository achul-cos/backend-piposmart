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
	Mode   string
	Preset string
	Seed   int64
	AsOf   time.Time
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
		options.Preset = "minimal"
		options.Seed = 1
		options.AsOf = time.Now().UTC().Truncate(24 * time.Hour)
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
			case "--as-of":
				asOf, err := time.Parse("2006-01-02", value)
				if err != nil {
					return Options{}, fmt.Errorf("--as-of harus format YYYY-MM-DD: %w", err)
				}
				options.AsOf = asOf
			default:
				return Options{}, fmt.Errorf("argumen seed demo tidak dikenal: %s", key)
			}
		}
		if options.Preset != "minimal" {
			return Options{}, fmt.Errorf("preset demo %q belum tersedia; gunakan minimal", options.Preset)
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
	)
	fmt.Fprintf(output, "seed %s selesai", options.Mode)
	if options.Mode == ModeDemo {
		fmt.Fprintf(output, " (preset=%s, seed=%d, as_of=%s)", options.Preset, options.Seed, options.AsOf.Format("2006-01-02"))
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
		if err := seedDemoMinimal(ctx, tx, options); err != nil {
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
		nullableDate(options.AsOf),
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

func seedPartnerTypes(ctx context.Context, tx *sql.Tx) error {
	rows := []struct {
		code, name, mode, description string
		value                         string
	}{
		{"REFERRAL_REGULAR", "Mitra Regular", "FIXED", "Mitra dengan komisi nominal tetap.", "0.00"},
		{"REFERRAL_STRATEGIC", "Mitra Strategic", "PERCENTAGE", "Mitra strategis dengan komisi persentase.", "0.00"},
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

	for ownerIndex := 1; ownerIndex <= 4; ownerIndex++ {
		owner := fake.BuildOwner(ownerIndex)
		ownerID, err := fake.CreateOwner(ctx, tx, owner)
		if err != nil {
			return err
		}

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
	return nil
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
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%s", options.Mode, options.Preset, options.Seed, options.AsOf.Format("2006-01-02"))))
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
  crm seed demo --preset=minimal --seed=20260723 --as-of=2026-07-01`
