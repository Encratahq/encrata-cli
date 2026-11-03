//go:build !windows

package notify

import "github.com/gen2brain/beeep"

// send delivers through beeep (Notification Center on macOS, D-Bus/notify-send
// on Linux). Neither supports action buttons through beeep, so the results link
// goes on its own line in the body.
func send(m Message, iconPath string) error {
	beeep.AppName = AppName
	body := m.Body
	if m.URL != "" {
		body += "\n" + m.URL
	}
	return beeep.Notify(m.Title, body, iconPath)
}
