package seeder

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	"backend_crm_piposmart/internal/platform/factory"
)

const (
	defaultLargeSeedScale   = 10
	interactionsPerLeadMin  = 5
	interactionsPerLeadSpan = 11 // interactions = min + rand(0..span) => 5-15
	closingRatePercent      = 12
)

var largeSeedScaleOwnerCounts = map[int]int{
	1:  50,
	2:  100,
	3:  200,
	4:  500,
	5:  1000,
	6:  2000,
	7:  3000,
	8:  5000,
	9:  1000,
	10: 18000,
}

type largeClosingScenario struct {
	PlanCode      string
	PromotionCode string
	Status        string
	TopupAmount   string
}

var largeClosingScenarios = []largeClosingScenario{
	{PlanCode: "BASIC_01_MONTHS", PromotionCode: "", Status: "PENDING_RECONCILIATION", TopupAmount: "500000.00"},
	{PlanCode: "BASIC_09_MONTHS", PromotionCode: "", Status: "CONFIRMED", TopupAmount: "1500000.00"},
	{PlanCode: "BUSINESS_12_MONTHS", PromotionCode: "FREE_1_MONTH_BUSINESS_12", Status: "CONFIRMED", TopupAmount: "2500000.00"},
	{PlanCode: "PRO_12_MONTHS", PromotionCode: "", Status: "CONFIRMED", TopupAmount: "3500000.00"},
	{PlanCode: "PRO_12_MONTHS", PromotionCode: "PRO_12_ANDROID_POS_BUNDLE", Status: "PENDING_RECONCILIATION", TopupAmount: "5000000.00"},
	{PlanCode: "PRO_24_MONTHS", PromotionCode: "FREE_2_MONTHS_PRO_24", Status: "CONFIRMED", TopupAmount: "7000000.00"},
}

