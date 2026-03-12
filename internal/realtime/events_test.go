package realtime

import (
	"testing"
)

func TestWebsocketURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{"https endpoint", "https://cloud.appwrite.io/v1", "wss://cloud.appwrite.io/v1/realtime"},
		{"http endpoint", "http://localhost:8080/v1", "ws://localhost:8080/v1/realtime"},
		{"trailing slash", "https://cloud.appwrite.io/v1/", "wss://cloud.appwrite.io/v1/realtime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := websocketURL(tt.endpoint)
			if got != tt.want {
				t.Errorf("websocketURL(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestParseEventAction(t *testing.T) {
	tests := []struct {
		name       string
		eventStr   string
		wantAction EventAction
		wantOk     bool
	}{
		{"create event", "databases.obeya.collections.items.documents.doc123.create", ActionCreate, true},
		{"update event", "databases.obeya.collections.items.documents.doc123.update", ActionUpdate, true},
		{"delete event", "databases.obeya.collections.items.documents.doc123.delete", ActionDelete, true},
		{"unknown event suffix", "databases.obeya.collections.items.documents.doc123.unknown", "", false},
		{"empty string", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, ok := parseEventAction(tt.eventStr)
			if ok != tt.wantOk {
				t.Errorf("parseEventAction(%q) ok = %v, want %v", tt.eventStr, ok, tt.wantOk)
			}
			if action != tt.wantAction {
				t.Errorf("parseEventAction(%q) action = %q, want %q", tt.eventStr, action, tt.wantAction)
			}
		})
	}
}

func TestParseCollectionID(t *testing.T) {
	tests := []struct {
		name     string
		eventStr string
		want     string
	}{
		{"items collection", "databases.obeya.collections.items.documents.doc123.create", "items"},
		{"item_history collection", "databases.obeya.collections.item_history.documents.doc456.create", "item_history"},
		{"short string", "too.short", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCollectionID(tt.eventStr)
			if got != tt.want {
				t.Errorf("parseCollectionID(%q) = %q, want %q", tt.eventStr, got, tt.want)
			}
		})
	}
}

func TestParseRealtimeMessage(t *testing.T) {
	msg := map[string]interface{}{
		"events": []interface{}{
			"databases.obeya.collections.items.documents.item-42.update",
		},
		"payload": map[string]interface{}{
			"$id":      "item-42",
			"board_id": "board-xyz",
			"title":    "Updated title",
			"status":   "in-progress",
		},
	}

	event := parseRealtimeMessage(msg, "board-xyz")
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	if event.Action != ActionUpdate {
		t.Errorf("expected update action, got %s", event.Action)
	}
	if event.DocumentID != "item-42" {
		t.Errorf("expected doc ID item-42, got %s", event.DocumentID)
	}
	if event.CollectionID != "items" {
		t.Errorf("expected collection items, got %s", event.CollectionID)
	}

	event = parseRealtimeMessage(msg, "wrong-board")
	if event != nil {
		t.Fatal("expected nil for wrong board_id")
	}
}
