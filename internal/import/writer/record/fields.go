// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import importpkg "github.com/choysum-dev/choysum/pkg/import"

// DefaultImportFields returns the approved default CSV columns for a model.
// Paths use "/" for ManyToOne leaf fields (symmetric with export DescribeFields).
func DefaultImportFields(model string) ([]string, error) {
	switch model {
	case "base.Country":
		return []string{
			"Name",
			"Code",
			"DefaultCurrencyId/Code",
			"ZipRequired",
			"StateRequired",
			"IsActive",
		}, nil
	case "partner.Partner":
		return []string{
			"Name",
			"Code",
			"CompanyId/Code",
			"CustomerRank",
			"SupplierRank",
			"IsActive",
		}, nil
	default:
		return nil, importpkg.Errorf(importpkg.CodeModelNotFound, "import describe is not implemented for model "+model)
	}
}
