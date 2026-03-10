package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSendsMessageOnFileChange(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond) // let watcher settle

	// Modify via atomic rename (same as json_store.go writeBoard)
	tmpFile := boardFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(`{"version":1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpFile, boardFile); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		// success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected file change notification, got timeout")
	}
}

func TestWatcherDebounceCoalescesRapidWrites(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	if err := os.WriteFile(boardFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	defer w.close()

	ch := w.events()

	time.Sleep(50 * time.Millisecond)

	// Write 5 times rapidly via atomic rename (same pattern as writeBoard)
	for i := 0; i < 5; i++ {
		tmpFile := boardFile + ".tmp"
		os.WriteFile(tmpFile, []byte(fmt.Sprintf(`{"version":%d}`, i)), 0644)
		os.Rename(tmpFile, boardFile)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire, then drain
	time.Sleep(200 * time.Millisecond)

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 1 {
		t.Fatalf("expected 1 debounced notification, got %d", count)
	}
}

func TestWatcherCloseStopsWatching(t *testing.T) {
	dir := t.TempDir()
	boardFile := filepath.Join(dir, "board.json")
	os.WriteFile(boardFile, []byte(`{}`), 0644)

	w, err := newBoardWatcher(boardFile)
	if err != nil {
		t.Fatal(err)
	}

	ch := w.events()
	w.close()

	// Write after close — should NOT get notification
	time.Sleep(50 * time.Millisecond)
	os.WriteFile(boardFile, []byte(`{"version":99}`), 0644)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("should not receive events after close")
		}
		// channel closed — expected
	case <-time.After(300 * time.Millisecond):
		// success — no event received
	}
}
