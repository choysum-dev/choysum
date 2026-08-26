// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package record

import exportpkg "github.com/choysum-dev/choysum/pkg/export"

// DefaultExportFields returns the approved default CSV columns for a model (EX14).
func DefaultExportFields(model string) ([]string, error) {
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
			"UpdatedAt",
		}, nil
	default:
		return nil, exportpkg.Errorf(exportpkg.CodeModelNotFound, "export is not implemented for model "+model)
	}
}
