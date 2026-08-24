// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// Table holds parsed CSV headers and rows.
type Table struct {
	Headers    []string
	Rows       [][]string
	RowNumbers []int // 1-based physical source line numbers for each data row
}

// ReadTable parses CSV bytes with comma delimiter and a single header row.
func ReadTable(data []byte) (Table, error) {
	data = StripUTF8BOM(data)
	if err := ValidateUTF8(data); err != nil {
		return Table{}, err
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.ReuseRecord = true
	reader.TrimLeadingSpace = true
	// Allow variable field counts so whitespace-only rows reach isBlankRow
	// before the explicit header-length check (encoding/csv rejects "   \n" otherwise).
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return Table{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "CSV is empty")
		}
		return Table{}, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, "read CSV header", err)
	}
	headers = append([]string(nil), headers...)
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	rows := make([][]string, 0, 8)
	rowNumbers := make([]int, 0, 8)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Table{}, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, csvReadErrorText(err), err)
		}
		line, _ := reader.FieldPos(0)
		row := append([]string(nil), record...)
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		if isBlankRow(row) {
			continue
		}
		if len(row) != len(headers) {
			return Table{}, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("CSV row %d has %d columns, want %d", line, len(row), len(headers)))
		}
		rows = append(rows, row)
		rowNumbers = append(rowNumbers, line)
	}
	return Table{Headers: headers, Rows: rows, RowNumbers: rowNumbers}, nil
}

func csvReadErrorText(err error) string {
	if pe, ok := err.(*csv.ParseError); ok {
		line := pe.StartLine
		if line == 0 {
			line = pe.Line
		}
		if line > 0 {
			return fmt.Sprintf("read CSV row %d", line)
		}
	}
	return "read CSV row"
}

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
