// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package csv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/plan"
	initdataplan "github.com/choysum-dev/choysum/internal/import/plan/initdata"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

const (
	initdataColID          = "id"
	initdataColModel       = "model"
	initdataColNoUpdate    = "noupdate"
	initdataColApplication = "application"
	initdataColModule      = "module"
)

// InitdataRecord is the JSON-records[] equivalent produced from an initdata CSV row.
type InitdataRecord struct {
	Module      string
	Name        string
	Application string
	Model       string
	NoUpdate    *bool
	Values      map[string]any
}

// BuildInitdataPlanFromCSV parses initdata CSV bytes into records (§6.5.1).
// applyingModule fills the module when id omits the module prefix (§6.5.2 / E12).
func BuildInitdataPlanFromCSV(raw []byte, applyingModule string) ([]InitdataRecord, error) {
	table, err := ReadTable(raw)
	if err != nil {
		return nil, err
	}
	colIndex, err := mapInitdataHeaders(table.Headers)
	if err != nil {
		return nil, err
	}
	out := make([]InitdataRecord, 0, len(table.Rows))
	for i, row := range table.Rows {
		rec, err := recordFromCSVRow(table.Headers, row, colIndex, applyingModule)
		if err != nil {
			line := i + 2
			if i < len(table.RowNumbers) {
				line = table.RowNumbers[i]
			}
			return nil, importpkg.ErrorfWrap(importpkg.CodeInvalidFormat, fmt.Sprintf("CSV row %d", line), err)
		}
		out = append(out, rec)
	}
	return out, nil
}

func mapInitdataHeaders(headers []string) (map[string]int, error) {
	colIndex := make(map[string]int, len(headers))
	for i, header := range headers {
		header = strings.TrimSpace(header)
		if header == "" {
			return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "CSV header must not be empty")
		}
		key := strings.ToLower(header)
		if _, dup := colIndex[key]; dup {
			return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("duplicate CSV header %q", header))
		}
		colIndex[key] = i
	}
	if _, ok := colIndex[initdataColID]; !ok {
		return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "initdata CSV requires an id column")
	}
	if _, ok := colIndex[initdataColModel]; !ok {
		return nil, importpkg.Errorf(importpkg.CodeInvalidFormat, "initdata CSV requires a model column")
	}
	return colIndex, nil
}

func recordFromCSVRow(headers []string, row []string, colIndex map[string]int, applyingModule string) (InitdataRecord, error) {
	idCell := strings.TrimSpace(row[colIndex[initdataColID]])
	if idCell == "" {
		return InitdataRecord{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "id is required")
	}
	module, name, err := splitExternalID(idCell, applyingModule)
	if err != nil {
		return InitdataRecord{}, err
	}
	model := strings.TrimSpace(row[colIndex[initdataColModel]])
	if model == "" {
		return InitdataRecord{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "model is required")
	}

	rec := InitdataRecord{
		Module: module,
		Name:   name,
		Model:  model,
		Values: map[string]any{},
	}
	if idx, ok := colIndex[initdataColApplication]; ok {
		rec.Application = strings.TrimSpace(row[idx])
	}
	if idx, ok := colIndex[initdataColModule]; ok {
		explicit := strings.TrimSpace(row[idx])
		if explicit != "" {
			rec.Module = explicit
		}
	}
	if idx, ok := colIndex[initdataColNoUpdate]; ok {
		cell := strings.TrimSpace(row[idx])
		if cell != "" {
			v, err := parseCSVBool(cell)
			if err != nil {
				return InitdataRecord{}, err
			}
			rec.NoUpdate = &v
		}
	}

	reserved := map[string]struct{}{
		initdataColID:          {},
		initdataColModel:       {},
		initdataColNoUpdate:    {},
		initdataColApplication: {},
		initdataColModule:      {},
	}
	for i, header := range headers {
		key := strings.ToLower(strings.TrimSpace(header))
		if _, skip := reserved[key]; skip {
			continue
		}
		cell := strings.TrimSpace(row[i])
		if cell == "" {
			continue
		}
		// Preserve original header casing for field paths (e.g. Code, Name).
		rec.Values[strings.TrimSpace(header)] = cell
	}
	return rec, nil
}

func splitExternalID(id string, applyingModule string) (module string, name string, err error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", importpkg.Errorf(importpkg.CodeInvalidFormat, "id is required")
	}
	if dot := strings.IndexByte(id, '.'); dot >= 0 {
		module = strings.TrimSpace(id[:dot])
		name = strings.TrimSpace(id[dot+1:])
		if module == "" || name == "" {
			return "", "", importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("invalid id %q (want module.name)", id))
		}
		if strings.Contains(name, ".") {
			return "", "", importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("invalid id %q (name must not contain '.')", id))
		}
		return module, name, nil
	}
	module = strings.TrimSpace(applyingModule)
	if module == "" {
		return "", "", importpkg.Errorf(importpkg.CodeInvalidFormat, "id without module requires applying module")
	}
	return module, id, nil
}

func parseCSVBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y":
		return true, nil
	case "0", "false", "no", "n":
		return false, nil
	default:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return false, importpkg.Errorf(importpkg.CodeInvalidFormat, fmt.Sprintf("invalid noupdate %q", raw))
		}
		return v, nil
	}
}

// buildInitdataFilePlan mirrors the JSON initdata adapter: Units are file batches
// for InitdataWriter → Loader (which parses .csv via BuildInitdataPlanFromCSV).
func buildInitdataFilePlan(spec importpkg.Spec) (plan.Plan, error) {
	modulePath := strings.TrimSpace(spec.Source.Path)
	if modulePath == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "source path is required for initdata")
	}
	moduleName := strings.TrimSpace(spec.Module)
	if moduleName == "" {
		return plan.Plan{}, importpkg.Errorf(importpkg.CodeInvalidFormat, "module is required for initdata")
	}

	units := make([]plan.Unit, 0, 2)
	nextIndex := 1
	if files := cleanInitdataPaths(spec.Options.InitdataFiles); len(files) > 0 {
		units = append(units, initdataplan.Unit{
			Index:       nextIndex,
			ModuleName:  moduleName,
			ModulePath:  modulePath,
			Application: strings.TrimSpace(spec.Application),
			Files:       files,
		})
		nextIndex++
	}
	if spec.Options.WithDemo {
		if files := cleanInitdataPaths(spec.Options.DemoFiles); len(files) > 0 {
			units = append(units, initdataplan.Unit{
				Index:       nextIndex,
				ModuleName:  moduleName,
				ModulePath:  modulePath,
				Application: strings.TrimSpace(spec.Application),
				Files:       files,
			})
		}
	}
	return plan.Plan{Units: units}, nil
}

func cleanInitdataPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, rel := range paths {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		out = append(out, rel)
	}
	return out
}
