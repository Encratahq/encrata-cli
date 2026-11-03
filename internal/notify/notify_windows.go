//go:build windows

package notify

import "git.sr.ht/~jackmordaunt/go-toast"

// send pushes a Windows toast. go-toast (the backend beeep uses on Windows) is
// called directly because beeep cannot attach action buttons: clicking the toast
// or its "View results" button opens m.URL through protocol activation.
func send(m Message, iconPath string) error {
	n := toast.Notification{
		AppID: AppName,
		Title: m.Title,
		Body:  m.Body,
		Icon:  iconPath,
		Audio: toast.Default,
	}
	if m.URL != "" {
		n.ActivationType = toast.Protocol
		n.ActivationArguments = m.URL
		n.Actions = []toast.Action{{Type: toast.Protocol, Content: "View results", Arguments: m.URL}}
	}
	return n.Push()
}
