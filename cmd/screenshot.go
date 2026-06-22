package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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
		if timeout, _ := cmd.Flags().GetInt("timeout"); timeout > 0 {
			req.Timeout = timeout
		}

		data, err := client.Screenshot(req)
		if err != nil {
			output.Error(err.Error())
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
		if err := os.WriteFile(outFile, img, 0o644); err != nil {
			output.Error(err.Error())
			return err
		}

		output.Header("Screenshot: " + result.URL)
		output.KV("Status", fmt.Sprintf("%d", result.StatusCode), "Format", result.Format, "Saved to", outFile, "Size", fmt.Sprintf("%d bytes", len(img)))
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().String("format", "", "Image format: png (default) or jpeg")
	screenshotCmd.Flags().Bool("viewport", false, "Capture only the viewport instead of the full page")
	screenshotCmd.Flags().String("selector", "", "CSS selector to capture only a specific element")
	screenshotCmd.Flags().Int("timeout", 0, "Timeout in milliseconds (max 60000)")
	screenshotCmd.Flags().StringP("output-file", "o", "", "Output file path (default screenshot.<format>)")
}
