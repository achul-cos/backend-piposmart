package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"

	"backend_crm_piposmart/internal/platform/factory"
)

// NOTE: `customer_leads.owner_id` memiliki UNIQUE KEY (uq_customer_leads_owner_id,
// lihat migrations/20260723000400_lead_ownership_assignment.sql). Artinya setiap
// owner (customer) hanya boleh memiliki TEPAT SATU lead aktif yang merepresentasikan
// status pipeline-nya. Volume data yang besar dicapai lewat banyaknya interactions
// per lead dan jumlah owner, BUKAN lewat banyak lead per owner.
const (
	largeOwnerCount         = 18000
	largeSalesCount         = 45
	largeSupervisorCount    = 9
	interactionsPerLeadMin  = 5
	interactionsPerLeadSpan = 11 // interactions = min + rand(0..span) => 5-15
	closingRatePercent      = 12
)

func seedDemoLarge(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)
	rng := rand.New(rand.NewSource(options.Seed))

	// 1. Create users: supervisors & sales team
	for i := 1; i <= largeSupervisorCount; i++ {
		user := fake.BuildUser("SUPERVISOR", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return err
		}
	}

	salesEmails := make([]string, 0, largeSalesCount)
	for i := 1; i <= largeSalesCount; i++ {
		user := fake.BuildUser("SALES", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return err
		}
		salesEmails = append(salesEmails, user.Email)
	}

	// 2. Generate temporal distribution: 2020-01-01 to asOf with growth curve
	timeline := generateGrowthTimeline(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), options.AsOf, largeOwnerCount, rng)

	closingPlans := []string{"BASIC_01_MONTHS", "BUSINESS_12_MONTHS", "PRO_12_MONTHS"}
	closingPromos := []string{"", "", "FREE_1_MONTH_BUSINESS_12", "PRO_12_ANDROID_POS_BUNDLE"}
	closingStatuses := []string{"PENDING_RECONCILIATION", "CONFIRMED", "CONFIRMED"}

	progress := newProgressBar(len(timeline))

	// 3. Create owners with temporal distribution, each with exactly one lead
	for idx, createdAt := range timeline {
		ownerIndex := idx + 1
		owner := buildLargeOwner(ownerIndex, createdAt, rng)
		ownerID, err := fake.CreateOwner(ctx, tx, owner)
		if err != nil {
			return fmt.Errorf("create owner %d: %w", ownerIndex, err)
		}

		outlet := fake.BuildOutlet(owner.Code, 1, owner)
		outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
		if err != nil {
			return fmt.Errorf("create outlet owner=%d: %w", ownerIndex, err)
		}

		salesEmail := salesEmails[rng.Intn(len(salesEmails))]
		leadCreatedAt := createdAt.AddDate(0, 0, rng.Intn(7)+1)
		lead := fake.BuildLead(owner.Code, 1, salesEmail)

		leadID, err := fake.CreateLead(ctx, tx, ownerID, outletID, lead)
		if err != nil {
			return fmt.Errorf("create lead owner=%d: %w", ownerIndex, err)
		}

		// Interactions: random 5-15 per lead, spread over time after lead creation
		interactionCount := interactionsPerLeadMin + rng.Intn(interactionsPerLeadSpan)
		for iIdx := 1; iIdx <= interactionCount; iIdx++ {
			remarkScore := rng.Intn(4)
			interaction := fake.BuildInteraction(iIdx, remarkScore)

			daysOffset := (iIdx - 1) * (rng.Intn(10) + 3)
			interaction.InteractionAt = clampToAsOf(leadCreatedAt.AddDate(0, 0, daysOffset), options.AsOf)
			interaction.FollowUpAt = interaction.InteractionAt.AddDate(0, 0, 3+rng.Intn(8))

			if _, err := fake.CreateInteraction(ctx, tx, leadID, interaction); err != nil {
				return fmt.Errorf("create interaction owner=%d lead=%d: %w", ownerIndex, leadID, err)
			}
		}

		// Random closing (~12% of leads close)
		if rng.Intn(100) < closingRatePercent {
			planCode := closingPlans[rng.Intn(len(closingPlans))]
			promoCode := closingPromos[rng.Intn(len(closingPromos))]
			status := closingStatuses[rng.Intn(len(closingStatuses))]

			closingTime := clampToAsOf(leadCreatedAt.AddDate(0, 0, 30+rng.Intn(90)), options.AsOf)
			closing := factory.Closing{
				PlanCode:           planCode,
				PromotionCode:      promoCode,
				DiscountAmount:     "0.00",
				UniqueTransferCode: ownerIndex*100 + rng.Intn(99),
				Status:             status,
				Note:               fmt.Sprintf("Large seed closing owner=%d", ownerIndex),
				ClosedAt:           closingTime,
			}

			if _, err := fake.CreateClosing(ctx, tx, leadID, closing); err != nil {
				return fmt.Errorf("create closing owner=%d lead=%d: %w", ownerIndex, leadID, err)
			}
		}

		progress.update(ownerIndex)
	}
	progress.finish()

	return nil
}

