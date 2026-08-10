package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// writeFlatCSV writes a header row plus one flattened row per result.
func writeFlatCSV(path string, cols []exportColumn, rows []map[string]interface{}) error {
	data, err := buildFlatCSV(cols, rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// buildFlatCSV returns the flattened CSV payload (header + rows).
func buildFlatCSV(cols []exportColumn, rows []map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.name
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, r := range rows {
		record := make([]string, len(cols))
		for i, c := range cols {
			record[i] = c.resolve(r)
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeRawJSON writes the raw, nested result objects (unflattened).
func writeRawJSON(path string, rows []map[string]interface{}) error {
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// --- Minimal XLSX writer (single sheet, inline strings, no dependencies) ---

const (
	xlsxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
		`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
		`</Types>`

	xlsxRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`

	xlsxWorkbook = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
		`<sheets><sheet name="Results" sheetId="1" r:id="rId1"/></sheets></workbook>`

	xlsxWorkbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
		`</Relationships>`

	xlsxSheetOpen = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`
)

var xlsxEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

// writeXLSX writes a single-sheet .xlsx with a header row and one row per result.
func writeXLSX(path string, cols []exportColumn, rows []map[string]interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	add := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, content)
		return err
	}

	if err := add("[Content_Types].xml", xlsxContentTypes); err != nil {
		return err
	}
	if err := add("_rels/.rels", xlsxRootRels); err != nil {
		return err
	}
	if err := add("xl/workbook.xml", xlsxWorkbook); err != nil {
		return err
	}
	if err := add("xl/_rels/workbook.xml.rels", xlsxWorkbookRels); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(xlsxSheetOpen)
	sb.WriteString("<sheetData>")
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = c.name
	}
	writeXLSXRow(&sb, 1, header)
	for i, r := range rows {
		values := make([]string, len(cols))
		for j, c := range cols {
			values[j] = c.resolve(r)
		}
		writeXLSXRow(&sb, i+2, values)
	}
	sb.WriteString("</sheetData></worksheet>")

	if err := add("xl/worksheets/sheet1.xml", sb.String()); err != nil {
		return err
	}
	return zw.Close()
}

func writeXLSXRow(sb *strings.Builder, rowNum int, values []string) {
	fmt.Fprintf(sb, `<row r="%d">`, rowNum)
	for i, v := range values {
		ref := xlsxColRef(i) + strconv.Itoa(rowNum)
		fmt.Fprintf(sb, `<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, xlsxEscaper.Replace(v))
	}
	sb.WriteString("</row>")
}

// xlsxColRef converts a 0-based column index to a spreadsheet letter (A, B, …, AA).
func xlsxColRef(i int) string {
	ref := ""
	for n := i + 1; n > 0; n /= 26 {
		n--
		ref = string(rune('A'+n%26)) + ref
	}
	return ref
}
