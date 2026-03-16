package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReviewContext_JSONRoundTrip(t *testing.T) {
	rc := ReviewContext{
		Purpose: "Replace cookie sessions",
		FilesChanged: []FileChange{
			{Path: "auth/middleware.go", Added: 82, Removed: 41, Diff: "+ new line\n- old line"},
		},
		TestsWritten: []TestResult{
			{Name: "TestJWT", Passed: true},
		},
		Proof: []ProofItem{
			{Check: "go vet", Status: "pass"},
			{Check: "edge cases", Status: "fail", Detail: "no concurrency tests"},
		},
		Reasoning: "JWT for debuggability",
		Reproduce: []string{"go test ./auth/ -run TestJWT"},
	}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ReviewContext
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Purpose != rc.Purpose {
		t.Errorf("Purpose = %q, want %q", got.Purpose, rc.Purpose)
	}
	if len(got.FilesChanged) != 1 || got.FilesChanged[0].Diff != rc.FilesChanged[0].Diff {
		t.Error("FilesChanged roundtrip failed")
	}
	if len(got.Reproduce) != 1 || got.Reproduce[0] != "go test ./auth/ -run TestJWT" {
		t.Error("Reproduce roundtrip failed")
	}
}

func TestHumanReview_JSONRoundTrip(t *testing.T) {
	hr := HumanReview{
		Status:     "reviewed",
		ReviewedBy: "user-123",
		ReviewedAt: time.Now().Truncate(time.Second),
	}
	data, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got HumanReview
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "reviewed" || got.ReviewedBy != "user-123" {
		t.Error("HumanReview roundtrip failed")
	}
}

func TestItem_ConfidencePointer(t *testing.T) {
	// nil = unset
	item := Item{ID: "a", Title: "test"}
	data, _ := json.Marshal(item)
	if string(data) != "" && json.Valid(data) {
		var got Item
		json.Unmarshal(data, &got)
		if got.Confidence != nil {
			t.Error("nil Confidence should remain nil after roundtrip")
		}
	}

	// explicit 0 = agent reports 0%
	zero := 0
	item.Confidence = &zero
	data, _ = json.Marshal(item)
	var got Item
	json.Unmarshal(data, &got)
	if got.Confidence == nil || *got.Confidence != 0 {
		t.Error("explicit 0 Confidence should survive roundtrip")
	}
}

func TestItem_BackwardCompatible(t *testing.T) {
	// Old JSON without new fields should deserialize cleanly
	oldJSON := `{"id":"abc","display_num":1,"type":"task","title":"old","status":"done","priority":"medium","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`
	var item Item
	if err := json.Unmarshal([]byte(oldJSON), &item); err != nil {
		t.Fatalf("backward compat unmarshal failed: %v", err)
	}
	if item.Sponsor != "" {
		t.Error("Sponsor should be empty for old items")
	}
	if item.Confidence != nil {
		t.Error("Confidence should be nil for old items")
	}
	if item.ReviewContext != nil {
		t.Error("ReviewContext should be nil for old items")
	}
	if item.HumanReview != nil {
		t.Error("HumanReview should be nil for old items")
	}
}
