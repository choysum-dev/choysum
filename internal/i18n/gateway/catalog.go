// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package i18ngateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

// CatalogHash computes the multi-app catalogHash for a lang (D2).
// Input is app → termHash; apps are sorted and joined as "app:termHash\n".
func CatalogHash(appTermHashes map[string]string) string {
	if len(appTermHashes) == 0 {
		return store.EmptyTermHash()
	}
	apps := make([]string, 0, len(appTermHashes))
	for app := range appTermHashes {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		apps = append(apps, app)
	}
	sort.Strings(apps)
	var b strings.Builder
	for _, app := range apps {
		hash := strings.TrimSpace(appTermHashes[app])
		if hash == "" {
			hash = store.EmptyTermHash()
		}
		b.WriteString(app)
		b.WriteByte(':')
		b.WriteString(hash)
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// LangToLocale maps terminology lang (zh_CN) to format locale (zh-CN).
func LangToLocale(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}
	return strings.ReplaceAll(lang, "_", "-")
}

// maxLangCodeLen matches base.Language.Code (varchar 16).
const maxLangCodeLen = 16

// validLang reports whether lang is a safe terminology language code
// (alphanumeric, underscore, hyphen; length ≤ Language.Code).
func validLang(lang string) bool {
	if lang == "" || len(lang) > maxLangCodeLen {
		return false
	}
	for _, r := range lang {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// frameworkModuleName is hosted in each real app's translation_term table (Scheme A).
const frameworkModuleName = "core"

func moduleBelongsToApp(modules []string, module string) bool {
	module = strings.TrimSpace(module)
	if module == "" {
		return false
	}
	for _, m := range modules {
		if m == module {
			return true
		}
	}
	return false
}

// installedModulesByApp returns ApplicationStr → installed module names.
// Application "core" is skipped (no core.I18n). Framework module "core" is
// appended to every host app so GetTranslations returns Module=core terms.
func installedModulesByApp(runtimeScope scope.Scope) (map[string][]string, error) {
	out := map[string][]string{}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return out, nil
	}
	session := runtimeScope.Session()
	if !session.Migrator().HasTable((&meta.IrModule{}).TableName()) {
		return out, nil
	}

	var modules []meta.IrModule
	if err := session.Where("status = ?", meta.Installed).Find(&modules).Error; err != nil {
		return nil, fmt.Errorf("list installed modules: %w", err)
	}

	seen := map[string]map[string]struct{}{}
	frameworkInstalled := false
	for _, mod := range modules {
		app := strings.TrimSpace(mod.ApplicationStr)
		name := strings.TrimSpace(mod.Name)
		if name == frameworkModuleName {
			frameworkInstalled = true
		}
		if app == "" || app == frameworkModuleName || name == "" {
			continue
		}
		if seen[app] == nil {
			seen[app] = map[string]struct{}{}
		}
		if _, ok := seen[app][name]; ok {
			continue
		}
		seen[app][name] = struct{}{}
		out[app] = append(out[app], name)
	}
	for app := range out {
		if frameworkInstalled {
			if _, ok := seen[app][frameworkModuleName]; !ok {
				seen[app][frameworkModuleName] = struct{}{}
				out[app] = append(out[app], frameworkModuleName)
			}
		}
		sort.Strings(out[app])
	}
	return out, nil
}
