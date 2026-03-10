package metrics

import (
	"fmt"
	"regexp"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

// MoveDetailRe matches history entries like "status: backlog -> in-progress".
var MoveDetailRe = regexp.MustCompile(`^status:\s*(\S+)\s*->\s*(\S+)$`)

// ColumnDwell tracks total and average dwell time for a single column.
type ColumnDwell struct {
	Total   time.Duration
	Count   int
	Average time.Duration
}

// DurationStat holds a computed duration with display string.
type DurationStat struct {
	Duration time.Duration
	Seconds  float64
	Display  string
}

// ThroughputStat holds weekly throughput numbers.
type ThroughputStat struct {
	ThisWeek int
	LastWeek int
	Total    int
	PerWeek  float64
}

// Result holds all computed metrics for a board.
type Result struct {
	TotalItems int
	DoneItems  int
	Dwell      map[string]*ColumnDwell
	CycleTime  *DurationStat
	LeadTime   *DurationStat
	Throughput ThroughputStat
}

// Compute calculates board metrics from the given items and reference time.
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
		result.CycleTime = &DurationStat{
			Duration: avg,
			Seconds:  avg.Seconds(),
			Display:  FormatDuration(avg),
		}
	}
	if avg := AvgDuration(leadTimes); avg > 0 {
		result.LeadTime = &DurationStat{
			Duration: avg,
			Seconds:  avg.Seconds(),
			Display:  FormatDuration(avg),
		}
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

// BuildThroughput computes weekly throughput from done timestamps.
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

// WeekStart returns the Monday 00:00 of the week containing t.
func WeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	d := t.AddDate(0, 0, -(weekday-1))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, t.Location())
}

// AvgDuration returns the average of the given durations, or 0 if empty.
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

// FormatDuration formats a duration as a human-readable string.
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