func seedDemoLarge(ctx context.Context, tx *sql.Tx, options Options) error {
	fake := factory.New(options.Seed, options.AsOf)
	rng := rand.New(rand.NewSource(options.Seed))

	ownerCount, err := largeSeedOwnerCountForScale(options.Scale)
	if err != nil {
		return err
	}
	supervisorCount, salesCount := largeSeedTeamCounts(ownerCount)

	for i := 1; i <= supervisorCount; i++ {
		user := fake.BuildUser("SUPERVISOR", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return err
		}
	}

	salesEmails := make([]string, 0, salesCount)
	for i := 1; i <= salesCount; i++ {
		user := fake.BuildUser("SALES", i)
		if _, err := fake.CreateUser(ctx, tx, user); err != nil {
			return err
		}
		salesEmails = append(salesEmails, user.Email)
	}

	timeline := generateGrowthTimeline(options.From, options.To, ownerCount, options.Variation, rng)
	progress := newProgressBar(len(timeline))

	for idx, createdAt := range timeline {
		ownerIndex := idx + 1
		owner := buildLargeOwner(ownerIndex, createdAt, rng)
		ownerID, err := fake.CreateOwner(ctx, tx, owner)
		if err != nil {
			return fmt.Errorf("create owner %d: %w", ownerIndex, err)
		}

		outletCount := largeSeedOutletCount(rng)
		var firstOutletID int64
		for outletIndex := 1; outletIndex <= outletCount; outletIndex++ {
			outlet := fake.BuildOutlet(owner.Code, outletIndex, owner)
			outletID, err := fake.CreateOutlet(ctx, tx, ownerID, outlet)
			if err != nil {
				return fmt.Errorf("create outlet owner=%d outlet=%d: %w", ownerIndex, outletIndex, err)
			}
			if outletIndex == 1 {
				firstOutletID = outletID
			}
		}

		salesEmail := salesEmails[rng.Intn(len(salesEmails))]
		leadCreatedAt := clampToAsOf(createdAt.AddDate(0, 0, rng.Intn(7)+1), options.AsOf)
		lead := fake.BuildLead(owner.Code, 1, salesEmail)
		lead.NextFollowUpAt = clampToAsOf(leadCreatedAt.AddDate(0, 0, 3+rng.Intn(6)), options.AsOf)

		leadID, err := fake.CreateLead(ctx, tx, ownerID, firstOutletID, lead)
		if err != nil {
			return fmt.Errorf("create lead owner=%d: %w", ownerIndex, err)
		}

		willClose := rng.Intn(100) < closingRatePercent
		interactionCount := interactionsPerLeadMin + rng.Intn(interactionsPerLeadSpan)
		trainingCreated := false
		for interactionIndex := 1; interactionIndex <= interactionCount; interactionIndex++ {
			remarkScore := largeSeedRemarkScore(rng, willClose, interactionIndex, interactionCount)
			interaction := fake.BuildInteraction(interactionIndex, remarkScore)
			interactionDaysOffset := (interactionIndex - 1) * (rng.Intn(9) + 3)
			interaction.InteractionAt = clampToAsOf(leadCreatedAt.AddDate(0, 0, interactionDaysOffset), options.AsOf)
			interaction.FollowUpAt = clampToAsOf(interaction.InteractionAt.AddDate(0, 0, 3+rng.Intn(8)), options.AsOf)
			if _, err := fake.CreateInteraction(ctx, tx, leadID, interaction); err != nil {
				return fmt.Errorf("create interaction owner=%d lead=%d idx=%d: %w", ownerIndex, leadID, interactionIndex, err)
			}

			if !trainingCreated && remarkScore == 2 && rng.Intn(100) < 35 {
				training := fake.BuildTrainingReport(interactionIndex, rng.Intn(100) < 55)
				training.ScheduledAt = clampToAsOf(interaction.InteractionAt.AddDate(0, 0, 2+rng.Intn(7)).Add(10*time.Hour), options.AsOf)
				if training.CompletedAt.Valid {
					completedAt := clampToAsOf(training.ScheduledAt.Add(90*time.Minute), options.AsOf)
					training.CompletedAt = sql.NullTime{Time: completedAt, Valid: true}
				}
				if _, err := fake.CreateTrainingReport(ctx, tx, leadID, training); err != nil {
					return fmt.Errorf("create training owner=%d lead=%d: %w", ownerIndex, leadID, err)
				}
				trainingCreated = true
			}
		}

		if !willClose && rng.Intn(100) < 28 {
			topup := fake.BuildWalletTopup(ownerIndex, largeSeedStandaloneTopupAmount(rng))
			topup.PaidAt = clampToAsOf(leadCreatedAt.AddDate(0, 0, 7+rng.Intn(45)).Add(9*time.Hour), options.AsOf)
			topup.ExternalReference = fmt.Sprintf("LARGE-TOPUP-OWNER-%06d", ownerIndex)
			topup.IdempotencyKey = fmt.Sprintf("large:topup:owner:%06d", ownerIndex)
			topup.Note = fmt.Sprintf("Large seed wallet top-up owner=%06d", ownerIndex)
			if _, err := fake.CreateWalletTopup(ctx, tx, ownerID, topup); err != nil {
				return fmt.Errorf("create standalone topup owner=%d: %w", ownerIndex, err)
			}
			if rng.Intn(100) < 20 {
				debit := fake.BuildWalletDebit(ownerIndex, largeSeedStandaloneDebitAmount(rng))
				debit.OccurredAt = clampToAsOf(topup.PaidAt.AddDate(0, 0, 2+rng.Intn(12)), options.AsOf)
				debit.ExternalReference = fmt.Sprintf("LARGE-DEBIT-OWNER-%06d", ownerIndex)
				debit.IdempotencyKey = fmt.Sprintf("large:debit:owner:%06d", ownerIndex)
				debit.Note = fmt.Sprintf("Large seed wallet debit owner=%06d", ownerIndex)
				if _, err := fake.CreateWalletDebit(ctx, tx, ownerID, debit); err != nil {
					return fmt.Errorf("create standalone debit owner=%d: %w", ownerIndex, err)
				}
			}
		}

		if willClose {
			scenario := largeClosingScenarios[rng.Intn(len(largeClosingScenarios))]
			closingTime := clampToAsOf(leadCreatedAt.AddDate(0, 0, 30+rng.Intn(90)), options.AsOf)
			closing := factory.Closing{
				PlanCode:           scenario.PlanCode,
				PromotionCode:      scenario.PromotionCode,
				DiscountAmount:     "0.00",
				UniqueTransferCode: ownerIndex*100 + rng.Intn(99),
				Status:             scenario.Status,
				Note:               fmt.Sprintf("Large seed closing owner=%d", ownerIndex),
				ClosedAt:           closingTime,
			}

			closingID, err := fake.CreateClosing(ctx, tx, leadID, closing)
			if err != nil {
				return fmt.Errorf("create closing owner=%d lead=%d: %w", ownerIndex, leadID, err)
			}

			if rng.Intn(100) < 72 {
				order := fake.BuildSubscriptionOrder(ownerIndex, scenario.PlanCode, scenario.PromotionCode, sql.NullInt64{Int64: closingID, Valid: true})
				order.PurchasedAt = clampToAsOf(closingTime.Add(2*time.Hour), options.AsOf)
				order.SubscriptionStartDate = dateOnlyUTC(order.PurchasedAt)
				order.ExternalReference = fmt.Sprintf("LARGE-SUB-ORDER-OWNER-%06d", ownerIndex)
				order.IdempotencyKey = fmt.Sprintf("large:subscription:owner:%06d", ownerIndex)
				order.Note = fmt.Sprintf("Large seed subscription order dari closing owner=%06d", ownerIndex)

				topupAmount, err := fake.ResolveSubscriptionOrderTopupAmount(ctx, tx, ownerID, order, scenario.TopupAmount)
				if err != nil {
					return fmt.Errorf("resolve linked topup owner=%d: %w", ownerIndex, err)
				}
				topup := fake.BuildWalletTopup(ownerIndex, topupAmount)
				topup.PaidAt = clampToAsOf(closingTime.AddDate(0, 0, -(1+rng.Intn(30))).Add(9*time.Hour), options.AsOf)
				topup.ExternalReference = fmt.Sprintf("LARGE-CLOSING-TOPUP-OWNER-%06d", ownerIndex)
				topup.IdempotencyKey = fmt.Sprintf("large:closing:topup:owner:%06d", ownerIndex)
				topup.Note = fmt.Sprintf("Large seed wallet top-up sebelum subscription order owner=%06d", ownerIndex)
				if _, err := fake.CreateWalletTopup(ctx, tx, ownerID, topup); err != nil {
					return fmt.Errorf("create linked topup owner=%d: %w", ownerIndex, err)
				}

				if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerID, order); err != nil {
					return fmt.Errorf("create linked subscription owner=%d: %w", ownerIndex, err)
				}
			}
		} else if rng.Intn(100) < 6 {
			topupPaidAt := clampToAsOf(leadCreatedAt.AddDate(0, 0, 14+rng.Intn(30)).Add(9*time.Hour), options.AsOf)
			order := fake.BuildSubscriptionOrder(ownerIndex, "BASIC_01_MONTHS", "", sql.NullInt64{})
			order.PurchasedAt = clampToAsOf(topupPaidAt.AddDate(0, 0, 2+rng.Intn(15)), options.AsOf)
			order.SubscriptionStartDate = dateOnlyUTC(order.PurchasedAt)
			order.ExternalReference = fmt.Sprintf("LARGE-HANGING-ORDER-OWNER-%06d", ownerIndex)
			order.IdempotencyKey = fmt.Sprintf("large:hanging:order:owner:%06d", ownerIndex)
			order.Note = fmt.Sprintf("Large seed hanging subscription order owner=%06d", ownerIndex)

			topupAmount, err := fake.ResolveSubscriptionOrderTopupAmount(ctx, tx, ownerID, order, "1500000.00")
			if err != nil {
				return fmt.Errorf("resolve hanging topup owner=%d: %w", ownerIndex, err)
			}
			topup := fake.BuildWalletTopup(ownerIndex, topupAmount)
			topup.PaidAt = topupPaidAt
			topup.ExternalReference = fmt.Sprintf("LARGE-HANGING-TOPUP-OWNER-%06d", ownerIndex)
			topup.IdempotencyKey = fmt.Sprintf("large:hanging:topup:owner:%06d", ownerIndex)
			topup.Note = fmt.Sprintf("Large seed top-up untuk hanging order owner=%06d", ownerIndex)
			if _, err := fake.CreateWalletTopup(ctx, tx, ownerID, topup); err != nil {
				return fmt.Errorf("create hanging topup owner=%d: %w", ownerIndex, err)
			}

			if _, err := fake.CreateSubscriptionOrder(ctx, tx, ownerID, order); err != nil {
				return fmt.Errorf("create hanging order owner=%d: %w", ownerIndex, err)
			}
		}

		progress.update(ownerIndex)
	}
	progress.finish()

	if err := seedPartnersDemo(ctx, tx, fake); err != nil {
		return err
	}

	return nil
}

