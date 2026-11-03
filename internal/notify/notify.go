// Package notify shows Encrata-branded desktop notifications. Whether and when
// to notify is the caller's decision (see the CLI's local notification
// preferences); this package only renders and delivers one message.
package notify

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
)

// AppName is the sender name the OS shows on every notification.
const AppName = "Encrata"

//go:embed encrata.png
var icon []byte

// Message is one desktop notification. URL, when set, is opened by the
// notification's "View results" button (Windows) or shown in the body elsewhere.
type Message struct {
	Title string
	Body  string
	URL   string
}

// Send delivers m as a desktop notification.
func Send(m Message) error {
	return send(m, iconPath())
}

// iconPath writes the embedded Encrata icon to the user's cache dir (the OS
// notifiers need a file path) and returns it, or "" if it cannot be written.
func iconPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(dir, "encrata", "encrata.png")
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, icon) {
		return path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return ""
	}
	if err := os.WriteFile(path, icon, 0o600); err != nil {
		return ""
	}
	return path
}
