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
	Headers []string
	Rows    [][]string
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
	lineNumber := 2
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Table{}, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, fmt.Sprintf("read CSV row %d", lineNumber), err)
		}
		row := append([]string(nil), record...)
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		if isBlankRow(row) {
			lineNumber++
			continue
		}
		if len(row) != len(headers) {
			return Table{}, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("CSV row %d has %d columns, want %d", lineNumber, len(row), len(headers)))
		}
		rows = append(rows, row)
		lineNumber++
	}
	return Table{Headers: headers, Rows: rows}, nil
}

func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
