package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var phoneCmd = &cobra.Command{
	Use:   "phone [number]",
	Short: "Look up a phone number",
	Long:  "Retrieve carrier, format, country, messaging, validation, risk, and breach data for a phone number.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return err
		}

		client := api.New(cfg.BaseURL, cfg.APIKey)
		data, err := client.PhoneSearch(cmd.Context(), args[0])
		if err != nil {
			output.Error(err.Error())
			return err
		}

		if cfg.Output == "json" {
			output.JSON(data)
			return nil
		}

		var result struct {
			Phone  string `json:"phone"`
			Valid  bool   `json:"valid"`
			Format *struct {
				International string `json:"international"`
				Local         string `json:"local"`
			} `json:"format"`
			Country *struct {
				Code     string `json:"code"`
				Name     string `json:"name"`
				Prefix   string `json:"prefix"`
				Region   string `json:"region"`
				City     string `json:"city"`
				Timezone string `json:"timezone"`
			} `json:"country"`
			Location string `json:"location"`
			Type     string `json:"type"`
			Carrier  *struct {
				Name     string `json:"name"`
				LineType string `json:"line_type"`
			} `json:"carrier"`
			Messaging *struct {
				SMSDomain string `json:"sms_domain"`
				SMSEmail  string `json:"sms_email"`
			} `json:"messaging"`
			Validation *struct {
				IsValid    bool   `json:"is_valid"`
				LineStatus string `json:"line_status"`
				IsVoip     bool   `json:"is_voip"`
				MinimumAge int    `json:"minimum_age"`
			} `json:"validation"`
			Registration *struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"registration"`
			Risk *struct {
				RiskLevel       string `json:"risk_level"`
				IsDisposable    bool   `json:"is_disposable"`
				IsAbuseDetected bool   `json:"is_abuse_detected"`
			} `json:"risk"`
			Breaches *struct {
				TotalBreaches     int    `json:"total_breaches"`
				DateFirstBreached string `json:"date_first_breached"`
				DateLastBreached  string `json:"date_last_breached"`
			} `json:"breaches"`
			Credits float64 `json:"credits"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			output.JSON(data)
			return nil
		}

		output.Header("Phone Lookup: " + args[0])

		valid := output.Err.Sprint("✗ Invalid")
		if result.Valid {
			valid = output.Success.Sprint("✓ Valid")
		}

		international := ""
		local := ""
		if result.Format != nil {
			international = result.Format.International
			local = result.Format.Local
		}

		countryName := ""
		countryCode := ""
		countryPrefix := ""
		region := ""
		city := ""
		timezone := ""
		if result.Country != nil {
			countryName = result.Country.Name
			countryCode = result.Country.Code
			countryPrefix = result.Country.Prefix
			region = result.Country.Region
			city = result.Country.City
			timezone = result.Country.Timezone
		}

		carrierName := ""
		lineType := ""
		if result.Carrier != nil {
			carrierName = result.Carrier.Name
			lineType = result.Carrier.LineType
		}

		output.KV(
			"Number", result.Phone,
			"Valid", valid,
			"International", international,
			"Local", local,
			"Country", fmt.Sprintf("%s (%s)", countryName, countryCode),
			"Prefix", countryPrefix,
			"Region", region,
			"City", city,
			"Timezone", timezone,
			"Location", result.Location,
			"Type", result.Type,
			"Carrier", carrierName,
			"Line Type", lineType,
		)

		if result.Messaging != nil && (result.Messaging.SMSDomain != "" || result.Messaging.SMSEmail != "") {
			fmt.Println()
			output.SubHeader("Messaging")
			output.KV(
				"SMS Domain", result.Messaging.SMSDomain,
				"SMS Email", result.Messaging.SMSEmail,
			)
		}

		if result.Validation != nil {
			fmt.Println()
			output.SubHeader("Validation")
			voip := output.Success.Sprint("No")
			if result.Validation.IsVoip {
				voip = output.Warn.Sprint("Yes")
			}
			output.KV(
				"Line Status", result.Validation.LineStatus,
				"VoIP", voip,
				"Min Age", fmt.Sprintf("%d years", result.Validation.MinimumAge),
			)
		}

		if result.Registration != nil && (result.Registration.Name != "" || result.Registration.Type != "") {
			fmt.Println()
			output.SubHeader("Registration")
			output.KV(
				"Name", result.Registration.Name,
				"Type", result.Registration.Type,
			)
		}

		if result.Risk != nil {
			fmt.Println()
			output.SubHeader("Risk")
			riskLevel := result.Risk.RiskLevel
			switch riskLevel {
			case "low":
				riskLevel = output.Success.Sprint(riskLevel)
			case "medium":
				riskLevel = output.Warn.Sprint(riskLevel)
			case "high":
				riskLevel = output.Err.Sprint(riskLevel)
			}
			disposable := output.Success.Sprint("No")
			if result.Risk.IsDisposable {
				disposable = output.Err.Sprint("Yes")
			}
			abuse := output.Success.Sprint("No")
			if result.Risk.IsAbuseDetected {
				abuse = output.Err.Sprint("Yes")
			}
			output.KV(
				"Risk Level", riskLevel,
				"Disposable", disposable,
				"Abuse Detected", abuse,
			)
		}

		if result.Breaches != nil && result.Breaches.TotalBreaches > 0 {
			fmt.Println()
			output.SubHeader("Breaches")
			output.KV(
				"Total", fmt.Sprintf("%d", result.Breaches.TotalBreaches),
				"First Breached", result.Breaches.DateFirstBreached,
				"Last Breached", result.Breaches.DateLastBreached,
			)
		}

		fmt.Println()
		output.Dim.Printf("  Credits used: %.0f\n", result.Credits)
		return nil
	},
}
