package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/metrics"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show board completion stats and throughput",
	Long:  "Display average dwell time per column, cycle time, lead time, and weekly throughput.",
	RunE:  runMetrics,
}

func init() {
	rootCmd.AddCommand(metricsCmd)
	metricsCmd.Flags().String("format", "", "output format (json)")
}

// JSON output types for CLI serialization.

type metricsResult struct {
	TotalItems int                  `json:"total_items"`
	DoneItems  int                  `json:"done_items"`
	Dwell      map[string]dwellJSON `json:"column_dwell"`
	CycleTime  *durationJSON        `json:"cycle_time,omitempty"`
	LeadTime   *durationJSON        `json:"lead_time,omitempty"`
	Throughput throughputJSON        `json:"throughput"`
}

type dwellJSON struct {
	AverageSeconds float64 `json:"average_seconds"`
	Display        string  `json:"display"`
	Count          int     `json:"count"`
}

type durationJSON struct {
	Seconds float64 `json:"seconds"`
	Display string  `json:"display"`
}

type throughputJSON struct {
	ThisWeek int     `json:"this_week"`
	LastWeek int     `json:"last_week"`
	Total    int     `json:"total"`
	PerWeek  float64 `json:"per_week"`
}

func runMetrics(cmd *cobra.Command, _ []string) error {
	eng, err := getEngine()
	if err != nil {
		return err
	}

	items, err := eng.ListItems(engine.ListFilter{})
	if err != nil {
		return err
	}

	computed := metrics.Compute(items, time.Now())
	result := convertResult(computed)

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return printMetricsJSON(result)
	}

	printMetricsText(result)
	return nil
}

func convertResult(r metrics.Result) metricsResult {
	mr := metricsResult{
		TotalItems: r.TotalItems,
		DoneItems:  r.DoneItems,
		Dwell:      buildDwellJSON(r.Dwell),
		Throughput: throughputJSON{
			ThisWeek: r.Throughput.ThisWeek,
			LastWeek: r.Throughput.LastWeek,
			Total:    r.Throughput.Total,
			PerWeek:  r.Throughput.PerWeek,
		},
	}
	if r.CycleTime != nil {
		mr.CycleTime = &durationJSON{
			Seconds: r.CycleTime.Seconds,
			Display: r.CycleTime.Display,
		}
	}
	if r.LeadTime != nil {
		mr.LeadTime = &durationJSON{
			Seconds: r.LeadTime.Seconds,
			Display: r.LeadTime.Display,
		}
	}
	return mr
}

func buildDwellJSON(dwells map[string]*metrics.ColumnDwell) map[string]dwellJSON {
	out := make(map[string]dwellJSON)
	for col, d := range dwells {
		if col == "done" {
			continue
		}
		out[col] = dwellJSON{
			AverageSeconds: d.Average.Seconds(),
			Display:        metrics.FormatDuration(d.Average),
			Count:          d.Count,
		}
	}
	return out
}

func printMetricsText(r metricsResult) {
	fmt.Fprintf(os.Stdout, "Board Metrics (%d items, %d done)\n\n", r.TotalItems, r.DoneItems)

	printDwellTable(r.Dwell)
	printTimingSummary(r.CycleTime, r.LeadTime)
	printThroughputSummary(r.Throughput)
}

func printDwellTable(dwell map[string]dwellJSON) {
	cols := []string{"backlog", "todo", "in-progress", "review"}
	fmt.Fprintln(os.Stdout, "Column Dwell Time (avg):")
	for _, col := range cols {
		d, ok := dwell[col]
		display := "—"
		if ok && d.Count > 0 {
			display = d.Display
		}
		fmt.Fprintf(os.Stdout, "  %-16s %s\n", col, display)
	}
	fmt.Fprintln(os.Stdout)
}

func printTimingSummary(cycleTime, leadTime *durationJSON) {
	ct := "—"
	if cycleTime != nil {
		ct = cycleTime.Display
	}
	lt := "—"
	if leadTime != nil {
		lt = leadTime.Display
	}
	fmt.Fprintf(os.Stdout, "Cycle Time (in-progress → done):  %s\n", ct)
	fmt.Fprintf(os.Stdout, "Lead Time  (created → done):      %s\n\n", lt)
}

func printThroughputSummary(t throughputJSON) {
	fmt.Fprintln(os.Stdout, "Throughput:")
	fmt.Fprintf(os.Stdout, "  This week:   %d items\n", t.ThisWeek)
	fmt.Fprintf(os.Stdout, "  Last week:   %d items\n", t.LastWeek)

	perWeekStr := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", t.PerWeek), "0"), ".")
	fmt.Fprintf(os.Stdout, "  All time:    %d items (%s/week)\n", t.Total, perWeekStr)
}

func printMetricsJSON(r metricsResult) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
