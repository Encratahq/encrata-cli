package output

import (
	"fmt"
	"sync"
	"time"
)

type Spinner struct {
	message string
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)

		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		ticker := time.NewTicker(90 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Print("\r\033[K")
				return
			case <-ticker.C:
				fmt.Printf("\r  %s %s", brandColor(frames[i%len(frames)]), mutedColor(s.message))
				i++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.once.Do(func() {
		close(s.stop)
		<-s.done
	})
}
