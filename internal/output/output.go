package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var lastError string

// brandColor prints text in 256-color terracotta (closest to #cc785c)
func brandColor(s string) string {
	return fmt.Sprintf("\033[38;5;173m%s\033[0m", s)
}

func brandBold(s string) string {
	return fmt.Sprintf("\033[1;38;5;173m%s\033[0m", s)
}

func mutedColor(s string) string {
	return fmt.Sprintf("\033[38;5;245m%s\033[0m", s)
}

func accentColor(s string) string {
	return fmt.Sprintf("\033[38;5;109m%s\033[0m", s)
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
	fmt.Fprintf(os.Stderr, "  %s %s\n", "\033[1;31m✗\033[0m", msg)
}

func Info(msg string) {
	fmt.Printf("  %s %s\n", accentColor("▸"), msg)
}

func Header(title string) {
	fmt.Println()
	fmt.Printf("  %s\n", brandBold(title))
	fmt.Println()
}

func SubHeader(title string) {
	fmt.Printf("  %s\n", mutedColor("── "+title+" ──"))
}

func SuccessMsg(msg string) {
	fmt.Printf("  %s %s\n", "\033[1;32m✓\033[0m", msg)
}

func Banner() {
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
	Bold    = Printer{style: func(s string) string { return fmt.Sprintf("\033[1m%s\033[0m", s) }}
	Dim     = Printer{style: mutedColor}
	Warn    = Printer{style: func(s string) string { return fmt.Sprintf("\033[1;33m%s\033[0m", s) }}
	Success = Printer{style: func(s string) string { return fmt.Sprintf("\033[1;32m%s\033[0m", s) }}
	Err     = Printer{style: func(s string) string { return fmt.Sprintf("\033[1;31m%s\033[0m", s) }}
)