// progressBar renders an in-place, installer-style progress indicator on
// stderr (\r without newline), updated at most a few times per second so it
// doesn't itself become the bottleneck.
type progressBar struct {
	total     int
	startedAt time.Time
	lastDraw  time.Time
	width     int
}

func newProgressBar(total int) *progressBar {
	return &progressBar{total: total, startedAt: time.Now(), width: 30}
}

func (p *progressBar) update(current int) {
	now := time.Now()
	done := current >= p.total
	if !done && now.Sub(p.lastDraw) < 150*time.Millisecond {
		return
	}
	p.lastDraw = now

	pct := float64(current) / float64(p.total)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(p.width))
	bar := "[" + repeatRune('#', filled) + repeatRune('-', p.width-filled) + "]"

	elapsed := now.Sub(p.startedAt)
	var eta time.Duration
	if current > 0 {
		eta = time.Duration(float64(elapsed) / float64(current) * float64(p.total-current))
	}

	fmt.Fprintf(os.Stderr, "\r%s %5.1f%%  %d/%d owners  elapsed %s  eta %s   ",
		bar, pct*100, current, p.total,
		elapsed.Truncate(time.Second), eta.Truncate(time.Second))
}

func (p *progressBar) finish() {
	p.lastDraw = time.Time{}
	p.update(p.total)
	fmt.Fprintln(os.Stderr)
}

func repeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	runes := make([]rune, n)
	for i := range runes {
		runes[i] = r
	}
	return string(runes)
}

// clampToAsOf prevents generated events (interactions, closings) from landing
// in the future relative to the seeder's reference date.
func clampToAsOf(value, asOf time.Time) time.Time {
	if value.After(asOf) {
		return asOf
	}
	return value
}

// generateGrowthTimeline generates realistic timestamps following a startup growth curve:
// slow start -> growth -> acceleration -> plateau, with random daily spikes simulating
// trend/news-driven surges in customer acquisition. Owners are allocated per-day
// proportional to the growth-curve weight of that day (largest remainder method),
// guaranteeing exactly targetCount timestamps are produced even when targetCount
// greatly exceeds the number of days in the range (multiple owners share a day).
func generateGrowthTimeline(startDate, endDate time.Time, targetCount int, rng *rand.Rand) []time.Time {
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	if totalDays < 1 || targetCount < 1 {
		return nil
	}

	weights := make([]float64, totalDays)
	var totalWeight float64
	for day := 0; day < totalDays; day++ {
		phase := float64(day) / float64(totalDays)
		var multiplier float64

		switch {
		case phase < 0.2:
			// Startup phase: slow (20% of timeline)
			multiplier = 0.3 + rng.Float64()*0.3
		case phase < 0.6:
			// Growth phase: faster (40% of timeline)
			multiplier = 1.0 + rng.Float64()*1.0
		case phase < 0.9:
			// Acceleration: even faster (30% of timeline)
			multiplier = 1.5 + rng.Float64()*1.5
		default:
			// Maturity: steady (10% of timeline)
			multiplier = 0.8 + rng.Float64()*0.4
		}

		// Random daily spike (10% of days): simulate tren/berita-driven surge
		if rng.Float64() < 0.10 {
			multiplier *= 2.5
		}

		weights[day] = multiplier
		totalWeight += multiplier
	}

	// Largest remainder method: allocate integer owner counts per day
	// proportional to that day's weight, while guaranteeing the total
	// equals exactly targetCount.
	dayCounts := make([]int, totalDays)
	rawCounts := make([]float64, totalDays)
	assigned := 0
	for day := 0; day < totalDays; day++ {
		rawCounts[day] = weights[day] / totalWeight * float64(targetCount)
		dayCounts[day] = int(rawCounts[day])
		assigned += dayCounts[day]
	}

	type remainder struct {
		day  int
		frac float64
	}
	remainders := make([]remainder, totalDays)
	for day := 0; day < totalDays; day++ {
		remainders[day] = remainder{day: day, frac: rawCounts[day] - float64(dayCounts[day])}
	}
	sort.Slice(remainders, func(i, j int) bool { return remainders[i].frac > remainders[j].frac })

	remaining := targetCount - assigned
	for i := 0; i < remaining && i < len(remainders); i++ {
		dayCounts[remainders[i].day]++
	}

	timeline := make([]time.Time, 0, targetCount)
	for day := 0; day < totalDays; day++ {
		if dayCounts[day] <= 0 {
			continue
		}
		date := startDate.AddDate(0, 0, day)
		for k := 0; k < dayCounts[day]; k++ {
			hour := rng.Intn(24)
			minute := rng.Intn(60)
			timeline = append(timeline, time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC))
		}
	}

	return timeline
}

