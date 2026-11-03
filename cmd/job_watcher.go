package cmd

import (
	"context"
	"sync"

	"github.com/Encratahq/cli/internal/api"
)

// jobWatcher holds the completion stream open in the background while a command
// waits on its own job. It is started BEFORE the job is created: the server only
// pushes events that happen while a client is connected, so subscribing after
// create could miss a fast job's completion. Best effort: if the stream is
// unavailable the caller's status poll still finishes the wait.
type jobWatcher struct {
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	seen    map[string]api.JobEvent
	waiters map[string]chan api.JobEvent
}

// startJobWatcher subscribes in the background until stop is called or ctx ends.
func startJobWatcher(ctx context.Context, client api.SubscribeAPI) *jobWatcher {
	ctx, cancel := context.WithCancel(ctx)
	w := &jobWatcher{
		cancel:  cancel,
		done:    make(chan struct{}),
		seen:    make(map[string]api.JobEvent),
		waiters: make(map[string]chan api.JobEvent),
	}
	go func() {
		defer close(w.done)
		_ = client.Subscribe(ctx, api.SubscribeHooks{}, func(ev api.JobEvent) error {
			w.deliver(ev)
			return nil
		})
	}()
	return w
}

func (w *jobWatcher) deliver(ev api.JobEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.seen[ev.JobID]; ok {
		return
	}
	w.seen[ev.JobID] = ev
	if ch, ok := w.waiters[ev.JobID]; ok {
		ch <- ev
	}
}

// finished returns a channel that receives jobID's terminal event once, even if
// the event arrived before the call.
func (w *jobWatcher) finished(jobID string) <-chan api.JobEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan api.JobEvent, 1)
	if ev, ok := w.seen[jobID]; ok {
		ch <- ev
	} else {
		w.waiters[jobID] = ch
	}
	return ch
}

// stop closes the stream and waits for the reader to exit.
func (w *jobWatcher) stop() {
	w.cancel()
	<-w.done
}
