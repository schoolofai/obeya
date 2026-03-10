# Dashboard TUI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a real-time dashboard view to the Obeya TUI showing velocity, cycle time, WIP limits, and epic burndown.

**Architecture:** Extract shared metrics logic from `cmd/metrics.go` into `internal/metrics/`, add new velocity/burndown/WIP computations, then build a dashboard view state in the existing Bubble Tea TUI with chart rendering helpers.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Unicode block characters for charts.

**Spec:** `docs/superpowers/specs/2026-03-10-dashboard-tui-design.md`

---

## Chunk 1: Extract Metrics Package

### Task 1: Extract core metrics into internal/metrics/metrics.go

**Files:**
- Create: `internal/metrics/metrics.go`
- Create: `internal/metrics/metrics_test.go`
- Modify: `cmd/metrics.go`

- [ ] **Step 1: Write the failing test for Compute()**

Create `internal/metrics/metrics_test.go`:

```go
package metrics

import (
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

func TestCompute_EmptyItems(t *testing.T) {
	result := Compute(nil, time.Now())
	if result.TotalItems != 0 {
		t.Errorf("expected 0 total items, got %d", result.TotalItems)
	}
	if result.DoneItems != 0 {
		t.Errorf("expected 0 done items, got %d", result.DoneItems)
	}
}

func TestCompute_WithDoneItem(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []*domain.Item{
		{
			ID:        "abc123",
			Status:    "done",
			CreatedAt: now.Add(-48 * time.Hour),
			History: []domain.ChangeRecord{
				{Action: "created", Detail: "created task: Test", Timestamp: now.Add(-48 * time.Hour)},
				{Action: "moved", Detail: "status: backlog -> in-progress", Timestamp: now.Add(-24 * time.Hour)},
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-2 * time.Hour)},
			},
		},
	}
	result := Compute(items, now)
	if result.TotalItems != 1 {
		t.Errorf("expected 1 total item, got %d", result.TotalItems)
	}
	if result.DoneItems != 1 {
		t.Errorf("expected 1 done item, got %d", result.DoneItems)
	}
	if result.CycleTime == nil {
		t.Fatal("expected non-nil cycle time")
	}
	// Cycle time: 24h - 2h = 22h
	if result.CycleTime.Seconds < 79000 || result.CycleTime.Seconds > 80000 {
		t.Errorf("expected ~22h cycle time, got %.0fs", result.CycleTime.Seconds)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metrics/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Create internal/metrics/metrics.go with extracted logic**

Move computation logic from `cmd/metrics.go` into `internal/metrics/metrics.go`. Keep the same algorithm, export the types and functions:

```go
package metrics

import (
	"fmt"
	"regexp"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

var MoveDetailRe = regexp.MustCompile(`^status:\s*(\S+)\s*->\s*(\S+)$`)

type ColumnDwell struct {
	Total   time.Duration
	Count   int
	Average time.Duration
}

type Result struct {
	TotalItems int
	DoneItems  int
	Dwell      map[string]*ColumnDwell
	CycleTime  *DurationStat
	LeadTime   *DurationStat
	Throughput ThroughputStat
}

type DurationStat struct {
	Duration time.Duration
	Seconds  float64
	Display  string
}

type ThroughputStat struct {
	ThisWeek int
	LastWeek int
	Total    int
	PerWeek  float64
}

func Compute(items []*domain.Item, now time.Time) Result {
	dwells := make(map[string]*ColumnDwell)
	var cycleTimes, leadTimes []time.Duration
	var doneTimes []time.Time
	doneCount := 0

	for _, item := range items {
		if item.Status == "done" {
			doneCount++
		}
		doneTS, hasDone := processDwells(item, dwells)
		if !hasDone {
			continue
		}
		doneTimes = append(doneTimes, doneTS)
		ct := computeCycleTime(item, doneTS)
		if ct > 0 {
			cycleTimes = append(cycleTimes, ct)
		}
		lt := doneTS.Sub(item.CreatedAt)
		if lt > 0 {
			leadTimes = append(leadTimes, lt)
		}
	}

	for _, d := range dwells {
		if d.Count > 0 {
			d.Average = d.Total / time.Duration(d.Count)
		}
	}

	result := Result{
		TotalItems: len(items),
		DoneItems:  doneCount,
		Dwell:      dwells,
		Throughput: BuildThroughput(doneTimes, now),
	}
	if avg := AvgDuration(cycleTimes); avg > 0 {
		result.CycleTime = &DurationStat{Duration: avg, Seconds: avg.Seconds(), Display: FormatDuration(avg)}
	}
	if avg := AvgDuration(leadTimes); avg > 0 {
		result.LeadTime = &DurationStat{Duration: avg, Seconds: avg.Seconds(), Display: FormatDuration(avg)}
	}
	return result
}

func processDwells(item *domain.Item, dwells map[string]*ColumnDwell) (time.Time, bool) {
	var doneTS time.Time
	var hasDone bool

	for i, entry := range item.History {
		if entry.Action != "moved" {
			continue
		}
		m := MoveDetailRe.FindStringSubmatch(entry.Detail)
		if m == nil {
			continue
		}
		fromCol := m[1]
		toCol := m[2]

		enterTS := findEnterTime(item, fromCol, i)
		dwell := entry.Timestamp.Sub(enterTS)
		if dwell > 0 {
			addDwell(dwells, fromCol, dwell)
		}

		if toCol == "done" {
			doneTS = entry.Timestamp
			hasDone = true
		}
	}
	return doneTS, hasDone
}

func findEnterTime(item *domain.Item, col string, beforeIdx int) time.Time {
	for i := beforeIdx - 1; i >= 0; i-- {
		e := item.History[i]
		if e.Action == "moved" {
			m := MoveDetailRe.FindStringSubmatch(e.Detail)
			if m != nil && m[2] == col {
				return e.Timestamp
			}
		}
	}
	if col == "backlog" {
		return item.CreatedAt
	}
	if len(item.History) > 0 {
		return item.History[0].Timestamp
	}
	return item.CreatedAt
}

func addDwell(dwells map[string]*ColumnDwell, col string, d time.Duration) {
	if dwells[col] == nil {
		dwells[col] = &ColumnDwell{}
	}
	dwells[col].Total += d
	dwells[col].Count++
}

func computeCycleTime(item *domain.Item, doneTS time.Time) time.Duration {
	for _, e := range item.History {
		if e.Action != "moved" {
			continue
		}
		m := MoveDetailRe.FindStringSubmatch(e.Detail)
		if m != nil && m[2] == "in-progress" {
			return doneTS.Sub(e.Timestamp)
		}
	}
	return 0
}

func BuildThroughput(doneTimes []time.Time, now time.Time) ThroughputStat {
	thisWeekStart := WeekStart(now)
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)

	var thisWeek, lastWeek int
	for _, t := range doneTimes {
		if !t.Before(thisWeekStart) {
			thisWeek++
		} else if !t.Before(lastWeekStart) {
			lastWeek++
		}
	}

	var perWeek float64
	if len(doneTimes) > 0 {
		earliest := doneTimes[0]
		for _, t := range doneTimes[1:] {
			if t.Before(earliest) {
				earliest = t
			}
		}
		weeks := now.Sub(earliest).Hours() / (24 * 7)
		if weeks < 1 {
			weeks = 1
		}
		perWeek = float64(len(doneTimes)) / weeks
	}

	return ThroughputStat{
		ThisWeek: thisWeek,
		LastWeek: lastWeek,
		Total:    len(doneTimes),
		PerWeek:  perWeek,
	}
}

func WeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	d := t.AddDate(0, 0, -(weekday-1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, t.Location())
}

func AvgDuration(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total / time.Duration(len(ds))
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/metrics/ -v`
Expected: PASS

- [ ] **Step 5: Rewrite cmd/metrics.go as thin CLI wrapper**

Replace the computation logic in `cmd/metrics.go` with calls to `metrics.Compute()`. Keep the JSON/text output formatting in `cmd/metrics.go` since it's CLI-specific. The key change: remove all `computeMetrics`, `processDwells`, `computeCycleTime`, `buildThroughput`, `findEnterTime`, `addDwell`, `avgDuration`, `formatDuration`, `weekStart` functions. Replace `computeMetrics(items, time.Now())` call with `metrics.Compute(items, time.Now())`. Adapt the formatting functions to use `metrics.Result` instead of local `metricsResult`.

- [ ] **Step 6: Run existing tests to verify no regression**

Run: `go build ./... && go test ./...`
Expected: All pass, `ob metrics` still works

- [ ] **Step 7: Commit**

```bash
git add internal/metrics/metrics.go internal/metrics/metrics_test.go cmd/metrics.go
git commit -m "refactor: extract metrics computation into internal/metrics package"
```

---

## Chunk 2: New Metrics — Velocity, WIP, Burndown

### Task 2: Add DailyVelocity and RollingAverage

**Files:**
- Modify: `internal/metrics/metrics.go`
- Modify: `internal/metrics/metrics_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/metrics/metrics_test.go`:

```go
func TestDailyVelocity_NoDoneItems(t *testing.T) {
	items := []*domain.Item{
		{ID: "a", Status: "in-progress", History: []domain.ChangeRecord{}},
	}
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	result := DailyVelocity(items, 14, now)
	if len(result) != 14 {
		t.Errorf("expected 14 day slots, got %d", len(result))
	}
	for _, d := range result {
		if d.Count != 0 {
			t.Errorf("expected 0 count for %v, got %d", d.Date, d.Count)
		}
	}
}

func TestDailyVelocity_CountsEachDoneEvent(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []*domain.Item{
		{
			ID:     "a",
			Status: "done",
			History: []domain.ChangeRecord{
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-2 * time.Hour)},
			},
		},
		{
			ID:     "b",
			Status: "done",
			History: []domain.ChangeRecord{
				// Done, reopened, done again — counts twice
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-26 * time.Hour)},
				{Action: "moved", Detail: "status: done -> in-progress", Timestamp: now.Add(-25 * time.Hour)},
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-3 * time.Hour)},
			},
		},
	}
	result := DailyVelocity(items, 14, now)
	// Today (Mar 10): item a done + item b second done = 2
	todayCount := result[13].Count
	if todayCount != 2 {
		t.Errorf("expected 2 completions today, got %d", todayCount)
	}
	// Yesterday (Mar 9): item b first done = 1
	yesterdayCount := result[12].Count
	if yesterdayCount != 1 {
		t.Errorf("expected 1 completion yesterday, got %d", yesterdayCount)
	}
}

