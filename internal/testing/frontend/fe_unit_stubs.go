// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package frontend

import "strings"

// feUnitStubPaths holds absolute paths to FE unit stub files under testdata/fe_stubs.
type feUnitStubPaths struct {
	ElementPlus    string
	Icons          string
	Router         string
	PageMount      string
	OPage          string
	ChildView      string
	AuthStore      string
	I18n           string
	I18nStore      string
	Registry       string
	Scope          string
	Permission     string
	PageComposable string
}

func feUnitPackageStubPath(importPath string, stubs feUnitStubPaths) (string, bool) {
	switch importPath {
	case "element-plus":
		return stubs.ElementPlus, true
	case "@element-plus/icons-vue":
		return stubs.Icons, true
	case "vue-router":
		return stubs.Router, true
	case "@choysum/page-mount":
		return stubs.PageMount, true
	default:
		return "", false
	}
}

// feUnitPathStubPath resolves product import paths to FE unit stubs.
// p is the raw import path; joined is ResolveDir+path when relative; importer is the importing file.
func feUnitPathStubPath(p, joined, importer string, stubs feUnitStubPaths) (string, bool) {
	isPageOrView := strings.HasSuffix(importer, ".vue") &&
		(strings.Contains(importer, "/web/pages/") || strings.Contains(importer, "/web/views/"))

	switch {
	case strings.Contains(joined, "/web/web/components/") || strings.Contains(p, "/web/web/components/") || strings.Contains(p, "@/web/web/components/"):
		if strings.Contains(p, "OPage.vue") || strings.Contains(joined, "OPage.vue") {
			return stubs.OPage, true
		}
		if strings.HasSuffix(p, ".vue") || strings.Contains(joined, ".vue") {
			return stubs.ChildView, true
		}
	case strings.HasSuffix(p, "FormView.vue") || strings.HasSuffix(joined, "FormView.vue") ||
		strings.HasSuffix(p, "ListView.vue") || strings.HasSuffix(joined, "ListView.vue") ||
		strings.HasSuffix(p, "KanbanView.vue") || strings.HasSuffix(joined, "KanbanView.vue"):
		if isPageOrView {
			return stubs.ChildView, true
		}
	case strings.Contains(p, "web/web/stores/registry") || strings.Contains(joined, "/web/web/stores/registry"):
		// Never stub @/core/web/stores/registry (core unit tests need registerStoreFactory).
		// Skip FE *.test.ts importers that assert the real registry surface.
		if strings.Contains(importer, ".test.ts") || strings.Contains(importer, ".spec.ts") {
			return "", false
		}
		return stubs.Registry, true
	case strings.Contains(p, "storeScopeManager") || strings.Contains(joined, "storeScopeManager"):
		if strings.Contains(importer, ".test.ts") || strings.Contains(importer, ".spec.ts") {
			return "", false
		}
		return stubs.Scope, true
	case strings.Contains(p, "composables/usePermission") || strings.HasSuffix(p, "/usePermission") || strings.HasSuffix(p, "/usePermission.ts"):
		if isPageOrView {
			return stubs.Permission, true
		}
	case strings.Contains(p, "composables/usePageContext") || strings.Contains(joined, "composables/usePageContext") ||
		strings.Contains(p, "composables/useListView") || strings.Contains(joined, "composables/useListView"):
		if isPageOrView {
			return stubs.PageComposable, true
		}
	case strings.Contains(joined, "/auth/web/stores/auth") || (strings.Contains(p, "stores/auth") && !strings.Contains(importer, "/stores/auth/")):
		fromProduct := isPageOrView || strings.Contains(importer, "Login.vue")
		barrel := strings.HasSuffix(p, "/stores/auth") ||
			p == "../stores/auth" ||
			p == "./stores/auth" ||
			strings.Contains(p, "@/auth/web/stores/auth") ||
			strings.HasSuffix(joined, "/stores/auth") ||
			strings.HasSuffix(joined, "/stores/auth/index") ||
			strings.HasSuffix(joined, "/stores/auth/index.ts") ||
			strings.HasSuffix(joined, "/stores/auth/index.js")
		if fromProduct && barrel {
			return stubs.AuthStore, true
		}
	case (strings.Contains(p, "web/web/i18n") || strings.Contains(joined, "/web/web/i18n")) &&
		!strings.Contains(p, "i18nStore") && !strings.Contains(joined, "i18nStore"):
		if isPageOrView {
			return stubs.I18n, true
		}
	case strings.HasSuffix(p, "/stores/i18nStore") ||
		strings.HasSuffix(p, "/stores/i18nStore.ts") ||
		strings.HasSuffix(p, "/stores/i18nStore/index") ||
		strings.HasSuffix(p, "/stores/i18nStore/index.ts") ||
		p == "@/web/web/stores/i18nStore" ||
		strings.HasSuffix(joined, "/stores/i18nStore") ||
		strings.HasSuffix(joined, "/stores/i18nStore/index.ts"):
		// Keep real i18nStore for mapping/unit tests that assert its public surface.
		if strings.Contains(importer, ".test.ts") || strings.Contains(importer, ".spec.ts") {
			return "", false
		}
		return stubs.I18nStore, true
	case strings.Contains(p, "web/stores/i18nStore/") || strings.Contains(joined, "/stores/i18nStore/"):
		// Subpath imports from product SFCs → avoid dayjs locale graph.
		if isPageOrView {
			return stubs.I18nStore, true
		}
	}
	return "", false
}
