package tui

import (
	"testing"
	"time"

	"github.com/niladribose/obeya/internal/realtime"
)

func TestCloudWatcherImplementsBoardWatcher(t *testing.T) {
	var _ boardWatcher = (*cloudBoardWatcher)(nil)
}

func TestCloudWatcherRelaysEvents(t *testing.T) {
	config := realtime.SubscriptionConfig{
		AppwriteEndpoint: "http://localhost:9999/v1",
		ProjectID:        "test",
		DatabaseID:       "obeya",
		BoardID:          "board-test",
	}

	client := realtime.NewClient(config)

	cw := &cloudBoardWatcher{
		client:  client,
		eventCh: make(chan struct{}, 1),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}

	go cw.relay()
	defer cw.close()

	cw.close()

	select {
	case <-cw.done:
		// Expected
	case <-time.After(time.Second):
		t.Fatal("done channel not closed after close()")
	}
}

func TestLocalWatcherImplementsBoardWatcher(t *testing.T) {
	var _ boardWatcher = (*localBoardWatcher)(nil)
}
