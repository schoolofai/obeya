package realtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{1, 1 * time.Second, 1 * time.Second},
		{2, 2 * time.Second, 2 * time.Second},
		{3, 4 * time.Second, 4 * time.Second},
		{4, 8 * time.Second, 8 * time.Second},
		{10, 30 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		delay := backoffDelay(tt.attempt)
		if delay < tt.wantMin || delay > tt.wantMax {
			t.Errorf("backoffDelay(%d) = %v, want between %v and %v",
				tt.attempt, delay, tt.wantMin, tt.wantMax)
		}
	}
}

func TestBuildURL(t *testing.T) {
	c := NewClient(SubscriptionConfig{
		AppwriteEndpoint: "https://cloud.appwrite.io/v1",
		ProjectID:        "proj-123",
		DatabaseID:       "obeya",
		BoardID:          "board-abc",
	})

	url := c.buildURL()

	if !strings.HasPrefix(url, "wss://cloud.appwrite.io/v1/realtime?") {
		t.Errorf("unexpected URL prefix: %s", url)
	}
	if !strings.Contains(url, "project=proj-123") {
		t.Errorf("URL missing project param: %s", url)
	}
	if !strings.Contains(url, "channels") {
		t.Errorf("URL missing channels param: %s", url)
	}
	if !strings.Contains(url, "databases.obeya.collections.items.documents") {
		t.Errorf("URL missing items channel: %s", url)
	}
	if !strings.Contains(url, "databases.obeya.collections.item_history.documents") {
		t.Errorf("URL missing item_history channel: %s", url)
	}
}

func TestClientReceivesEvents(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		event := map[string]interface{}{
			"data": map[string]interface{}{
				"events": []interface{}{
					"databases.obeya.collections.items.documents.item-1.create",
				},
				"payload": map[string]interface{}{
					"$id":         "item-1",
					"board_id":    "board-test",
					"display_num": float64(1),
					"type":        "task",
					"title":       "Test task",
					"status":      "todo",
				},
			},
		}

		msg, _ := json.Marshal(event)
		conn.WriteMessage(websocket.TextMessage, msg)

		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(SubscriptionConfig{
		AppwriteEndpoint: wsURL,
		ProjectID:        "test",
		DatabaseID:       "obeya",
		BoardID:          "board-test",
	})

	go func() {
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Errorf("dial failed: %v", err)
			return
		}
		defer conn.Close()

		client.mu.Lock()
		client.conn = conn
		client.mu.Unlock()

		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				return
			}
			client.handleMessage(msgBytes)
		}
	}()

	select {
	case event := <-client.Events():
		if event.Action != ActionCreate {
			t.Errorf("expected create action, got %s", event.Action)
		}
		if event.DocumentID != "item-1" {
			t.Errorf("expected document ID item-1, got %s", event.DocumentID)
		}
		if event.CollectionID != "items" {
			t.Errorf("expected collection items, got %s", event.CollectionID)
		}
		title, _ := event.Payload["title"].(string)
		if title != "Test task" {
			t.Errorf("expected title 'Test task', got %s", title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	client.Close()
}

func TestClientFiltersBoardID(t *testing.T) {
	client := NewClient(SubscriptionConfig{
		BoardID: "my-board",
	})

	msg := map[string]interface{}{
		"events": []interface{}{
			"databases.obeya.collections.items.documents.item-99.create",
		},
		"payload": map[string]interface{}{
			"$id":      "item-99",
			"board_id": "other-board",
			"title":    "Wrong board",
		},
	}

	wrapped := map[string]interface{}{
		"data": msg,
	}
	msgBytes, _ := json.Marshal(wrapped)
	client.handleMessage(msgBytes)

	select {
	case ev := <-client.Events():
		t.Fatalf("should not receive event for wrong board, got: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	client.Close()
}

func TestClientClose(t *testing.T) {
	client := NewClient(SubscriptionConfig{})
	client.Close()

	select {
	case <-client.done:
		// Expected
	default:
		t.Fatal("done channel should be closed after Close()")
	}

	// Double close should not panic
	client.Close()
}
