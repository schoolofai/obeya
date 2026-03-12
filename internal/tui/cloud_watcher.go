package tui

import (
	"sync"

	"github.com/niladribose/obeya/internal/realtime"
)

// cloudBoardWatcher bridges the realtime.Client to the boardWatcher interface.
type cloudBoardWatcher struct {
	client    *realtime.Client
	eventCh   chan struct{}
	errCh     chan error
	done      chan struct{}
	closeOnce sync.Once
}

func newCloudBoardWatcher(config realtime.SubscriptionConfig) *cloudBoardWatcher {
	client := realtime.NewClient(config)

	cw := &cloudBoardWatcher{
		client:  client,
		eventCh: make(chan struct{}, 1),
		errCh:   make(chan error, 8),
		done:    make(chan struct{}),
	}

	go client.Connect()
	go cw.relay()

	return cw
}

func (cw *cloudBoardWatcher) relay() {
	defer func() {
		close(cw.eventCh)
		close(cw.errCh)
	}()

	for {
		select {
		case <-cw.done:
			return
		case _, ok := <-cw.client.Events():
			if !ok {
				return
			}
			select {
			case cw.eventCh <- struct{}{}:
			default:
			}
		case err, ok := <-cw.client.Errors():
			if !ok {
				return
			}
			select {
			case cw.errCh <- err:
			default:
			}
		}
	}
}

func (cw *cloudBoardWatcher) events() <-chan struct{} {
	return cw.eventCh
}

func (cw *cloudBoardWatcher) errors() <-chan error {
	return cw.errCh
}

func (cw *cloudBoardWatcher) close() {
	cw.closeOnce.Do(func() {
		close(cw.done)
		cw.client.Close()
	})
}
