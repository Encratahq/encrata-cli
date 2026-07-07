package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/validation"
	"github.com/spf13/cobra"
)

var faceCmd = &cobra.Command{
	Use:   "face [image-url]",
	Short: "Search for matching faces from an image",
	Long:  "Find matching faces and linked identities from a public image URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		req := &api.FaceRequest{ImageURL: args[0]}
		if cmd.Flags().Changed("threshold") {
			t, _ := cmd.Flags().GetFloat64("threshold")
			if err := validation.Threshold(t); err != nil {
				return err
			}
			req.Threshold = &t
		}

		spinner := startSpinner("Searching faces...")
		data, err := client.FaceSearch(cmd.Context(), req)
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
			ImageURL      string  `json:"image_url"`
			Matched       bool    `json:"matched"`
			Threshold     float64 `json:"threshold"`
			FacesDetected int     `json:"faces_detected"`
			Matches       []struct {
				UUID        string  `json:"uuid"`
				Name        string  `json:"name"`
				Probability float64 `json:"probability"`
			} `json:"matches"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Face Search")
		matched := output.Err.Sprint("✗ No match")
		if result.Matched {
			matched = output.Success.Sprint("✓ Matched")
		}
		output.KV(
			"Result", matched,
			"Faces Detected", fmt.Sprintf("%d", result.FacesDetected),
			"Threshold", fmt.Sprintf("%.2f", result.Threshold),
		)
		fmt.Println()

		if len(result.Matches) > 0 {
			output.Bold.Println("  Matches:")
			for i, m := range result.Matches {
				name := m.Name
				if name == "" {
					name = m.UUID
				}
				output.Bold.Printf("  [%d] %s\n", i+1, name)
				output.KV("Confidence", fmt.Sprintf("%.0f%%", m.Probability*100))
			}
			fmt.Println()
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	faceCmd.Flags().Float64("threshold", 0, "Match confidence threshold between 0 and 1")
}