func TestRollingAverage(t *testing.T) {
	days := []DayCount{
		{Count: 3}, {Count: 0}, {Count: 6},
	}
	avg := RollingAverage(days, 3)
	if len(avg) != 3 {
		t.Fatalf("expected 3 values, got %d", len(avg))
	}
	// Window=3: [3], [3,0], [3,0,6] → 3.0, 1.5, 3.0
	if avg[0] != 3.0 {
		t.Errorf("expected avg[0]=3.0, got %f", avg[0])
	}
	if avg[1] != 1.5 {
		t.Errorf("expected avg[1]=1.5, got %f", avg[1])
	}
	if avg[2] != 3.0 {
		t.Errorf("expected avg[2]=3.0, got %f", avg[2])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metrics/ -run "TestDailyVelocity|TestRollingAverage" -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement DailyVelocity and RollingAverage**

Add to `internal/metrics/metrics.go`:

```go
type DayCount struct {
	Date  time.Time
	Count int
}

// DailyVelocity returns per-day completion counts for the last N days.
// Scans all item history for "moved to done" events independently —
// counts each done transition (including reopened items done again).
func DailyVelocity(items []*domain.Item, days int, now time.Time) []DayCount {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startDate := today.AddDate(0, 0, -(days - 1))

	result := make([]DayCount, days)
	for i := range result {
		result[i].Date = startDate.AddDate(0, 0, i)
	}

	for _, item := range items {
		for _, entry := range item.History {
			if entry.Action != "moved" {
				continue
			}
			m := MoveDetailRe.FindStringSubmatch(entry.Detail)
			if m == nil || m[2] != "done" {
				continue
			}
			entryDate := time.Date(entry.Timestamp.Year(), entry.Timestamp.Month(), entry.Timestamp.Day(), 0, 0, 0, 0, entry.Timestamp.Location())
			dayIdx := int(entryDate.Sub(startDate).Hours() / 24)
			if dayIdx >= 0 && dayIdx < days {
				result[dayIdx].Count++
			}
		}
	}
	return result
}

// RollingAverage computes a rolling average over the given window size.
func RollingAverage(days []DayCount, window int) []float64 {
	result := make([]float64, len(days))
	for i := range days {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		sum := 0
		for j := start; j <= i; j++ {
			sum += days[j].Count
		}
		result[i] = float64(sum) / float64(i-start+1)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metrics/ -run "TestDailyVelocity|TestRollingAverage" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metrics/metrics.go internal/metrics/metrics_test.go
git commit -m "feat: add DailyVelocity and RollingAverage metrics"
```

### Task 3: Add WIPStatus

**Files:**
- Modify: `internal/metrics/metrics.go`
- Modify: `internal/metrics/metrics_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/metrics/metrics_test.go`:

```go
func TestWIPStatus_WithLimits(t *testing.T) {
	board := &domain.Board{
		Columns: []domain.Column{
			{Name: "backlog", Limit: 20},
			{Name: "todo", Limit: 5},
			{Name: "in-progress", Limit: 3},
			{Name: "review", Limit: 5},
			{Name: "done", Limit: 0},
		},
		Items: map[string]*domain.Item{
			"a": {Status: "in-progress"},
			"b": {Status: "in-progress"},
			"c": {Status: "in-progress"},
			"d": {Status: "in-progress"},
			"e": {Status: "todo"},
			"f": {Status: "todo"},
			"g": {Status: "todo"},
			"h": {Status: "todo"},
		},
	}
	result := WIPStatus(board)
	// in-progress: 4/3 = over
	for _, col := range result {
		if col.Name == "in-progress" {
			if col.Count != 4 {
				t.Errorf("expected 4 in-progress, got %d", col.Count)
			}
			if col.Level != "over" {
				t.Errorf("expected 'over', got %q", col.Level)
			}
		}
		if col.Name == "todo" {
			if col.Count != 4 {
				t.Errorf("expected 4 todo, got %d", col.Count)
			}
			if col.Level != "warn" {
				t.Errorf("expected 'warn', got %q", col.Level)
			}
		}
	}
}

func TestWIPStatus_NoLimits(t *testing.T) {
	board := &domain.Board{
		Columns: []domain.Column{
			{Name: "backlog", Limit: 0},
		},
		Items: map[string]*domain.Item{
			"a": {Status: "backlog"},
		},
	}
	result := WIPStatus(board)
	if result[0].Level != "ok" {
		t.Errorf("expected 'ok' for no-limit column, got %q", result[0].Level)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metrics/ -run TestWIPStatus -v`
Expected: FAIL

- [ ] **Step 3: Implement WIPStatus**

Add to `internal/metrics/metrics.go`:

```go
type ColumnWIP struct {
	Name  string
	Count int
	Limit int
	Level string // "ok", "warn", "over"
}

// WIPStatus computes current item counts vs column limits.
func WIPStatus(board *domain.Board) []ColumnWIP {
	counts := make(map[string]int)
	for _, item := range board.Items {
		counts[item.Status]++
	}

	result := make([]ColumnWIP, 0, len(board.Columns))
	for _, col := range board.Columns {
		if col.Name == "done" {
			continue
		}
		wip := ColumnWIP{
			Name:  col.Name,
			Count: counts[col.Name],
			Limit: col.Limit,
			Level: "ok",
		}
		if col.Limit > 0 {
			ratio := float64(wip.Count) / float64(col.Limit)
			if ratio > 1.0 {
				wip.Level = "over"
			} else if ratio >= 0.8 {
				wip.Level = "warn"
			}
		}
		result = append(result, wip)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metrics/ -run TestWIPStatus -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metrics/metrics.go internal/metrics/metrics_test.go
git commit -m "feat: add WIPStatus metrics computation"
```

### Task 4: Add EpicBurndown

**Files:**
- Modify: `internal/metrics/metrics.go`
- Modify: `internal/metrics/metrics_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/metrics/metrics_test.go`:

```go
func TestEpicBurndown_BasicCase(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	epic := &domain.Item{
		ID:        "epic1",
		Type:      domain.ItemTypeEpic,
		CreatedAt: now.Add(-72 * time.Hour), // 3 days ago
	}
	children := []*domain.Item{
		{
			ID: "c1", ParentID: "epic1", Status: "done",
			History: []domain.ChangeRecord{
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-48 * time.Hour)},
			},
		},
		{
			ID: "c2", ParentID: "epic1", Status: "done",
			History: []domain.ChangeRecord{
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-24 * time.Hour)},
			},
		},
		{
			ID: "c3", ParentID: "epic1", Status: "in-progress",
			History: []domain.ChangeRecord{},
		},
	}
	points := EpicBurndown(epic, children, now)
	// Should have: start point (3 remaining), then 2 done events, then now point
	if len(points) < 3 {
		t.Fatalf("expected at least 3 burndown points, got %d", len(points))
	}
	// First point: all 3 remaining
	if points[0].Remaining != 3 {
		t.Errorf("expected 3 remaining at start, got %d", points[0].Remaining)
	}
	// Last point: 1 remaining
	last := points[len(points)-1]
	if last.Remaining != 1 {
		t.Errorf("expected 1 remaining at end, got %d", last.Remaining)
	}
}

func TestEpicBurndown_NoChildren(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	epic := &domain.Item{ID: "epic1", CreatedAt: now.Add(-24 * time.Hour)}
	points := EpicBurndown(epic, nil, now)
	if len(points) != 0 {
		t.Errorf("expected 0 points for no children, got %d", len(points))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metrics/ -run TestEpicBurndown -v`
Expected: FAIL

- [ ] **Step 3: Implement EpicBurndown**

Add to `internal/metrics/metrics.go`:

```go
// Add "sort" to the existing import block at the top of metrics.go

type BurndownPoint struct {
	Date      time.Time
	Remaining int
	Ideal     float64
}

// EpicBurndown computes remaining children over time for a given epic.
// Algorithm: start with total count, subtract one at each child's done timestamp.
func EpicBurndown(epic *domain.Item, children []*domain.Item, now time.Time) []BurndownPoint {
	total := len(children)
	if total == 0 {
		return nil
	}

	// Collect all "moved to done" timestamps from children
	type doneEvent struct {
		ts time.Time
	}
	var events []doneEvent
	for _, child := range children {
		for _, entry := range child.History {
			if entry.Action != "moved" {
				continue
			}
			m := MoveDetailRe.FindStringSubmatch(entry.Detail)
			if m != nil && m[2] == "done" {
				events = append(events, doneEvent{ts: entry.Timestamp})
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ts.Before(events[j].ts)
	})

	span := now.Sub(epic.CreatedAt)
	idealPerPoint := func(ts time.Time) float64 {
		if span <= 0 {
			return 0
		}
		elapsed := ts.Sub(epic.CreatedAt)
		return float64(total) * (1.0 - elapsed.Seconds()/span.Seconds())
	}

	points := make([]BurndownPoint, 0, len(events)+2)
	// Start point
	points = append(points, BurndownPoint{
		Date:      epic.CreatedAt,
		Remaining: total,
		Ideal:     float64(total),
	})

	remaining := total
	for _, ev := range events {
		remaining--
		points = append(points, BurndownPoint{
			Date:      ev.ts,
			Remaining: remaining,
			Ideal:     idealPerPoint(ev.ts),
		})
	}

	// End point (now)
	if len(events) == 0 || !events[len(events)-1].ts.Equal(now) {
		points = append(points, BurndownPoint{
			Date:      now,
			Remaining: remaining,
			Ideal:     idealPerPoint(now),
		})
	}

	return points
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metrics/ -run TestEpicBurndown -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metrics/metrics.go internal/metrics/metrics_test.go
git commit -m "feat: add EpicBurndown metrics computation"
```

### Task 5: Add BoardItems helper

**Files:**
- Modify: `internal/metrics/metrics.go`
- Modify: `internal/metrics/metrics_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBoardItems(t *testing.T) {
	board := &domain.Board{
		Items: map[string]*domain.Item{
			"a": {ID: "a"},
			"b": {ID: "b"},
		},
	}
	items := BoardItems(board)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/metrics/ -run TestBoardItems -v`
Expected: FAIL — function not defined

- [ ] **Step 3: Implement BoardItems**

Add to `internal/metrics/metrics.go`:

```go
// BoardItems converts a board's item map to a slice for metrics functions.
func BoardItems(board *domain.Board) []*domain.Item {
	items := make([]*domain.Item, 0, len(board.Items))
	for _, item := range board.Items {
		items = append(items, item)
	}
	return items
}
```

- [ ] **Step 4: Run all metrics tests**

Run: `go test ./internal/metrics/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metrics/metrics.go internal/metrics/metrics_test.go
git commit -m "feat: add BoardItems helper for map-to-slice conversion"
```

---

## Chunk 3: Chart Rendering Helpers

### Task 6: Create internal/tui/charts.go

**Files:**
- Create: `internal/tui/charts.go`
- Create: `internal/tui/charts_test.go`

- [ ] **Step 1: Write failing tests for RenderBarChart**

Create `internal/tui/charts_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestRenderBarChart_EmptyData(t *testing.T) {
	result := RenderBarChart(nil, 10, 5)
	if !strings.Contains(result, "No data") {
		t.Errorf("expected 'No data' message, got %q", result)
	}
}

func TestRenderBarChart_SingleBar(t *testing.T) {
	data := []BarData{{Label: "M01", Value: 5}}
	result := RenderBarChart(data, 20, 5)
	if !strings.Contains(result, "█") {
		t.Errorf("expected bar character in output, got %q", result)
	}
	if !strings.Contains(result, "M01") {
		t.Errorf("expected label M01 in output, got %q", result)
	}
}

func TestRenderHorizontalBars_Basic(t *testing.T) {
	data := []HBarData{
		{Label: "backlog", Value: 2.0, Display: "2.0d"},
		{Label: "in-progress", Value: 5.0, Display: "5.0d"},
	}
	result := RenderHorizontalBars(data, 30)
	if !strings.Contains(result, "backlog") {
		t.Errorf("expected 'backlog' label in output")
	}
	if !strings.Contains(result, "█") {
		t.Errorf("expected bar character in output")
	}
}

func TestRenderBurndown_EmptyData(t *testing.T) {
	result := RenderBurndown(nil, nil, 5, 20)
	if !strings.Contains(result, "No data") {
		t.Errorf("expected 'No data' message, got %q", result)
	}
}

func TestRenderBurndown_AllDone(t *testing.T) {
	result := RenderBurndown([]int{0, 0}, []float64{2, 0}, 5, 20)
	if !strings.Contains(result, "All done!") {
		t.Errorf("expected 'All done!' message, got %q", result)
	}
}

func TestRenderBurndown_WithData(t *testing.T) {
	remaining := []int{5, 4, 3, 2}
	ideal := []float64{5, 3.3, 1.7, 0}
	result := RenderBurndown(remaining, ideal, 5, 30)
	if !strings.Contains(result, "█") {
		t.Errorf("expected bar character in output, got %q", result)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestRender" -v`
Expected: FAIL

- [ ] **Step 3: Implement chart rendering helpers**

Create `internal/tui/charts.go`:

```go
package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

var (
	chartTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	barColor     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	avgLineColor = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	axisColor    = lipgloss.NewStyle().Faint(true)
	wipOk        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	wipWarn      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	wipOver      = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	burnColor    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	idealColor   = lipgloss.NewStyle().Faint(true)
)

type BarData struct {
	Label string
	Value int
}

// RenderBarChart renders a vertical bar chart using Unicode block characters.
// width = available character width, height = available rows for bars.
func RenderBarChart(data []BarData, width, height int) string {
	if len(data) == 0 {
		return centerText("No data", width)
	}

	maxVal := 0
	for _, d := range data {
		if d.Value > maxVal {
			maxVal = d.Value
		}
	}
	if maxVal == 0 {
		return centerText("No completed items in last 14 days", width)
	}

	// Render bars bottom-up
	barWidth := 3
	gap := 1
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		threshold := float64(maxVal) * float64(height-row) / float64(height)
		var line strings.Builder
		for i, d := range data {
			if i > 0 {
				line.WriteString(strings.Repeat(" ", gap))
			}
			frac := float64(d.Value) / float64(maxVal) * float64(height)
			barRow := float64(height - row)
			if frac >= barRow {
				line.WriteString(barColor.Render(strings.Repeat(string(blockChars[7]), barWidth)))
			} else if frac > barRow-1 && frac < barRow {
				idx := int((frac - math.Floor(frac)) * float64(len(blockChars)))
				if idx >= len(blockChars) {
					idx = len(blockChars) - 1
				}
				line.WriteString(barColor.Render(strings.Repeat(string(blockChars[idx]), barWidth)))
			} else {
				_ = threshold // suppress unused
				line.WriteString(strings.Repeat(" ", barWidth))
			}
		}
		rows[row] = line.String()
	}

	// X-axis labels
	var labels strings.Builder
	for i, d := range data {
		if i > 0 {
			labels.WriteString(strings.Repeat(" ", gap))
		}
		lbl := d.Label
		if len(lbl) > barWidth {
			lbl = lbl[:barWidth]
		}
		labels.WriteString(axisColor.Render(fmt.Sprintf("%-*s", barWidth, lbl)))
	}

	return strings.Join(rows, "\n") + "\n" + labels.String()
}

type HBarData struct {
	Label   string
	Value   float64
	Display string
}

// RenderHorizontalBars renders horizontal bar rows.
func RenderHorizontalBars(data []HBarData, width int) string {
	if len(data) == 0 {
		return "No data"
	}

	maxLabel := 0
	for _, d := range data {
		if len(d.Label) > maxLabel {
			maxLabel = len(d.Label)
		}
	}

	maxVal := 0.0
	for _, d := range data {
		if d.Value > maxVal {
			maxVal = d.Value
		}
	}

	barSpace := width - maxLabel - 2 - 6 // label + gap + value display
	if barSpace < 5 {
		barSpace = 5
	}

	var lines []string
	for _, d := range data {
		barLen := 0
		if maxVal > 0 {
			barLen = int(d.Value / maxVal * float64(barSpace))
		}
		if barLen < 1 && d.Value > 0 {
			barLen = 1
		}
		bar := barColor.Render(strings.Repeat("█", barLen))
		line := fmt.Sprintf("%-*s %s %s", maxLabel, d.Label, bar, axisColor.Render(d.Display))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// RenderBurndown renders a simple vertical burndown chart.
func RenderBurndown(remaining []int, ideal []float64, height int, width int) string {
	if len(remaining) == 0 {
		return centerText("No data", width)
	}

	maxVal := remaining[0]
	for _, v := range remaining {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return centerText("All done!", width)
	}

	barWidth := 2
	gap := 1
	rows := make([]string, height)
	for row := 0; row < height; row++ {
		barRow := float64(height - row)
		var line strings.Builder
		for i, val := range remaining {
			if i > 0 {
				line.WriteString(strings.Repeat(" ", gap))
			}
			frac := float64(val) / float64(maxVal) * float64(height)
			// Check ideal line position
			idealFrac := 0.0
			if i < len(ideal) {
				idealFrac = ideal[i] / float64(maxVal) * float64(height)
			}
			idealHere := idealFrac >= barRow-0.5 && idealFrac < barRow+0.5

			if frac >= barRow {
				line.WriteString(burnColor.Render(strings.Repeat(string(blockChars[7]), barWidth)))
			} else if idealHere {
				line.WriteString(idealColor.Render(strings.Repeat("·", barWidth)))
			} else {
				line.WriteString(strings.Repeat(" ", barWidth))
			}
		}
		rows[row] = line.String()
	}

	return strings.Join(rows, "\n")
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	pad := (width - len(text)) / 2
	return strings.Repeat(" ", pad) + text
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestRender" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/charts.go internal/tui/charts_test.go
git commit -m "feat: add terminal chart rendering helpers"
```

---

## Chunk 4: Dashboard View Integration

### Task 7: Add stateDashboard and DashboardModel to TUI

**Note:** Tasks 7 and 8 from prior drafts are merged because `app.go` references `DashboardModel` — both files must exist to compile.

**Files:**
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/app.go`
- Create: `internal/tui/dashboard.go`

- [ ] **Step 1: Add stateDashboard to viewState enum**

In `internal/tui/keys.go`, add `stateDashboard` after `stateConfirm`:

```go
const (
	stateBoard viewState = iota
	stateDetail
	statePicker
	stateInput
	stateConfirm
	stateDashboard
)
```

Also add a new picker kind for epic selection:

```go
const (
	pickerColumn pickerKind = iota
	pickerUser
	pickerItem
	pickerType
	pickerEpic
)
```

- [ ] **Step 2: Add dashboard field to App struct and D keybinding**

In `internal/tui/app.go`, add to App struct:

```go
type App struct {
	// ... existing fields ...
	dashboard  DashboardModel
}
```

In `handleBoardKey`, add case for `"D"` (shift+D):

```go
case "D":
	a.dashboard = newDashboardModel(a.board, a.width, a.height)
	a.prevState = stateBoard
	a.state = stateDashboard
```

In `View()`, add case for `stateDashboard`:

```go
case stateDashboard:
	a.dashboard.SetSize(a.width, a.height)
	return a.dashboard.View()
```

In `handleKey`, add case:

```go
case stateDashboard:
	return a.handleDashboardKey(msg)
```

- [ ] **Step 3: Add handleDashboardKey method**

In `internal/tui/app.go`, add:

```go
func (a App) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "D", "esc":
		a.state = stateBoard
	case "q", "ctrl+c":
		return a.quit()
	case "tab":
		a.dashboard.NextPanel()
	case "E":
		epics := epicPickerLabels(a.board)
		if len(epics) > 0 {
			a.picker = newPickerModel("Select epic for burndown:", pickerEpic, epics)
			a.state = statePicker
			a.prevState = stateDashboard
		}
	case "R", "r":
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
	}
	return a, nil
}
```

Add helper to build epic picker labels:

```go
func epicPickerLabels(board *domain.Board) []string {
	var labels []string
	for _, item := range board.Items {
		if item.Type == domain.ItemTypeEpic {
			labels = append(labels, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
		}
	}
	return labels
}
```

In `executePickerSelection`, add case for `pickerEpic`:

```go
case pickerEpic:
	epicNum := extractItemNum(selected)
	a.dashboard.SelectEpic(a.board, epicNum)
	a.state = stateDashboard
	return a, nil
```

- [ ] **Step 4: Create DashboardModel**

Create `internal/tui/dashboard.go`:

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/metrics"
)

type dashPanel int

const (
	panelWIP dashPanel = iota
	panelVelocity
	panelCycleTime
	panelBurndown
	panelCount // sentinel
)

type DashboardModel struct {
	board       *domain.Board
	width       int
	height      int
	activePanel dashPanel

	// Computed metrics
	wip       []metrics.ColumnWIP
	velocity  []metrics.DayCount
	rollingAvg []float64
	metricsR  metrics.Result
	burndown  []metrics.BurndownPoint
	epicTitle string
	epicTotal int
}

func newDashboardModel(board *domain.Board, w, h int) DashboardModel {
	items := metrics.BoardItems(board)
	now := time.Now()
	m := DashboardModel{
		board:    board,
		width:    w,
		height:   h,
		wip:      metrics.WIPStatus(board),
		velocity: metrics.DailyVelocity(items, 14, now),
		metricsR: metrics.Compute(items, now),
	}
	m.rollingAvg = metrics.RollingAverage(m.velocity, 3)
	m.selectFirstEpic(board)
	return m
}

func (d *DashboardModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DashboardModel) NextPanel() {
	d.activePanel = (d.activePanel + 1) % panelCount
}

func (d *DashboardModel) SelectEpic(board *domain.Board, ref string) {
	itemID := board.ResolveID(ref)
	epic, ok := board.Items[itemID]
	if !ok || epic.Type != domain.ItemTypeEpic {
		return
	}
	children := childrenOf(board, epic.ID)
	d.burndown = metrics.EpicBurndown(epic, children, time.Now())
	d.epicTitle = fmt.Sprintf("#%d %s", epic.DisplayNum, epic.Title)
	d.epicTotal = len(children)
}

func (d *DashboardModel) selectFirstEpic(board *domain.Board) {
	for _, item := range board.Items {
		if item.Type == domain.ItemTypeEpic && item.Status != "done" {
			d.SelectEpic(board, item.ID)
			return
		}
	}
}

func childrenOf(board *domain.Board, parentID string) []*domain.Item {
	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children
}

func (d DashboardModel) View() string {
	if d.width < 80 || d.height < 24 {
		return centerText("Terminal too small (need 80x24)", d.width)
	}

	var sections []string

	// Title
	title := chartTitle.Render("OBEYA DASHBOARD")
	sections = append(sections, lipgloss.PlaceHorizontal(d.width, lipgloss.Center, title))

	// WIP status bar
	sections = append(sections, d.renderWIPBar())

	// Velocity chart
	sections = append(sections, d.renderVelocity())

	// Bottom panels
	narrow := d.width < 100
	if narrow {
		sections = append(sections, d.renderCycleTime(d.width-4))
		sections = append(sections, d.renderBurndownPanel(d.width-4))
	} else {
		halfW := (d.width - 4) / 2
		left := d.renderCycleTime(halfW)
		right := d.renderBurndownPanel(halfW)
		sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
	}

	// Help bar
	help := helpStyle.Render("D: board · Esc: back · Tab: panels · E: epic · R: refresh")
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

func (d DashboardModel) renderWIPBar() string {
	var parts []string
	parts = append(parts, chartTitle.Render("WIP"))
	for _, col := range d.wip {
		var style lipgloss.Style
		switch col.Level {
		case "over":
			style = wipOver
		case "warn":
			style = wipWarn
		default:
			style = wipOk
		}
		if col.Limit > 0 {
			parts = append(parts, style.Render(fmt.Sprintf("%s %d/%d", col.Name, col.Count, col.Limit)))
		} else {
			parts = append(parts, fmt.Sprintf("%s %d", col.Name, col.Count))
		}
	}
	// Append cycle/lead time
	ct := "—"
	if d.metricsR.CycleTime != nil {
		ct = d.metricsR.CycleTime.Display
	}
	lt := "—"
	if d.metricsR.LeadTime != nil {
		lt = d.metricsR.LeadTime.Display
	}
	parts = append(parts, axisColor.Render(fmt.Sprintf("│ cycle: %s · lead: %s", ct, lt)))

	line := strings.Join(parts, "  ")
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.width - 2).
		Padding(0, 1)
	if d.activePanel == panelWIP {
		border = border.BorderForeground(lipgloss.Color("14"))
	}
	return border.Render(line)
}

func (d DashboardModel) renderVelocity() string {
	barData := make([]BarData, len(d.velocity))
	for i, v := range d.velocity {
		// Label format: first char of month + day, e.g. "M10" for Mar 10
		month := v.Date.Format("Jan")
		lbl := string([]rune(month)[0]) + v.Date.Format("02")
		barData[i] = BarData{
			Label: lbl,
			Value: v.Count,
		}
	}

	totalCount := 0
	for _, v := range d.velocity {
		totalCount += v.Count
	}
	avg := float64(totalCount) / float64(len(d.velocity))

	velocityH := d.height / 3
	if velocityH < 4 {
		velocityH = 4
	}

	header := chartTitle.Render(fmt.Sprintf("VELOCITY (14d) — avg %.1f/day", avg))
	chart := RenderBarChart(barData, d.width-4, velocityH)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(d.width - 2).
		Padding(0, 1)
	if d.activePanel == panelVelocity {
		border = border.BorderForeground(lipgloss.Color("14"))
	}
	return border.Render(header + "\n" + chart)
}

func (d DashboardModel) renderCycleTime(width int) string {
	var data []HBarData
	cols := []string{"backlog", "todo", "in-progress", "review"}
	for _, col := range cols {
		dwell, ok := d.metricsR.Dwell[col]
		if !ok {
			data = append(data, HBarData{Label: col, Value: 0, Display: "—"})
			continue
		}
		days := dwell.Average.Hours() / 24
		data = append(data, HBarData{
			Label:   col,
			Value:   days,
			Display: metrics.FormatDuration(dwell.Average),
		})
	}

	header := chartTitle.Render("CYCLE TIME")
	chart := RenderHorizontalBars(data, width-4)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(width).
		Padding(0, 1)
	if d.activePanel == panelCycleTime {
		border = border.BorderForeground(lipgloss.Color("14"))
	}
	return border.Render(header + "\n" + chart)
}

func (d DashboardModel) renderBurndownPanel(width int) string {
	header := chartTitle.Render("BURNDOWN")

	if d.epicTitle == "" {
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Width(width).
			Padding(0, 1)
		if d.activePanel == panelBurndown {
			border = border.BorderForeground(lipgloss.Color("14"))
		}
		return border.Render(header + "\n" + centerText("No epics on board", width-4))
	}

	remaining := make([]int, len(d.burndown))
	ideal := make([]float64, len(d.burndown))
	for i, p := range d.burndown {
		remaining[i] = p.Remaining
		ideal[i] = p.Ideal
	}

	lastRemaining := 0
	if len(remaining) > 0 {
		lastRemaining = remaining[len(remaining)-1]
	}

	titleLine := fmt.Sprintf("%s: %s — %d/%d left", header, d.epicTitle, lastRemaining, d.epicTotal)
	burnH := d.height / 4
	if burnH < 3 {
		burnH = 3
	}
	chart := RenderBurndown(remaining, ideal, burnH, width-4)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Width(width).
		Padding(0, 1)
	if d.activePanel == panelBurndown {
		border = border.BorderForeground(lipgloss.Color("14"))
	}
	return border.Render(titleLine + "\n" + chart)
}
```

- [ ] **Step 5: Verify the full project compiles**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 6: Manual test the dashboard**

Run: `go run . tui` then press `D` to toggle dashboard view. Verify:
- WIP status bar renders with column counts
- Velocity chart shows bars (or "No completed items" message)
- Cycle time shows horizontal bars
- Burndown shows chart or "No epics" message
- `D`/`Esc` returns to board, `Tab` cycles panel focus, `R` refreshes

- [ ] **Step 7: Commit**

```bash
git add internal/tui/dashboard.go internal/tui/keys.go internal/tui/app.go
git commit -m "feat: add dashboard view to TUI with velocity, WIP, cycle time, and burndown"
```

---

## Chunk 5: Update boardFileChangedMsg to Refresh Dashboard

### Task 8: Dashboard recomputes on file changes

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Update boardLoadedMsg handler**

In `app.go` `Update()`, update the `boardLoadedMsg` handler to also refresh the dashboard when in dashboard state. This is sufficient for file-change reactivity because `boardFileChangedMsg` already triggers `loadBoard()`, which sends a `boardLoadedMsg` when complete:

```go
case boardLoadedMsg:
	a.board = msg.board
	a.columns = extractColumns(msg.board)
	a.clampCursor()
	if a.state == stateDashboard {
		a.dashboard = newDashboardModel(a.board, a.width, a.height)
	}
	return a, nil
```

- [ ] **Step 2: Verify it compiles and works**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: dashboard auto-refreshes on board file changes"
```

---

## Chunk 6: Final Verification

### Task 9: Run full test suite and verify

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues

- [ ] **Step 3: Build and verify CLI metrics still works**

Run: `go build -o ob . && ./ob metrics`
Expected: Same output format as before the refactor

- [ ] **Step 4: Manual TUI smoke test**

Run: `./ob tui`, verify:
1. Board view works as before
2. Press `D` → dashboard appears
3. Press `Tab` → panel highlight cycles
4. Press `E` → epic picker appears (if epics exist)
5. Press `D` or `Esc` → back to board
6. Press `R` → dashboard refreshes

- [ ] **Step 5: Final commit if any fixes were needed**

```bash
git add -A && git commit -m "fix: address issues found during dashboard verification"
```
