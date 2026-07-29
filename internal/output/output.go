package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var lastError string

// colorEnabled gates ANSI color. Honors the NO_COLOR convention by default and
// is overridable via SetColor (e.g. the --no-color flag).
var colorEnabled = os.Getenv("NO_COLOR") == ""

// quiet suppresses decorative chrome (headers, banners, info lines, spinner).
var quiet bool

// SetColor enables or disables ANSI color output globally.
func SetColor(enabled bool) { colorEnabled = enabled }

// SetQuiet toggles suppression of decorative output (results still print).
func SetQuiet(q bool) { quiet = q }

// Quiet reports whether decorative output is suppressed.
func Quiet() bool { return quiet }

// colorize wraps s in an ANSI SGR code unless color is disabled.
func colorize(code, s string) string {
	if !colorEnabled {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// brandColor prints text in 256-color terracotta (closest to #cc785c)
func brandColor(s string) string {
	return colorize("38;5;173", s)
}

func brandBold(s string) string {
	return colorize("1;38;5;173", s)
}

func mutedColor(s string) string {
	return colorize("38;5;245", s)
}

func accentColor(s string) string {
	return colorize("38;5;109", s)
}

func JSON(data json.RawMessage) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(buf.String())
}

func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	fmt.Print("  ")
	for i, h := range headers {
		fmt.Fprint(os.Stdout, brandBold(h))
		fmt.Fprint(os.Stdout, strings.Repeat(" ", widths[i]-len(h)+3))
	}
	fmt.Println()
	// Separator
	fmt.Print("  ")
	for i := range headers {
		fmt.Fprintf(os.Stdout, "%s   ", mutedColor(strings.Repeat("─", widths[i])))
	}
	fmt.Println()
	// Rows
	for _, row := range rows {
		fmt.Print("  ")
		for i, cell := range row {
			if i < len(widths) {
				fmt.Fprintf(os.Stdout, "%-*s", widths[i]+3, cell)
			}
		}
		fmt.Println()
	}
}

func KV(pairs ...string) {
	if len(pairs)%2 != 0 {
		return
	}
	maxKey := 0
	for i := 0; i < len(pairs); i += 2 {
		if len(pairs[i]) > maxKey {
			maxKey = len(pairs[i])
		}
	}
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		val := pairs[i+1]
		if val == "" {
			val = mutedColor("—")
		}
		fmt.Printf("  %s%s  %s\n", brandBold(key), strings.Repeat(" ", maxKey-len(key)), val)
	}
}

func Error(msg string) {
	if msg == lastError {
		return
	}
	lastError = msg
	fmt.Fprintf(os.Stderr, "  %s %s\n", colorize("1;31", "✗"), msg)
}

func Info(msg string) {
	if quiet {
		return
	}
	fmt.Printf("  %s %s\n", accentColor("▸"), msg)
}

func Header(title string) {
	if quiet {
		return
	}
	fmt.Println()
	fmt.Printf("  %s\n", brandBold(title))
	fmt.Println()
}

func SubHeader(title string) {
	fmt.Printf("  %s\n", mutedColor("── "+title+" ──"))
}

func SuccessMsg(msg string) {
	fmt.Printf("  %s %s\n", colorize("1;32", "✓"), msg)
}

// SavedPath prints a "Result saved to" confirmation to STDERR so it never
// corrupts JSON written to STDOUT (e.g. when piping `--json` to a file).
func SavedPath(path string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", colorize("1;32", "✓"), "Result saved to: "+path)
}

func Banner() {
	if quiet {
		return
	}
	fmt.Println()
	fmt.Printf("  %s\n", brandBold("encrata"))
	fmt.Printf("  %s\n", mutedColor("intelligence lookups from your terminal"))
	fmt.Println()
}

// Printer is a styled printer with Println, Printf, Sprint, Sprintf methods.
type Printer struct {
	style func(string) string
}

func (p Printer) Println(a ...interface{}) {
	fmt.Println(p.style(fmt.Sprint(a...)))
}

func (p Printer) Printf(format string, a ...interface{}) {
	fmt.Print(p.style(fmt.Sprintf(format, a...)))
}

func (p Printer) Sprint(a ...interface{}) string {
	return p.style(fmt.Sprint(a...))
}

func (p Printer) Sprintf(format string, a ...interface{}) string {
	return p.style(fmt.Sprintf(format, a...))
}

// Package-level styled printers
var (
	Brand   = Printer{style: brandColor}
	Accent  = Printer{style: accentColor}
	Bold    = Printer{style: func(s string) string { return colorize("1", s) }}
	Dim     = Printer{style: mutedColor}
	Warn    = Printer{style: func(s string) string { return colorize("1;33", s) }}
	Success = Printer{style: func(s string) string { return colorize("1;32", s) }}
	Err     = Printer{style: func(s string) string { return colorize("1;31", s) }}
)
