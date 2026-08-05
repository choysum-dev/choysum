// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package injectappmodel

import (
	"strings"

	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

// ReleaseSchedule clears one process-wide NeedInject claim for modelName/app.
func ReleaseSchedule(modelName, app string) {
	app = strings.TrimSpace(app)
	if app == "" {
		return
	}
	if spec, ok := specByName(modelName); ok {
		spec.scheduled.Delete(app)
	}
}

// GeneratedSourceForTest exposes generatedSource for legacy backend tests.
func GeneratedSourceForTest(modelName, modulesPath, application string) string {
	spec, ok := specByName(modelName)
	if !ok {
		return ""
	}
	return generatedSource(spec, modulesPath, application)
}

// IsGeneratedPathForTest reports whether path is the Spec's generated path.
func IsGeneratedPathForTest(modelName, path string) bool {
	spec, ok := specByName(modelName)
	if !ok {
		return false
	}
	return isGeneratedPath(spec, path)
}

// GeneratedPathForTest returns the absolute generated path under modulePath.
func GeneratedPathForTest(modelName, modulePath string) string {
	spec, ok := specByName(modelName)
	if !ok {
		return ""
	}
	return generatedPath(spec, modulePath)
}

// ModelsInForTest filters parser results for the Spec.
func ModelsInForTest(modelName string, results []*parser.ParserResult, modulePath string) []*meta.Model {
	spec, ok := specByName(modelName)
	if !ok {
		return nil
	}
	return modelsIn(spec, results, modulePath)
}

// HandwrittenForTest filters non-generated models.
func HandwrittenForTest(modelName string, models []*meta.Model) []*meta.Model {
	spec, ok := specByName(modelName)
	if !ok {
		return nil
	}
	return handwrittenModels(spec, models)
}

// GeneratedForTest filters generated-path models.
func GeneratedForTest(modelName string, models []*meta.Model) []*meta.Model {
	spec, ok := specByName(modelName)
	if !ok {
		return nil
	}
	return generatedModels(spec, models)
}
