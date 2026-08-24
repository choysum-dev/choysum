// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package importcli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var importModelFilenamePattern = regexp.MustCompile(`^([a-z][a-z0-9]*)[._-]([A-Z][a-zA-Z0-9]*)$`)

// ValidateCSVSourcePath ensures the import source path refers to a local .csv file.
func ValidateCSVSourcePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("import source path is required")
	}
	base := filepath.Base(path)
	if !strings.HasSuffix(strings.ToLower(base), ".csv") {
		return fmt.Errorf("import source must be a .csv file")
	}
	return nil
}

// ModelFromFilename derives app.Model from names such as base.Country.csv, base_Country.csv, or base-Country.csv.
func ModelFromFilename(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("import source path is required")
	}

	base := filepath.Base(path)
	if base == "." || base == ".." || base == string(filepath.Separator) {
		return "", fmt.Errorf("import source path is invalid")
	}
	if err := ValidateCSVSourcePath(path); err != nil {
		return "", err
	}

	stem := strings.TrimSuffix(base, filepath.Ext(base))
	matches := importModelFilenamePattern.FindStringSubmatch(stem)
	if len(matches) != 3 {
		return "", fmt.Errorf(
			"cannot infer model from filename %q; use app_Model.csv, app.Model.csv, or app-Model.csv, or pass --model",
			base,
		)
	}
	return matches[1] + "." + matches[2], nil
}
