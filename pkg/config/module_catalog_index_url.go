// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"net/url"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
)

// ValidateModuleCatalogIndexURL validates the configured addon catalog index URL.
func ValidateModuleCatalogIndexURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return xfmt.Errorf("module_catalog_index_url is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return xfmt.Errorf("invalid module_catalog_index_url %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return xfmt.Errorf("invalid module_catalog_index_url %q: only http/https are supported", raw)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return xfmt.Errorf("invalid module_catalog_index_url %q: host is required", raw)
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(parsed.Path)), "/index.json") {
		return xfmt.Errorf("invalid module_catalog_index_url %q: must point to an index.json resource", raw)
	}

	return nil
}
