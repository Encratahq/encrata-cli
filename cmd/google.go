package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var googleCmd = &cobra.Command{
	Use:   "google [query]",
	Short: "Perform a Google search",
	Long:  "Search Google for web results, news, images, videos, scholar, places, maps, shopping, patents, or autocomplete.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)

		searchType, _ := cmd.Flags().GetString("type")
		country, _ := cmd.Flags().GetString("country")
		lang, _ := cmd.Flags().GetString("lang")
		num, _ := cmd.Flags().GetInt("num")
		page, _ := cmd.Flags().GetInt("page")

		req := &api.GoogleRequest{
			Query:   args[0],
			Type:    searchType,
			Country: country,
			Lang:    lang,
			Num:     num,
			Page:    page,
		}

		data, err := client.GoogleSearch(cmd.Context(), req)
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Query   string `json:"query"`
			Type    string `json:"type"`
			Organic []struct {
				Title   string `json:"title"`
				Link    string `json:"link"`
				Snippet string `json:"snippet"`
			} `json:"organic"`
			AnswerBox *struct {
				Title  string `json:"title"`
				Answer string `json:"answer"`
			} `json:"answer_box"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header(fmt.Sprintf("Google %s: %s", result.Type, args[0]))

		if result.AnswerBox != nil && result.AnswerBox.Answer != "" {
			output.Bold.Printf("  Answer: %s\n", result.AnswerBox.Answer)
			fmt.Println()
		}

		if len(result.Organic) > 0 {
			for i, r := range result.Organic {
				output.Bold.Printf("  %d. %s\n", i+1, r.Title)
				output.Dim.Printf("     %s\n", r.Link)
				if r.Snippet != "" {
					fmt.Printf("     %s\n", truncate(r.Snippet, 120))
				}
				fmt.Println()
			}
		}

		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}

func init() {
	googleCmd.Flags().String("type", "search", "Search type: search, news, images, videos, scholar, places, maps, shopping, patents, autocomplete")
	googleCmd.Flags().String("country", "", "Country code (e.g. us, in)")
	googleCmd.Flags().String("lang", "", "Language code (e.g. en)")
	googleCmd.Flags().Int("num", 10, "Number of results")
	googleCmd.Flags().Int("page", 1, "Page number")
}
