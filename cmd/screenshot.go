package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot [url]",
	Short: "Capture a screenshot of a web page",
	Long:  "Capture a full-page or element screenshot of a web page and save it to a file (PNG or JPEG).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		req := &api.ScreenshotRequest{URL: args[0]}
		if format, _ := cmd.Flags().GetString("format"); format != "" {
			req.Format = format
		}
		if cmd.Flags().Changed("viewport") {
			v := false
			req.FullPage = &v
		}
		if selector, _ := cmd.Flags().GetString("selector"); selector != "" {
			req.Selector = selector
		}
		if cmd.Flags().Changed("no-js") {
			v := false
			req.RenderJS = &v
		}
		if cmd.Flags().Changed("block-ads") {
			v, _ := cmd.Flags().GetBool("block-ads")
			req.BlockAds = &v
		}
		if cmd.Flags().Changed("block-trackers") {
			v, _ := cmd.Flags().GetBool("block-trackers")
			req.BlockTrackers = &v
		}
		if waitFor, _ := cmd.Flags().GetString("wait-for"); waitFor != "" {
			req.WaitFor = waitFor
		}
		if headers, _ := cmd.Flags().GetStringArray("header"); len(headers) > 0 {
			parsed, err := parseHeaderFlags(headers)
			if err != nil {
				return err
			}
			req.Headers = parsed
		}
		if timeout, _ := cmd.Flags().GetInt("timeout"); cmd.Flags().Changed("timeout") {
			if err := validation.Timeout(timeout); err != nil {
				return err
			}
			if timeout > 0 {
				req.Timeout = timeout
			}
		}

		spinner := startSpinner("Capturing screenshot...")
		data, err := client.Screenshot(cmd.Context(), req)
		stopSpinner(spinner)
		if err != nil {
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Success    bool    `json:"success"`
			URL        string  `json:"url"`
			StatusCode int     `json:"status_code"`
			Screenshot string  `json:"screenshot"`
			Format     string  `json:"format"`
			Error      string  `json:"error"`
			ErrorCode  string  `json:"error_code"`
			Credits    float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		if !result.Success {
			msg := result.Error
			if msg == "" {
				msg = result.ErrorCode
			}
			output.Error("screenshot failed: " + msg)
			return fmt.Errorf("screenshot failed")
		}

		img, err := base64.StdEncoding.DecodeString(result.Screenshot)
		if err != nil {
			output.Error("failed to decode image: " + err.Error())
			return err
		}

		outFile, _ := cmd.Flags().GetString("output-file")
		if outFile == "" {
			ext := result.Format
			if ext == "" {
				ext = "png"
			}
			outFile = "screenshot." + ext
		}

		displayPath := outFile
		if abs, err := filepath.Abs(outFile); err == nil {
			displayPath = abs
		}
		if err := os.WriteFile(outFile, img, 0o644); err != nil {
			output.Error(err.Error())
			return err
		}

		output.Header("Screenshot: " + result.URL)
		output.KV("Status", fmt.Sprintf("%d", result.StatusCode), "Format", result.Format, "Saved to", displayPath, "Size", fmt.Sprintf("%d bytes", len(img)))
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().String("format", "", "Image format: png (default) or jpeg")
	screenshotCmd.Flags().Bool("viewport", false, "Capture only the viewport instead of the full page")
	screenshotCmd.Flags().String("selector", "", "CSS selector to capture only a specific element")
	screenshotCmd.Flags().Bool("no-js", false, "Disable JavaScript rendering")
	screenshotCmd.Flags().Bool("block-ads", false, "Block ads while capturing")
	screenshotCmd.Flags().Bool("block-trackers", false, "Block trackers while capturing")
	screenshotCmd.Flags().String("wait-for", "", "CSS selector to wait for before capturing")
	screenshotCmd.Flags().StringArray("header", nil, "Custom request header as name=value (repeatable)")
	screenshotCmd.Flags().Int("timeout", 0, "Timeout in milliseconds (max 60000)")
	screenshotCmd.Flags().StringP("output-file", "o", "", "Output file path (default screenshot.<format>)")
}

func parseHeaderFlags(values []string) (map[string]string, error) {
	headers := make(map[string]string, len(values))
	for _, value := range values {
		name, headerValue, ok := strings.Cut(value, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid header %q, expected name=value", value)
		}
		headers[name] = strings.TrimSpace(headerValue)
	}
	return headers, nil
}
