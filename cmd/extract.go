package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
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
		if timeout, _ := cmd.Flags().GetInt("timeout"); timeout > 0 {
			req.Timeout = timeout
		}

		data, err := client.Extract(req)
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
	extractCmd.Flags().Int("timeout", 0, "Timeout in milliseconds (max 60000)")
}
