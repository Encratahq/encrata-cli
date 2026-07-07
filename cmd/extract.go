package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract [url]",
	Short: "Extract clean data from a web page",
	Long:  "Extract clean markdown or structured data from a web page. Use --mode markdown (default) for readable content, or pass --selector name=css to pull specific fields.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		req := &api.ExtractRequest{URL: args[0]}
		if mode, _ := cmd.Flags().GetString("mode"); mode != "" {
			req.Mode = mode
		}
		if selectors, _ := cmd.Flags().GetStringSlice("selector"); len(selectors) > 0 {
			req.Selectors = make(map[string]string)
			for _, s := range selectors {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					req.Selectors[parts[0]] = parts[1]
				}
			}
			if req.Mode == "" {
				req.Mode = "selectors"
			}
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

		spinner := startSpinner("Extracting page...")
		data, err := client.Extract(cmd.Context(), req)
		stopSpinner(spinner)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Success    bool            `json:"success"`
			URL        string          `json:"url"`
			StatusCode int             `json:"status_code"`
			Extracted  json.RawMessage `json:"extracted"`
			Error      string          `json:"error"`
			ErrorCode  string          `json:"error_code"`
			Credits    float64         `json:"credits"`
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
			output.Error("extraction failed: " + msg)
			return fmt.Errorf("extraction failed")
		}

		output.Header("Extract: " + result.URL)
		output.KV("Status", fmt.Sprintf("%d", result.StatusCode))
		fmt.Println()

		// Markdown/text extraction comes back as a JSON string; selectors as an object.
		var asString string
		if json.Unmarshal(result.Extracted, &asString) == nil {
			fmt.Println(asString)
		} else {
			output.JSON(result.Extracted)
		}
		fmt.Println()
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	extractCmd.Flags().String("mode", "", "Extraction mode: markdown (default), text, or selectors")
	extractCmd.Flags().StringSlice("selector", nil, "Field selector as name=css (repeatable)")
	extractCmd.Flags().Bool("no-js", false, "Disable JavaScript rendering")
	extractCmd.Flags().Bool("block-ads", false, "Block ads while extracting")
	extractCmd.Flags().Bool("block-trackers", false, "Block trackers while extracting")
	extractCmd.Flags().String("wait-for", "", "CSS selector to wait for before extracting")
	extractCmd.Flags().StringArray("header", nil, "Custom request header as name=value (repeatable)")
	extractCmd.Flags().Int("timeout", 0, "Timeout in milliseconds (max 60000)")
}