func largeSeedOwnerCountForScale(scale int) (int, error) {
	count, ok := largeSeedScaleOwnerCounts[scale]
	if !ok {
		return 0, fmt.Errorf("--scale harus salah satu dari 1,2,3,4,5,6,7,8,9,10")
	}
	return count, nil
}

func largeSeedTeamCounts(ownerCount int) (supervisorCount int, salesCount int) {
	supervisorCount = maxInt(1, int(math.Ceil(float64(ownerCount)/2000.0)))
	salesCount = maxInt(3, int(math.Ceil(float64(ownerCount)/400.0)))
	return supervisorCount, salesCount
}

func largeSeedOutletCount(rng *rand.Rand) int {
	roll := rng.Intn(100)
	switch {
	case roll < 5:
		return 3
	case roll < 25:
		return 2
	default:
		return 1
	}
}

func largeSeedRemarkScore(rng *rand.Rand, willClose bool, interactionIndex, interactionCount int) int {
	if willClose {
		if interactionIndex >= interactionCount-1 {
			if rng.Intn(100) < 70 {
				return 2
			}
			return 1
		}
		if rng.Intn(100) < 65 {
			return 1
		}
		return 2
	}

	roll := rng.Intn(100)
	switch {
	case roll < 18:
		return 0
	case roll < 68:
		return 1
	default:
		return 2
	}
}

