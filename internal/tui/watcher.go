package tui

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// boardFileChangedMsg signals that the board file was modified on disk.
type boardFileChangedMsg struct{}

// watcherStartedMsg carries the initialized watcher (or nil on failure).
type watcherStartedMsg struct {
	watcher boardWatcher
	err     error
}

const debounceInterval = 100 * time.Millisecond

// boardWatcher is the common interface for local (fsnotify) and cloud (WebSocket) watchers.
type boardWatcher interface {
	events() <-chan struct{}
	errors() <-chan error
	close()
}

// localBoardWatcher watches a directory for changes to a specific file.
// It watches the directory (not the file) because writeBoard() uses
// atomic rename (tmp -> board.json), which replaces the inode.
type localBoardWatcher struct {
	watcher   *fsnotify.Watcher
	eventCh   chan struct{}
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once
	fileName  string // just the base name, e.g. "board.json"
}

func newLocalBoardWatcher(boardFilePath string) (*localBoardWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(boardFilePath)
	if err := w.Add(dir); err != nil {
		w.Close()
		return nil, err
	}

	bw := &localBoardWatcher{
		watcher:  w,
		eventCh:  make(chan struct{}, 1),
		errCh:    make(chan error, 1),
		done:     make(chan struct{}),
		fileName: filepath.Base(boardFilePath),
	}

	go bw.loop()
	return bw, nil
}

func (bw *localBoardWatcher) loop() {
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
		close(bw.eventCh)
		close(bw.errCh)
	}()

	for {
		select {
		case event, ok := <-bw.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != bw.fileName {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceInterval, func() {
					select {
					case bw.eventCh <- struct{}{}:
					default:
						// channel full, notification already pending
					}
				})
			}
		case err, ok := <-bw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case bw.errCh <- err:
			default:
			}
		case <-bw.done:
			return
		}
	}
}

func (bw *localBoardWatcher) events() <-chan struct{} {
	return bw.eventCh
}

func (bw *localBoardWatcher) errors() <-chan error {
	return bw.errCh
}

func (bw *localBoardWatcher) close() {
	bw.closeOnce.Do(func() {
		close(bw.done)
		bw.watcher.Close()
	})
}
