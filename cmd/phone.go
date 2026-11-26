package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Encratahq/cli/internal/api"
	"github.com/Encratahq/cli/internal/output"
	"github.com/spf13/cobra"
)

var phoneCmd = &cobra.Command{
	Use:   "phone",
	Short: "Phone number intelligence: validity, risk, breaches, owner identity, VoIP",
	Long: `Look up one phone number.

Numbers must include the country code. Spaces, dashes, dots and parentheses
are fine, and a leading 00 is read as +: "+1 (415) 555-2671" and
"0014155552671" both become +14155552671.

Each lookup costs 1 credit. validity, risk, breaches and identity are free when
repeated for the same number within the billing window.

Exit codes: by default a lookup exits 0 whatever it finds. Pass
--fail-on-finding on risk, breaches or voip to exit 2 when the number is
suspicious or high risk, breached, or a VoIP line (3 = auth, 4 = credits).

Examples:
  encrata phone search +14155552671
  encrata phone validity "+44 20 7946 0958"
  encrata phone risk +14155552671 --fail-on-finding
  encrata phone breaches +14155552671
  encrata phone identity +14155552671
  encrata phone voip +14155552671 --current`,
}

var phoneSearchCmd = &cobra.Command{
	Use:   "search [phone]",
	Short: "Full phone report: format, country, carrier, line type, validation, risk, breaches (1 credit)",
	Long: `Run the full phone report for one number: international and local format,
country and location, carrier and line type, SMS gateway, validation (line
status, VoIP, minimum age), registration, risk and breach exposure.

--include limits the report to some sections: format, country, carrier,
line_type, messaging, validation, registration, risk, breaches.`,
	Example: `  encrata phone search +14155552671
  encrata phone search +14155552671 --include carrier,risk
  encrata phone search +14155552671 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		include, _ := cmd.Flags().GetStringSlice("include")
		for i := range include {
			include[i] = strings.ToLower(strings.TrimSpace(include[i]))
		}
		return runPhoneLookup(cmd, args[0], "Phone Report", "Looking up number...",
			func(c api.API, ctx context.Context, phone string) (json.RawMessage, error) {
				return c.PhoneSearch(ctx, phone, include)
			}, renderPhoneSearch, nil)
	},
}

var phoneValidityCmd = &cobra.Command{
	Use:   "validity [phone]",
	Short: "Is the number real, active and dialable: valid / invalid / risky (1 credit)",
	Long: `Check whether a number is real, active and dialable. The status is valid,
invalid, risky (a VoIP or disposable number) or unknown, with the line type,
carrier, country and the signals behind it.`,
	Example: `  encrata phone validity +14155552671
  encrata phone validity "+44 20 7946 0958" --json`,
	Args: cobra.ExactArgs(1),
	RunE: phoneFeatureRunE(api.PhoneFeatureValidity, "Phone Validity", "Checking validity...", renderPhoneValidity, nil),
}

var phoneRiskCmd = &cobra.Command{
	Use:   "risk [phone]",
	Short: "Fraud and abuse risk: safe / suspicious / high_risk (1 credit)",
	Long: `Assess fraud and abuse risk for a number: a verdict of safe, suspicious or
high_risk, the provider's risk level, abuse reports, disposable and VoIP flags,
breach exposure, and the reasons behind the verdict.`,
	Example: `  encrata phone risk +14155552671
  encrata phone risk +14155552671 --fail-on-finding`,
	Args: cobra.ExactArgs(1),
	RunE: phoneFeatureRunE(api.PhoneFeatureRisk, "Phone Risk", "Assessing risk...", renderPhoneRisk, phoneRiskIsFinding),
}

var phoneBreachesCmd = &cobra.Command{
	Use:   "breaches [phone]",
	Short: "Has the number been exposed in known data breaches (1 credit)",
	Example: `  encrata phone breaches +14155552671
  encrata phone breaches +14155552671 --fail-on-finding`,
	Args: cobra.ExactArgs(1),
	RunE: phoneFeatureRunE(api.PhoneFeatureBreaches, "Phone Breaches", "Checking breaches...", renderPhoneBreaches, phoneBreachedIsFinding),
}

var phoneIdentityCmd = &cobra.Command{
	Use:   "identity [phone]",
	Short: "Reverse phone lookup: owner name, emails, socials, locations (1 credit)",
	Long: `Reverse phone lookup: who the number belongs to, from public sources. Shows
the owner name, other names, emails, usernames, social profiles, locations,
addresses, relatives and the sources they were found in.`,
	Example: `  encrata phone identity +14155552671`,
	Args: cobra.ExactArgs(1),
	RunE: phoneFeatureRunE(api.PhoneFeatureIdentity, "Phone Identity", "Looking up owner...", renderPhoneIdentity, nil),
}

var phoneVoipCmd = &cobra.Command{
	Use:   "voip [phone]",
	Short: "Is the number a VoIP line, with carrier and portability (1 credit)",
	Long: `Detect whether a number is a VoIP line. --current also resolves number
portability, so the carrier and line type are today's rather than the
original allocation.`,
	Example: `  encrata phone voip +14155552671
  encrata phone voip +14155552671 --current --fail-on-finding`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := "static"
		if cur, _ := cmd.Flags().GetBool("current"); cur {
			mode = "current"
		}
		return runPhoneLookup(cmd, args[0], "Phone VoIP", "Checking line...",
			func(c api.API, ctx context.Context, phone string) (json.RawMessage, error) {
				return c.PhoneVoip(ctx, phone, mode)
			}, renderPhoneVoip, phoneVoipIsFinding)
	},
}

func init() {
	phoneSearchCmd.Flags().StringSlice("include", nil, "Only these sections: format,country,carrier,line_type,messaging,validation,registration,risk,breaches")
	phoneSearchCmd.Flags().String("out", "", "Write the full JSON result to a file")

	for _, c := range []*cobra.Command{phoneValidityCmd, phoneRiskCmd, phoneBreachesCmd, phoneIdentityCmd} {
		c.Flags().String("out", "", "Write the full JSON result to a file")
	}
	for _, c := range []*cobra.Command{phoneRiskCmd, phoneBreachesCmd, phoneVoipCmd} {
		c.Flags().Bool("fail-on-finding", false, "Exit with code 2 when the lookup flags the number")
	}
	phoneVoipCmd.Flags().Bool("current", false, "Resolve number portability (today's carrier and line type)")
	phoneVoipCmd.Flags().String("out", "", "Write the full JSON result to a file")

	phoneCmd.AddCommand(phoneSearchCmd, phoneValidityCmd, phoneRiskCmd, phoneBreachesCmd, phoneIdentityCmd, phoneVoipCmd)
}

// phoneFeatureRunE builds the RunE of a {phone}-body sub-lookup.
func phoneFeatureRunE(feature, title, spinnerMsg string, render func(map[string]interface{}), isFinding func(map[string]interface{}) bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return runPhoneLookup(cmd, args[0], title, spinnerMsg,
			func(c api.API, ctx context.Context, phone string) (json.RawMessage, error) {
				return c.PhoneLookup(ctx, feature, phone)
			}, render, isFinding)
	}
}

// runPhoneLookup normalizes the number, runs one lookup, renders it, and applies
// --out and --fail-on-finding.
func runPhoneLookup(
	cmd *cobra.Command,
	raw, title, spinnerMsg string,
	call func(api.API, context.Context, string) (json.RawMessage, error),
	render func(map[string]interface{}),
	isFinding func(map[string]interface{}) bool,
) error {
	phone, err := api.NormalizePhone(raw)
	if err != nil {
		return friendlyFormatError(cmd, err.Error())
	}
	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner(spinnerMsg)
	data, err := call(client, cmd.Context(), phone)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	var r map[string]interface{}
	if jsonMode() {
		output.JSON(data)
		_ = json.Unmarshal(data, &r)
	} else {
		if !decode(data, &r) {
			return nil
		}
		output.Header(title + ": " + phone)
		render(r)
		printCredits(r)
	}

	if out, _ := cmd.Flags().GetString("out"); out != "" {
		if err := saveResult(out, data); err != nil {
			return err
		}
	}
	if isFinding != nil && r != nil && isFinding(r) && failOnFinding(cmd) {
		return errBreachDetected
	}
	return nil
}

// ── findings (--fail-on-finding) ────────────────────────────────────────────

func phoneRiskIsFinding(r map[string]interface{}) bool {
	switch strings.ToLower(field(r, "verdict")) {
	case "suspicious", "high_risk":
		return true
	}
	return false
}

func phoneBreachedIsFinding(r map[string]interface{}) bool {
	return boolField(r, "breached") == "true"
}

func phoneVoipIsFinding(r map[string]interface{}) bool {
	return boolField(r, "is_voip") == "true"
}

// ── renderers ───────────────────────────────────────────────────────────────

// phoneVerdictColor colors the phone verdicts the shared palette lacks.
func phoneVerdictColor(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "suspicious":
		return output.Warn.Sprint(v)
	case "high_risk", "high-risk":
		return output.Err.Sprint(v)
	}
	return statusColor(v)
}

func renderPhoneSearch(r map[string]interface{}) {
	country := field(r, "country.name")
	if code := field(r, "country.code"); code != "" && country != "" {
		country += " (" + code + ")"
	}
	valid := "invalid"
	if boolField(r, "valid") == "true" {
		valid = "valid"
	}
	printNonEmptyKV(
		"Phone", field(r, "phone"),
		"Valid", statusColor(valid),
		"International", field(r, "format.international"),
		"Local", field(r, "format.local"),
		"Type", field(r, "type"),
		"Carrier", field(r, "carrier.name"),
		"Line type", field(r, "carrier.line_type"),
		"Country", country,
		"Dial prefix", field(r, "country.prefix"),
		"Location", firstNonEmpty(field(r, "location"), joinNonEmpty(", ", field(r, "country.city"), field(r, "country.region"))),
		"Timezone", field(r, "country.timezone"),
	)
	fmt.Println()
	minAge := field(r, "validation.minimum_age")
	if minAge == "0" {
		minAge = ""
	}
	section("Validation",
		"Line status", field(r, "validation.line_status"),
		"VoIP", yesNo(boolField(r, "validation.is_voip")),
		"Minimum age", minAge,
	)
	section("Registration",
		"Name", field(r, "registration.name"),
		"Type", field(r, "registration.type"),
	)
	section("Risk",
		"Risk level", field(r, "risk.risk_level"),
		"Disposable", yesNo(boolField(r, "risk.is_disposable")),
		"Abuse detected", yesNo(boolField(r, "risk.is_abuse_detected")),
	)
	if _, ok := r["breaches"]; ok {
		section("Breaches",
			"Total", firstNonEmpty(field(r, "breaches.total_breaches"), "0"),
			"First breached", field(r, "breaches.date_first_breached"),
			"Last breached", field(r, "breaches.date_last_breached"),
		)
	}
	section("Messaging",
		"SMS domain", field(r, "messaging.sms_domain"),
		"SMS email", field(r, "messaging.sms_email"),
	)
}

func renderPhoneValidity(r map[string]interface{}) {
	printNonEmptyKV(
		"Phone", firstNonEmpty(field(r, "format"), field(r, "phone")),
		"Status", statusColor(field(r, "status")),
		"Reason", field(r, "reason"),
		"Line type", field(r, "line_type"),
		"Line status", field(r, "line_status"),
		"Carrier", field(r, "carrier"),
		"Country", field(r, "country"),
		"Location", joinNonEmpty(", ", field(r, "city"), field(r, "region")),
		"Timezone", field(r, "timezone"),
		"VoIP", yesNo(boolField(r, "is_voip")),
		"SMS-capable", yesNo(boolField(r, "sms_capable")),
		"Signals", listField(r, "signals"),
	)
	fmt.Println()
}

func renderPhoneRisk(r map[string]interface{}) {
	breaches := firstNonEmpty(field(r, "total_breaches"), "0")
	printNonEmptyKV(
		"Phone", field(r, "phone"),
		"Verdict", phoneVerdictColor(field(r, "verdict")),
		"Risk level", field(r, "risk_level"),
		"Abuse reported", yesNo(boolField(r, "is_abuse_detected")),
		"Disposable", yesNo(boolField(r, "is_disposable")),
		"VoIP", yesNo(boolField(r, "is_voip")),
		"Breaches", breaches,
		"First breached", field(r, "first_breached"),
		"Last breached", field(r, "last_breached"),
	)
	fmt.Println()
	if reasons := firstArr(r, "reasons"); len(reasons) > 0 {
		output.Bold.Println("  Reasons:")
		for _, x := range reasons {
			fmt.Println("    - " + fmt.Sprint(x))
		}
		fmt.Println()
	}
}

func renderPhoneBreaches(r map[string]interface{}) {
	breached := "no"
	if boolField(r, "breached") == "true" {
		breached = output.Err.Sprint("yes")
	}
	printNonEmptyKV(
		"Phone", field(r, "phone"),
		"Breached", breached,
		"Breaches", firstNonEmpty(field(r, "total_breaches"), "0"),
		"First breached", field(r, "first_breached"),
		"Last breached", field(r, "last_breached"),
		"Summary", field(r, "summary"),
	)
	fmt.Println()
}

func renderPhoneIdentity(r map[string]interface{}) {
	found := "no"
	if boolField(r, "found") == "true" {
		found = output.Success.Sprint("yes")
	}
	printNonEmptyKV(
		"Phone", field(r, "phone"),
		"Owner found", found,
		"Owner", field(r, "owner_name"),
		"Carrier", field(r, "carrier"),
		"Line type", field(r, "line_type"),
		"Country", field(r, "country"),
		"Location", joinNonEmpty(", ", field(r, "city"), field(r, "region")),
	)
	fmt.Println()
	section("Found",
		"Names", listField(r, "names"),
		"Emails", listField(r, "emails"),
		"Usernames", listField(r, "usernames"),
		"Locations", listField(r, "locations"),
		"Addresses", listField(r, "addresses"),
		"Relatives", listField(r, "relatives"),
	)
	if socials := firstArr(r, "socials"); len(socials) > 0 {
		rows := make([][]string, 0, len(socials))
		for _, s := range socials {
			if m, ok := s.(map[string]interface{}); ok {
				rows = append(rows, []string{firstNonEmpty(getStr(m, "platform"), "—"), getStr(m, "url")})
			}
		}
		output.Bold.Println("  Social profiles:")
		output.Table([]string{"Platform", "URL"}, rows)
		fmt.Println()
	}
	if sources := firstArr(r, "sources"); len(sources) > 0 {
		rows := make([][]string, 0, len(sources))
		for _, s := range sources {
			if m, ok := s.(map[string]interface{}); ok {
				rows = append(rows, []string{firstNonEmpty(getStr(m, "site"), "—"), truncate(getStr(m, "title"), 60), getStr(m, "link")})
			}
		}
		output.Bold.Println("  Sources:")
		output.Table([]string{"Site", "Title", "Link"}, rows)
		fmt.Println()
	}
}

func renderPhoneVoip(r map[string]interface{}) {
	voip := "no"
	if boolField(r, "is_voip") == "true" {
		voip = output.Warn.Sprint("yes")
	}
	ported := ""
	if b := boolField(r, "ported"); b != "" {
		ported = yesNo(b)
	}
	country := field(r, "country")
	if code := field(r, "country_code"); code != "" && country != "" {
		country += " (" + code + ")"
	}
	printNonEmptyKV(
		"Phone", field(r, "phone"),
		"VoIP", voip,
		"VoIP status", field(r, "voip_status"),
		"Valid", yesNo(boolField(r, "valid")),
		"Reason", field(r, "reason"),
		"Line type", field(r, "line_type"),
		"Original line type", field(r, "original_line_type"),
		"Carrier", field(r, "carrier"),
		"Current carrier", field(r, "current_carrier"),
		"Ported", ported,
		"Country", country,
		"Region", field(r, "region"),
		"Timezone", listField(r, "timezone"),
		"Mode", field(r, "mode"),
	)
	fmt.Println()
}