func largeSeedStandaloneTopupAmount(rng *rand.Rand) string {
	amounts := []string{"500000.00", "750000.00", "1000000.00", "1500000.00", "2500000.00"}
	return amounts[rng.Intn(len(amounts))]
}

func largeSeedStandaloneDebitAmount(rng *rand.Rand) string {
	amounts := []string{"100000.00", "150000.00", "250000.00", "300000.00"}
	return amounts[rng.Intn(len(amounts))]
}

func dateOnlyUTC(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

// generateGrowthTimeline distributes timestamps across a date range. The `variation`
// parameter controls how even vs spiky the daily distribution is:
//   - 1.0 => uniform per day
//   - 0.5 => balanced between uniform and random spikes
//   - 0.0 => highly random / extreme spread
func generateGrowthTimeline(startDate, endDate time.Time, targetCount int, variation float64, rng *rand.Rand) []time.Time {
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	if totalDays < 1 || targetCount < 1 {
		return nil
	}

	weights := make([]float64, totalDays)
	var totalWeight float64
	randomStrength := 1 - variation
	for day := 0; day < totalDays; day++ {
		if variation >= 0.9999 {
			weights[day] = 1
			totalWeight += 1
			continue
		}

		baseUniform := variation
		randomComponent := 0.15 + rng.Float64()*0.85
		if rng.Float64() < randomStrength {
			randomComponent *= 1 + rng.Float64()*(4+10*randomStrength)
		}
		weight := baseUniform + (randomStrength * randomComponent)
		if weight <= 0 {
			weight = 0.01
		}
		weights[day] = weight
		totalWeight += weight
	}

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
