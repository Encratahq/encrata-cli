package output

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
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
	// Off a TTY (piped/redirected/CI) or in --quiet mode, skip the animation
	// entirely so we don't spray carriage returns and ANSI escapes into
	// captured stderr/logs.
	if quiet || !term.IsTerminal(int(os.Stderr.Fd())) {
		close(s.done)
		return
	}
	go func() {
		defer close(s.done)

		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		ticker := time.NewTicker(90 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r  %s %s", brandColor(frames[i%len(frames)]), mutedColor(s.message))
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
