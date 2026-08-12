package seeder

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type seedProgressPrinter struct {
	stageCount   int
	stageIndex   int
	stageName    string
	unitLabel    string
	total        int
	current      int
	startedAt    time.Time
	stageStarted time.Time
	lastDraw     time.Time
	width        int
	metricOrder  []string
	metrics      map[string]int
}

func newSeedProgressPrinter(stageCount int, metricOrder []string) *seedProgressPrinter {
	return &seedProgressPrinter{
		stageCount:  stageCount,
		startedAt:   time.Now(),
		width:       30,
		metricOrder: append([]string(nil), metricOrder...),
		metrics:     make(map[string]int),
	}
}

func (p *seedProgressPrinter) StartStage(name, unitLabel string, total int) {
	if p.stageIndex > 0 {
		fmt.Fprintln(os.Stderr)
	}
	p.stageIndex++
	p.stageName = name
	p.unitLabel = unitLabel
	p.total = total
	p.current = 0
	p.stageStarted = time.Now()
	p.lastDraw = time.Time{}
	for key := range p.metrics {
		delete(p.metrics, key)
	}
	fmt.Fprintf(os.Stderr, "[seed] tahap %d/%d: %s", p.stageIndex, p.stageCount, name)
	if total > 0 {
		fmt.Fprintf(os.Stderr, " (%d %s)", total, unitLabel)
	}
	fmt.Fprintln(os.Stderr)
}

func (p *seedProgressPrinter) SetMetric(name string, value int) {
	p.metrics[name] = value
}

func (p *seedProgressPrinter) AddMetric(name string, delta int) {
	p.metrics[name] += delta
}

func (p *seedProgressPrinter) Update(current int) {
	p.current = current
	p.render(false)
}

func (p *seedProgressPrinter) FinishStage() {
	if p.total > 0 {
		p.current = p.total
	}
	p.render(true)
}

func (p *seedProgressPrinter) Close() {
	fmt.Fprintln(os.Stderr)
}

func (p *seedProgressPrinter) render(force bool) {
	now := time.Now()
	if !force && now.Sub(p.lastDraw) < 150*time.Millisecond {
		return
	}
	p.lastDraw = now

	pct := 1.0
	switch {
	case p.total > 0:
		pct = float64(p.current) / float64(p.total)
	case p.current == 0:
		pct = 0
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	filled := int(pct * float64(p.width))
	bar := "[" + repeatRune('#', filled) + repeatRune('-', p.width-filled) + "]"

	elapsed := now.Sub(p.startedAt).Truncate(time.Second)
	stageElapsed := now.Sub(p.stageStarted).Truncate(time.Second)
	var eta time.Duration
	if p.total > 0 && p.current > 0 && p.current < p.total {
		eta = time.Duration(float64(now.Sub(p.stageStarted)) / float64(p.current) * float64(p.total-p.current))
	}

	progressText := fmt.Sprintf("%d/%d %s", p.current, p.total, p.unitLabel)
	if p.total <= 0 {
		progressText = fmt.Sprintf("%d %s", p.current, p.unitLabel)
	}

	fmt.Fprintf(
		os.Stderr,
		"\r%s %5.1f%%  tahap %d/%d %-24s %s  total %s  tahap %s  eta %s  %s   ",
		bar,
		pct*100,
		p.stageIndex,
		p.stageCount,
		truncateProgressLabel(p.stageName, 24),
		progressText,
		elapsed,
		stageElapsed,
		eta.Truncate(time.Second),
		p.metricsText(),
	)
}

func (p *seedProgressPrinter) metricsText() string {
	if len(p.metricOrder) == 0 {
		return ""
	}
	parts := make([]string, 0, len(p.metricOrder))
	for _, key := range p.metricOrder {
		value, ok := p.metrics[key]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%d", key, value))
	}
	return strings.Join(parts, "  ")
}

func truncateProgressLabel(label string, width int) string {
	if len(label) <= width {
		return label
	}
	if width <= 3 {
		return label[:width]
	}
	return label[:width-3] + "..."
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type subscriptionSeedStats struct {
	NewOwners  int
	NewOutlets int
	NewLeads   int
	Topups     int
	Transfers  int
	Orders     int
}

type subscriptionSeedProgress func(current, total int, stats subscriptionSeedStats)

type mitraSeedStats struct {
	Partners    int
	Assignments int
}

type mitraSeedProgress func(current, total int, stats mitraSeedStats)

type demoPartnerSeedStats struct {
	Partners         int
	Assignments      int
	Interactions     int
	Referrals        int
	Commissions      int
	Payouts          int
	BackfillOwners   int
	BackfillOutlets  int
	BackfillLeads    int
	BackfillClosings int
	BackfillTopups   int
	BackfillOrders   int
}

type demoPartnerSeedProgress func(current, total int, stats demoPartnerSeedStats)
