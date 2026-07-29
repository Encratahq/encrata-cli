package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Encratahq/cli/internal/output"
	"github.com/Encratahq/cli/internal/password"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// errBreachDetected signals that a check succeeded but found at least one
// breached password. Execute() maps it to a non-zero exit code (1) without
// printing an error banner, so scripts can use `encrata password` as a guard.
var errBreachDetected = errors.New("password breach detected")

var passwordCmd = &cobra.Command{
	Use:   "password [PASSWORD]",
	Short: "Check whether a password has appeared in known data breaches",
	Long: `Check a password against known data breaches (HIBP k-anonymity).

Your password is hashed locally with SHA-1 and only the hash is sent — the
plaintext never leaves your machine and is never logged, cached, or stored.

With no argument the password is read interactively without echo, so it never
lands in your shell history. Use --file or --stdin to check many passwords at
once (one per line, de-duplicated, max 1000).

Exit codes: by default the command exits 0 even when a breach is found. Pass
--fail-on-finding to exit 2 when any password is breached (3 = auth, 4 = credits).

Examples:
  encrata password
  encrata password 'hunter2'
  encrata password --file passwords.txt
  cat passwords.txt | encrata password --stdin`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPassword,
}

func init() {
	passwordCmd.Flags().String("file", "", "Check passwords from a file (one per line)")
	passwordCmd.Flags().Bool("stdin", false, "Read passwords from STDIN (one per line)")
	passwordCmd.Flags().Bool("fail-on-finding", false, "Exit with code 2 if any password is found in a breach")
}

// failOnFinding reports whether the command should exit non-zero (code 2) when
// a breach/finding is detected. Findings are opt-in so a clean scan and a
// broken run are never confused.
func failOnFinding(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("fail-on-finding")
	return v
}

func runPassword(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")
	useStdin, _ := cmd.Flags().GetBool("stdin")

	if file != "" && useStdin {
		return friendlyFormatError(cmd, "choose either --file or --stdin, not both")
	}
	if (file != "" || useStdin) && len(args) > 0 {
		return friendlyFormatError(cmd, "provide a password argument or --file/--stdin, not both")
	}

	if file != "" || useStdin {
		return runPasswordBulk(cmd, file, useStdin)
	}
	return runPasswordSingle(cmd, args)
}

func runPasswordSingle(cmd *cobra.Command, args []string) error {
	var plaintext []byte
	if len(args) == 1 {
		plaintext = []byte(args[0])
	} else {
		pw, err := promptPassword(cmd)
		if err != nil {
			return err
		}
		plaintext = pw
	}
	if len(plaintext) == 0 {
		password.Zero(plaintext)
		return friendlyFormatError(cmd, "password is required")
	}

	// Hash locally and immediately wipe the plaintext buffer.
	hash := password.Hash(plaintext)
	password.Zero(plaintext)

	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner("Checking password...")
	data, err := client.PasswordBreaches(cmd.Context(), hash)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	var result password.SingleResult
	if !decode(data, &result) {
		return nil
	}

	if jsonMode() {
		output.JSON(data)
	} else {
		output.Header("Password Breach Check")
		renderPasswordSingle(result)
	}

	if result.Breach() && failOnFinding(cmd) {
		return errBreachDetected
	}
	return nil
}

func runPasswordBulk(cmd *cobra.Command, file string, useStdin bool) error {
	var raw []byte
	var err error
	if useStdin || file == "" {
		raw, err = io.ReadAll(cmd.InOrStdin())
	} else {
		raw, err = os.ReadFile(file)
	}
	if err != nil {
		return err
	}

	// Hash every line locally, wiping each plaintext buffer, then wipe the
	// raw input so no plaintext lingers in memory.
	lines := password.SplitLines(raw)
	password.Zero(raw)
	hashes := password.PrepareHashes(lines)

	if len(hashes) == 0 {
		return friendlyFormatError(cmd, "no passwords found in input")
	}
	if len(hashes) > password.MaxBulk {
		return friendlyFormatError(cmd, fmt.Sprintf("too many passwords: %d unique (maximum %d per request)", len(hashes), password.MaxBulk))
	}

	client, err := newClient()
	if err != nil {
		return err
	}

	spinner := startSpinner(fmt.Sprintf("Checking %d %s...", len(hashes), plural(len(hashes), "password", "passwords")))
	data, err := client.PasswordBreachesBulk(cmd.Context(), hashes)
	stopSpinner(spinner)
	if err != nil {
		output.Error(err.Error())
		return err
	}

	var result password.BulkResult
	if !decode(data, &result) {
		return nil
	}

	if jsonMode() {
		output.JSON(data)
	} else {
		output.Header(fmt.Sprintf("Password Breach Check: %d %s", result.Total, plural(result.Total, "password", "passwords")))
		renderPasswordBulk(result)
	}

	if result.Breach() && failOnFinding(cmd) {
		return errBreachDetected
	}
	return nil
}

// promptPassword reads a password from the terminal without echoing it.
func promptPassword(cmd *cobra.Command) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, friendlyFormatError(cmd, "no password provided — pass it as an argument, or use --file/--stdin for bulk input")
	}
	fmt.Fprint(cmd.ErrOrStderr(), output.Bold.Sprint("Password: "))
	pw, err := term.ReadPassword(fd)
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	return pw, nil
}

func renderPasswordSingle(r password.SingleResult) {
	if r.Found {
		output.Err.Printf("  ⚠ BREACHED — seen %s %s\n", password.Commas(r.Count), plural(r.Count, "time", "times"))
	} else {
		output.Success.Println("  ✓ Not found in any known breach")
	}
	fmt.Println()
	printNonEmptyKV("Prefix", r.Prefix)
	output.Dim.Printf("  Credits used: %d\n", r.Credits)
}

func renderPasswordBulk(r password.BulkResult) {
	rows := make([][]string, 0, len(r.Results))
	for _, entry := range r.Results {
		found := output.Success.Sprint("no")
		if entry.Found {
			found = output.Err.Sprint("yes")
		}
		rows = append(rows, []string{
			firstNonEmpty(entry.Prefix, "—"),
			found,
			password.Commas(entry.Count),
		})
	}
	output.Table([]string{"Prefix", "Found", "Count"}, rows)
	fmt.Println()

	breached := output.Success.Sprintf("%d", r.Breached)
	if r.Breached > 0 {
		breached = output.Err.Sprintf("%d", r.Breached)
	}
	fmt.Printf("  %s of %d breached · %d %s\n",
		breached, r.Total, r.Credits, plural(r.Credits, "credit", "credits"))
}