func buildLargeOwner(index int, createdAt time.Time, rng *rand.Rand) factory.Owner {
	provinces := []string{
		"DKI Jakarta", "Jawa Barat", "Jawa Timur", "Sumatera Utara", "Bali",
		"Sumatera Selatan", "Riau", "Lampung", "Bandung", "Yogyakarta",
	}
	cities := map[string][]string{
		"DKI Jakarta":      {"Jakarta Selatan", "Jakarta Barat", "Jakarta Timur", "Jakarta Pusat"},
		"Jawa Barat":       {"Bandung", "Bekasi", "Depok", "Bogor", "Cikarang"},
		"Jawa Timur":       {"Surabaya", "Malang", "Sidoarjo", "Gresik", "Lamongan"},
		"Sumatera Utara":   {"Medan", "Binjai", "Deli Serdang", "Pematang Siantar"},
		"Bali":             {"Denpasar", "Badung", "Gianyar", "Ubud", "Sanur"},
		"Sumatera Selatan": {"Palembang", "Sekayu", "Musi Rawas"},
		"Riau":             {"Pekanbaru", "Dumai", "Kampar"},
		"Lampung":          {"Bandar Lampung", "Metro", "Tulang Bawang"},
		"Bandung":          {"Bandung", "Cimahi", "Sumedang"},
		"Yogyakarta":       {"Yogyakarta", "Sleman", "Bantul"},
	}

	laundryNames := []string{
		"Laundry", "Laundri", "Cuci Kilat", "Cuci Express", "Servis Cepat",
		"Fresh Clean", "Bersih Jaya", "Prestasi", "Mandiri", "Maju Jaya",
		"Sejahtera", "Cerah", "Binar", "Gemilang", "Sukses",
	}

	province := provinces[rng.Intn(len(provinces))]
	cityOptions := cities[province]
	if len(cityOptions) == 0 {
		cityOptions = cities["DKI Jakarta"]
	}
	city := cityOptions[rng.Intn(len(cityOptions))]

	laundryName := laundryNames[rng.Intn(len(laundryNames))]

	return factory.Owner{
		Code:      fmt.Sprintf("OWN-%06d", index),
		Name:      fmt.Sprintf("%s %s %03d", laundryName, city, index),
		Phone:     fmt.Sprintf("62813%08d", 1000000+index),
		Email:     fmt.Sprintf("owner.%06d@gmail.com", index),
		BrandName: fmt.Sprintf("%s %s", laundryName, city),
		Province:  province,
		City:      city,
		Address:   fmt.Sprintf("Jl. Merdeka No. %d, %s, %s", (index%500)+1, city, province),
		CreatedAt: createdAt,
	}
}
