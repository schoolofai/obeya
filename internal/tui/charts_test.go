package tui

import (
	"strings"
	"testing"
)

func TestRenderBarChart_EmptyData(t *testing.T) {
	result := RenderBarChart(nil, 10, 5)
	if !strings.Contains(result, "No data") {
		t.Errorf("expected 'No data', got %q", result)
	}
}

func TestRenderBarChart_AllZero(t *testing.T) {
	data := []BarData{{Label: "M01", Value: 0}, {Label: "M02", Value: 0}}
	result := RenderBarChart(data, 40, 5)
	if !strings.Contains(result, "No completed items in last 14 days") {
		t.Errorf("expected zero-data message, got %q", result)
	}
}

func TestRenderBarChart_SingleBar(t *testing.T) {
	data := []BarData{{Label: "M01", Value: 5}}
	result := RenderBarChart(data, 20, 5)
	if !strings.Contains(result, "█") {
		t.Errorf("expected bar char")
	}
	if !strings.Contains(result, "M01") {
		t.Errorf("expected label")
	}
}

func TestRenderHorizontalBars_Basic(t *testing.T) {
	data := []HBarData{
		{Label: "backlog", Value: 2.0, Display: "2.0d"},
		{Label: "in-progress", Value: 5.0, Display: "5.0d"},
	}
	result := RenderHorizontalBars(data, 30)
	if !strings.Contains(result, "backlog") {
		t.Error("expected 'backlog'")
	}
	if !strings.Contains(result, "█") {
		t.Error("expected bar char")
	}
}

func TestRenderHorizontalBars_Empty(t *testing.T) {
	result := RenderHorizontalBars(nil, 30)
	if !strings.Contains(result, "No data") {
		t.Error("expected 'No data'")
	}
}

func TestRenderBurndown_EmptyData(t *testing.T) {
	result := RenderBurndown(nil, nil, 5, 20)
	if !strings.Contains(result, "No data") {
		t.Error("expected 'No data'")
	}
}

func TestRenderBurndown_AllDone(t *testing.T) {
	result := RenderBurndown([]int{0, 0}, []float64{2, 0}, 5, 20)
	if !strings.Contains(result, "All done!") {
		t.Error("expected 'All done!'")
	}
}

func TestRenderBurndown_WithData(t *testing.T) {
	remaining := []int{5, 4, 3, 2}
	ideal := []float64{5, 3.3, 1.7, 0}
	result := RenderBurndown(remaining, ideal, 5, 30)
	if !strings.Contains(result, "█") {
		t.Error("expected bar char")
	}
}

func TestCenterText(t *testing.T) {
	result := centerText("hi", 10)
	if !strings.Contains(result, "hi") {
		t.Error("expected text")
	}
	if len(result) != 6 { // 4 spaces + 2 chars
		t.Errorf("expected length 6, got %d: %q", len(result), result)
	}
}
