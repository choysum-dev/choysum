// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package i18nimport loads module PO catalogs into per-application TranslationTerm rows.
// PO text is parsed via internal/i18n/po (GNU gettext subset). Runtime lookup never uses
// gettext Get() — only TermStore cache after Warm/Invalidate.
package i18nimport

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/po"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/scope"
	"gorm.io/gorm"
)

// ImportStats summarizes one PO import.
type ImportStats struct {
	Upserted   int
	SkippedOverride int
	RejectedNoCtxt  int
	SkippedObsolete int
	Lang       string
}

// ImportModulePo parses poText and upserts packaged terms for (application, module, lang).
// Existing Source=override rows are not overwritten. Obsolete (#~) entries are ignored
// for DB writes and never DELETE existing rows (D12a). Entries without msgctxt are
// rejected and logged (D12c).
func ImportModulePo(runtimeScope scope.Scope, reg *store.Registry, application, module, lang string, poText []byte) (*ImportStats, error) {
	application = strings.TrimSpace(application)
	module = strings.TrimSpace(module)
	lang = strings.TrimSpace(lang)
	stats := &ImportStats{Lang: lang}

	if application == "" || application == "core" || module == "" || lang == "" {
		return stats, nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return stats, fmt.Errorf("import po: missing runtime session")
	}

	entries, err := po.Parse(bytes.NewReader(poText))
	if err != nil {
		return stats, fmt.Errorf("parse po: %w", err)
	}

	tableName := i18nmodels.TranslationTermTableName(application)
	if err := i18nmodels.EnsureTranslationTermTable(runtimeScope, application); err != nil {
		return stats, err
	}
	session := runtimeScope.Session()
	logger := runtimeScope.Logger()
	if logger == nil {
		logger = slog.Default()
	}

	for _, entry := range entries {
		if po.IsHeader(entry) {
			continue
		}
		if entry.Obsolete {
			stats.SkippedObsolete++
			continue
		}
		if strings.TrimSpace(entry.Msgctxt) == "" {
			stats.RejectedNoCtxt++
			logger.Warn("i18n import rejected PO entry without msgctxt",
				"application", application, "module", module, "lang", lang, "msgid", entry.Msgid)
			continue
		}

		kind := i18nmodels.KindLiteral
		var existing i18nmodels.TranslationTerm
		err := session.Table(tableName).
			Where("module = ? AND lang = ? AND scope = ? AND src = ? AND kind = ?",
				module, lang, entry.Msgctxt, entry.Msgid, kind).
			Take(&existing).Error
		if err == nil {
			if existing.Source == i18nmodels.SourceOverride {
				stats.SkippedOverride++
				continue
			}
			existing.Value = entry.Msgstr
			existing.Source = i18nmodels.SourcePackaged
			existing.Application = application
			if comments := joinComments(entry); comments != "" {
				existing.Comments = comments
			}
			if err := session.Table(tableName).Save(&existing).Error; err != nil {
				return stats, fmt.Errorf("update term: %w", err)
			}
			stats.Upserted++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return stats, fmt.Errorf("lookup term: %w", err)
		}

		row := i18nmodels.TranslationTerm{
			Application: application,
			Module:      module,
			Lang:        lang,
			Scope:       entry.Msgctxt,
			Src:         entry.Msgid,
			Value:       entry.Msgstr,
			Kind:        kind,
			Source:      i18nmodels.SourcePackaged,
			Comments:    joinComments(entry),
		}
		if err := session.Table(tableName).Create(&row).Error; err != nil {
			return stats, fmt.Errorf("create term: %w", err)
		}
		stats.Upserted++
	}

	if reg != nil {
		reg.RememberModuleApplication(module, application)
		ts := reg.StoreFor(application)
		ts.InvalidateModule(module)
		_ = ts.WarmLanguage(lang)
	}
	return stats, nil
}

// ImportModuleI18nDir imports all *.po files under moduleRoot/i18n (skip if missing).
func ImportModuleI18nDir(runtimeScope scope.Scope, reg *store.Registry, application, module, moduleRoot string) error {
	i18nDir := filepath.Join(moduleRoot, "i18n")
	entries, err := os.ReadDir(i18nDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".po") {
			continue
		}
		lang := strings.TrimSuffix(name, filepath.Ext(name))
		if lang == "" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(i18nDir, name))
		if err != nil {
			return err
		}
		if _, err := ImportModulePo(runtimeScope, reg, application, module, lang, raw); err != nil {
			return fmt.Errorf("import %s: %w", name, err)
		}
	}
	return nil
}

// DeleteModuleTerms removes all terminology rows for module in the application table
// and invalidates the Go cache.
func DeleteModuleTerms(runtimeScope scope.Scope, reg *store.Registry, application, module string) error {
	application = strings.TrimSpace(application)
	module = strings.TrimSpace(module)
	if application == "" || application == "core" || module == "" {
		return nil
	}
	if runtimeScope == nil || runtimeScope.Session() == nil {
		return fmt.Errorf("delete terms: missing runtime session")
	}
	tableName := i18nmodels.TranslationTermTableName(application)
	if !runtimeScope.Session().Migrator().HasTable(tableName) {
		return nil
	}
	if err := runtimeScope.Session().Table(tableName).Where("module = ?", module).Delete(&i18nmodels.TranslationTerm{}).Error; err != nil {
		return err
	}
	if reg != nil {
		reg.StoreFor(application).InvalidateModule(module)
	}
	return nil
}

func joinComments(entry po.Entry) string {
	parts := append([]string{}, entry.TranslatorComments...)
	parts = append(parts, entry.ExtractedComments...)
	if len(entry.References) > 0 {
		parts = append(parts, strings.Join(entry.References, " "))
	}
	return strings.Join(parts, "\n")
}
