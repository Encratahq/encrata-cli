package cmd

import "github.com/Encratahq/cli/internal/output"

func startSpinner(message string) *output.Spinner {
	if jsonMode() {
		return nil
	}

	spinner := output.NewSpinner(message)
	spinner.Start()
	return spinner
}

func stopSpinner(spinner *output.Spinner) {
	if spinner != nil {
		spinner.Stop()
	}
}
