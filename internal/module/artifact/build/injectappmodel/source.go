// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// virtualServiceEntryPath is the in-memory service entry for EnsureServiceEntry.
// It is never written to package.json or the module source tree.
func virtualServiceEntryPath(modulePath string) string {
	return filepath.ToSlash(filepath.Clean(filepath.Join(strings.TrimSpace(modulePath), "service/index.ts")))
}

// virtualServiceEntrySource is a minimal TS module so esbuild can treat the
// path as an entry; inject models arrive via EntryPointImports.
func virtualServiceEntrySource() string {
	return "// Virtual service entry ensured by injectappmodel (TranslationTerm).\nexport {};\n"
}

// generatedSource builds C2 thin-class source with absolute imports so esbuild can
// resolve them even when the pseudo __generated__ directory is not on disk.
func generatedSource(spec *Spec, modulesPath, application string) string {
	modulesPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(modulesPath)))
	application = strings.TrimSpace(application)
	if application == "" {
		application = "application"
	}
	coreService := filepath.ToSlash(filepath.Join(modulesPath, "core/service/index.ts"))
	baseModel := filepath.ToSlash(filepath.Join(modulesPath, spec.BaseModelFile))
	baseImportName := spec.ModelName + "BaseModel"

	var modelOpts string
	if spec.SoftDeleteFalse {
		modelOpts = fmt.Sprintf("{ application: %s, softDelete: false }", strconv.Quote(application))
	} else {
		modelOpts = fmt.Sprintf("{ application: %s }", strconv.Quote(application))
	}

	return fmt.Sprintf(`import { Model } from %s
import %s from %s

@Model('%s', %s)
export default class %s extends %s {}
`, strconv.Quote(coreService), baseImportName, strconv.Quote(baseModel), spec.ModelName, modelOpts, spec.ModelName, baseImportName)
}
