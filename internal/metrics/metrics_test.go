package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

func TestCompute_EmptyItems(t *testing.T) {
	result := Compute(nil, time.Now())
	if result.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", result.TotalItems)
	}
	if result.DoneItems != 0 {
		t.Errorf("DoneItems = %d, want 0", result.DoneItems)
	}
	if result.CycleTime != nil {
		t.Errorf("CycleTime = %v, want nil", result.CycleTime)
	}
	if result.LeadTime != nil {
		t.Errorf("LeadTime = %v, want nil", result.LeadTime)
	}
	if result.Throughput.Total != 0 {
		t.Errorf("Throughput.Total = %d, want 0", result.Throughput.Total)
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
		t.Errorf("TotalItems = %d, want 1", result.TotalItems)
	}
	if result.DoneItems != 1 {
		t.Errorf("DoneItems = %d, want 1", result.DoneItems)
	}
	if result.CycleTime == nil {
		t.Fatal("CycleTime is nil, want non-nil")
	}
	// Cycle time: in-progress -> done = 22h
	expectedCycle := 22 * time.Hour
	if result.CycleTime.Duration != expectedCycle {
		t.Errorf("CycleTime.Duration = %v, want %v", result.CycleTime.Duration, expectedCycle)
	}
	if result.LeadTime == nil {
		t.Fatal("LeadTime is nil, want non-nil")
	}
	// Lead time: created -> done = 46h
	expectedLead := 46 * time.Hour
	if result.LeadTime.Duration != expectedLead {
		t.Errorf("LeadTime.Duration = %v, want %v", result.LeadTime.Duration, expectedLead)
	}
	if result.Throughput.Total != 1 {
		t.Errorf("Throughput.Total = %d, want 1", result.Throughput.Total)
	}
}

func TestCompute_MultipleItems(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []*domain.Item{
		{
			ID: "item1", Status: "done",
			CreatedAt: now.Add(-72 * time.Hour),
			History: []domain.ChangeRecord{
				{Action: "created", Detail: "created task: A", Timestamp: now.Add(-72 * time.Hour)},
				{Action: "moved", Detail: "status: backlog -> in-progress", Timestamp: now.Add(-48 * time.Hour)},
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-24 * time.Hour)},
			},
		},
		{
			ID: "item2", Status: "in-progress",
			CreatedAt: now.Add(-24 * time.Hour),
			History: []domain.ChangeRecord{
				{Action: "created", Detail: "created task: B", Timestamp: now.Add(-24 * time.Hour)},
				{Action: "moved", Detail: "status: backlog -> in-progress", Timestamp: now.Add(-12 * time.Hour)},
			},
		},
	}
	result := Compute(items, now)

	if result.TotalItems != 2 {
		t.Errorf("TotalItems = %d, want 2", result.TotalItems)
	}
	if result.DoneItems != 1 {
		t.Errorf("DoneItems = %d, want 1", result.DoneItems)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
		{48 * time.Hour, "2d"},
		{50 * time.Hour, "2d 2h"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestAvgDuration(t *testing.T) {
	if got := AvgDuration(nil); got != 0 {
		t.Errorf("AvgDuration(nil) = %v, want 0", got)
	}
	ds := []time.Duration{10 * time.Hour, 20 * time.Hour}
	if got := AvgDuration(ds); got != 15*time.Hour {
		t.Errorf("AvgDuration = %v, want %v", got, 15*time.Hour)
	}
}

func TestWeekStart(t *testing.T) {
	// Tuesday 2026-03-10 should give Monday 2026-03-09
	tue := time.Date(2026, 3, 10, 15, 30, 0, 0, time.UTC)
	ws := WeekStart(tue)
	expected := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	if !ws.Equal(expected) {
		t.Errorf("WeekStart(%v) = %v, want %v", tue, ws, expected)
	}
}

func TestBuildThroughput(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	thisWeekStart := WeekStart(now) // Monday 2026-03-09
	doneTimes := []time.Time{
		thisWeekStart.Add(2 * time.Hour),  // this week
		thisWeekStart.Add(-24 * time.Hour), // last week
		thisWeekStart.Add(-48 * time.Hour), // last week
	}
	tp := BuildThroughput(doneTimes, now)
	if tp.ThisWeek != 1 {
		t.Errorf("ThisWeek = %d, want 1", tp.ThisWeek)
	}
	if tp.LastWeek != 2 {
		t.Errorf("LastWeek = %d, want 2", tp.LastWeek)
	}
	if tp.Total != 3 {
		t.Errorf("Total = %d, want 3", tp.Total)
	}
	if math.IsNaN(tp.PerWeek) || tp.PerWeek <= 0 {
		t.Errorf("PerWeek = %f, want > 0", tp.PerWeek)
	}
}

func TestDailyVelocity_NoDoneItems(t *testing.T) {
	items := []*domain.Item{{ID: "a", Status: "in-progress", History: []domain.ChangeRecord{}}}
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
		{ID: "a", Status: "done", History: []domain.ChangeRecord{
			{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-2 * time.Hour)},
		}},
		{ID: "b", Status: "done", History: []domain.ChangeRecord{
			{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-26 * time.Hour)},
			{Action: "moved", Detail: "status: done -> in-progress", Timestamp: now.Add(-25 * time.Hour)},
			{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-3 * time.Hour)},
		}},
	}
	result := DailyVelocity(items, 14, now)
	if result[13].Count != 2 {
		t.Errorf("expected 2 today, got %d", result[13].Count)
	}
	if result[12].Count != 1 {
		t.Errorf("expected 1 yesterday, got %d", result[12].Count)
	}
}

func TestRollingAverage(t *testing.T) {
	days := []DayCount{{Count: 3}, {Count: 0}, {Count: 6}}
	avg := RollingAverage(days, 3)
	if len(avg) != 3 {
		t.Fatalf("expected 3 values, got %d", len(avg))
	}
	if avg[0] != 3.0 {
		t.Errorf("expected 3.0, got %f", avg[0])
	}
	if avg[1] != 1.5 {
		t.Errorf("expected 1.5, got %f", avg[1])
	}
	if avg[2] != 3.0 {
		t.Errorf("expected 3.0, got %f", avg[2])
	}
}

func TestDwellComputation(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []*domain.Item{
		{
			ID: "d1", Status: "done",
			CreatedAt: now.Add(-72 * time.Hour),
			History: []domain.ChangeRecord{
				{Action: "created", Detail: "created task", Timestamp: now.Add(-72 * time.Hour)},
				{Action: "moved", Detail: "status: backlog -> in-progress", Timestamp: now.Add(-48 * time.Hour)},
				{Action: "moved", Detail: "status: in-progress -> done", Timestamp: now.Add(-24 * time.Hour)},
			},
		},
	}
	result := Compute(items, now)

	// backlog dwell: 72h - 48h = 24h
	if d, ok := result.Dwell["backlog"]; !ok {
		t.Error("missing backlog dwell")
	} else if d.Average != 24*time.Hour {
		t.Errorf("backlog dwell = %v, want 24h", d.Average)
	}

	// in-progress dwell: 48h - 24h = 24h
	if d, ok := result.Dwell["in-progress"]; !ok {
		t.Error("missing in-progress dwell")
	} else if d.Average != 24*time.Hour {
		t.Errorf("in-progress dwell = %v, want 24h", d.Average)
	}
}
