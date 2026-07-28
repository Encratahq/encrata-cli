package cmd

import "github.com/Encratahq/cli/internal/output"

// startSpinner starts a themed spinner for a network/loading operation. The
// spinner writes to STDERR, so it is safe to run in --json mode without
// corrupting the JSON payload on STDOUT.
func startSpinner(message string) *output.Spinner {
	spinner := output.NewSpinner(message)
	spinner.Start()
	return spinner
}

func stopSpinner(spinner *output.Spinner) {
	if spinner != nil {
		spinner.Stop()
	}
}
