// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package i18nimport loads module PO catalogs into per-application TranslationTerm rows.
// PO text is parsed with leonelquinteros/gotext (Po.Parse / Domain iteration only).
// Kind is read from `#. kind:` via internal/i18n/po (gotext does not expose extracted comments).
// Runtime lookup never uses gotext Get() — only TermStore cache after Warm/Invalidate.
package i18nimport

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	i18nmodels "github.com/choysum-dev/choysum/internal/i18n/models"
	"github.com/choysum-dev/choysum/internal/i18n/po"
	"github.com/choysum-dev/choysum/internal/i18n/store"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"
)

// ImportStats summarizes one PO import.
type ImportStats struct {
	Upserted        int
	SkippedOverride int
	RejectedNoCtxt  int
	SkippedObsolete int
	PurgedRetired   int
	Lang            string
}

var retiredS7Kinds = []string{
	"field_label",
	"selection_label",
	"menu",
	"route",
	"action",
}

type poTerm struct {
	scope    string
	src      string
	value    string
	kind     string
	comments string
}

// UpsertPackagedTerms is the sole packaged-term write helper: parse PO → upsert rows
// (skip Source=override) → purge retired S7 kinds → InvalidateModule + WarmLanguage.
// Install/CLI call this path only; do not dial model gRPC for packaged import.
// Obsolete (#~) entries do not prune ordinary rows (D12a). Entries without msgctxt
// are rejected and logged (D12c). Kind defaults to literal; `#. kind: <name>` overrides.
// Schema ownership is TranslationTerm MetaModel migrate; if the physical table is
// still missing, a create-only fallback runs (indexes/columns on existing tables
// are not evolved here).
func UpsertPackagedTerms(runtimeScope scope.Scope, reg *store.Registry, application, module, lang string, poText []byte) (*ImportStats, error) {
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

	terms, rejectedNoCtxt, obsolete := parsePoTerms(poText)
	stats.RejectedNoCtxt = rejectedNoCtxt
	stats.SkippedObsolete = obsolete

	tableName := i18nmodels.TranslationTermTableName(application)
	migrateErr := migrateTranslationTermTableIfMissing(runtimeScope, application, tableName)
	if migrateErr != nil {
		return stats, migrateErr
	}
	logger := runtimeScope.Logger()
	if logger == nil {
		logger = slog.Default()
	}
	if rejectedNoCtxt > 0 {
		logger.Warn("i18n import rejected PO entries without msgctxt",
			"application", application, "module", module, "lang", lang, "count", rejectedNoCtxt)
	}

	var affectedLangs map[string]struct{}
	err := runtimeScope.Transactor().Required(runtimeScope.Context(), func(txScope scope.Scope, _ scope.Transaction) error {
		session := txScope.Session()
		if session == nil {
			return fmt.Errorf("import po: missing transaction session")
		}

		var existingRows []i18nmodels.TranslationTerm
		if err := session.Table(tableName).
			Where("module = ? AND lang = ?", module, lang).
			Find(&existingRows).Error; err != nil {
			return fmt.Errorf("list existing terms: %w", err)
		}
		existingByKey := make(map[string]*i18nmodels.TranslationTerm, len(existingRows))
		for i := range existingRows {
			row := &existingRows[i]
			existingByKey[termLookupKey(row.Scope, row.Src, row.Kind)] = row
		}

		toCreate := make([]i18nmodels.TranslationTerm, 0)
		toUpdate := make([]i18nmodels.TranslationTerm, 0)
		for _, term := range terms {
			kind := i18nmodels.NormalizeKind(term.kind)
			key := termLookupKey(term.scope, term.src, kind)
			if existing, ok := existingByKey[key]; ok {
				if existing.Source == i18nmodels.SourceOverride {
					stats.SkippedOverride++
					continue
				}
				existing.Value = term.value
				existing.Source = i18nmodels.SourcePackaged
				existing.Application = application
				existing.Kind = kind
				if term.comments != "" {
					existing.Comments = term.comments
				}
				toUpdate = append(toUpdate, *existing)
				stats.Upserted++
				continue
			}

			toCreate = append(toCreate, i18nmodels.TranslationTerm{
				Application: application,
				Module:      module,
				Lang:        lang,
				Scope:       term.scope,
				Src:         term.src,
				Value:       term.value,
				Kind:        kind,
				Source:      i18nmodels.SourcePackaged,
				Comments:    term.comments,
			})
		}
		const importBatchSize = 100
		for i := 0; i < len(toUpdate); i += importBatchSize {
			end := i + importBatchSize
			if end > len(toUpdate) {
				end = len(toUpdate)
			}
			batch := toUpdate[i:end]
			if err := session.Table(tableName).Save(&batch).Error; err != nil {
				return fmt.Errorf("update terms: %w", err)
			}
		}
		if len(toCreate) > 0 {
			if err := session.Table(tableName).CreateInBatches(toCreate, importBatchSize).Error; err != nil {
				return fmt.Errorf("create terms: %w", err)
			}
			stats.Upserted += len(toCreate)
		}

		langs, purged, err := purgeRetiredS7Terms(session, tableName, module)
		if err != nil {
			return err
		}
		stats.PurgedRetired = purged
		affectedLangs = langs
		return nil
	})
	if err != nil {
		return stats, err
	}

	if reg != nil {
		// Framework module "core" is hosted in every real app table; never pin
		// module→application to a single host (breaks multi-app GetTranslations).
		if module != "core" {
			reg.RememberModuleApplication(module, application)
		}
		ts := reg.StoreFor(application)
		ts.InvalidateModule(module)
		if affectedLangs == nil {
			affectedLangs = map[string]struct{}{}
		}
		affectedLangs[lang] = struct{}{}
		for affectedLang := range affectedLangs {
			if err := ts.WarmLanguage(affectedLang); err != nil {
				return stats, fmt.Errorf("warm terminology cache for %s: %w", affectedLang, err)
			}
		}
	}
	return stats, nil
}

// Swapped in tests to force migrate failures without closing the DB.
var migrateTranslationTermTable = i18nmodels.MigrateTranslationTermTable

// migrateTranslationTermTableIfMissing creates the physical table when the write
// path runs without a prior MetaModel migrate (tests / install ordering).
// It does not AutoMigrate or re-ensure indexes when the table already exists;
// schema evolution stays on the TranslationTerm MetaModel path.
func migrateTranslationTermTableIfMissing(runtimeScope scope.Scope, application, tableName string) error {
	if runtimeScope.Session().Migrator().HasTable(tableName) {
		return nil
	}
	return migrateTranslationTermTable(runtimeScope, application)
}

func termLookupKey(scopeName, src, kind string) string {
	return scopeName + "\x00" + src + "\x00" + kind
}

func purgeRetiredS7Terms(session *scope.Session, tableName, module string) (map[string]struct{}, int, error) {
	var langs []string
	baseQuery := session.Table(tableName).
		Where("module = ? AND kind IN ?", module, retiredS7Kinds)
	if err := baseQuery.Session(&gorm.Session{}).Distinct("lang").Pluck("lang", &langs).Error; err != nil {
		return nil, 0, fmt.Errorf("list retired S7 term languages: %w", err)
	}
	result := baseQuery.Session(&gorm.Session{}).Unscoped().Delete(&i18nmodels.TranslationTerm{})
	if result.Error != nil {
		return nil, 0, fmt.Errorf("purge retired S7 terms: %w", result.Error)
	}
	affected := make(map[string]struct{}, len(langs))
	for _, affectedLang := range langs {
		affectedLang = strings.TrimSpace(affectedLang)
		if affectedLang != "" {
			affected[affectedLang] = struct{}{}
		}
	}
	return affected, int(result.RowsAffected), nil
}

// parsePoTerms uses gotext Po.Parse for msgstr values and internal po.Parse for Kind
// (`#. kind:`). Returns contexted terms, count of no-msgctxt msgid entries (excluding
// header), and count of obsolete #~ msgid lines (D12a observability).
func parsePoTerms(poText []byte) (terms []poTerm, rejectedNoCtxt int, obsolete int) {
	obsolete = countObsoleteMsgid(poText)
	kindByKey := kindIndexFromPO(poText)

	poObj := gotext.NewPo()
	poObj.Parse(poText)
	domain := poObj.GetDomain()
	if domain == nil {
		return nil, 0, obsolete
	}

	for msgid := range domain.GetTranslations() {
		if msgid == "" {
			continue // header
		}
		rejectedNoCtxt++
	}

	for ctx, byID := range domain.GetCtxTranslations() {
		ctx = strings.TrimSpace(ctx)
		for msgid, tr := range byID {
			if msgid == "" || tr == nil {
				continue
			}
			kind := kindByKey[poEntryKey(ctx, msgid)]
			terms = append(terms, poTerm{
				scope:    ctx,
				src:      msgid,
				value:    msgstrValue(tr),
				kind:     kind,
				comments: strings.Join(tr.Refs, " "),
			})
		}
	}
	return terms, rejectedNoCtxt, obsolete
}

func kindIndexFromPO(poText []byte) map[string]string {
	out := map[string]string{}
	entries, err := po.Parse(bytes.NewReader(poText))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if po.IsHeader(e) || e.Obsolete || e.Msgctxt == "" || e.Msgid == "" {
			continue
		}
		if kind := kindFromExtractedComments(e.ExtractedComments); kind != "" {
			out[poEntryKey(e.Msgctxt, e.Msgid)] = kind
		}
	}
	return out
}

func kindFromExtractedComments(comments []string) string {
	for _, c := range comments {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(strings.ToLower(c), "kind:") {
			return i18nmodels.NormalizeKind(strings.TrimSpace(c[len("kind:"):]))
		}
	}
	return ""
}

func poEntryKey(msgctxt, msgid string) string {
	return msgctxt + "\x00" + msgid
}

func msgstrValue(tr *gotext.Translation) string {
	if tr == nil || tr.Trs == nil {
		return ""
	}
	// Prefer raw msgstr[0]; do not use Translation.Get() (falls back to msgid when empty).
	return tr.Trs[0]
}

func countObsoleteMsgid(poText []byte) int {
	count := 0
	for _, line := range strings.Split(string(poText), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#~") && strings.Contains(trimmed, "msgid") && !strings.Contains(trimmed, "msgid_plural") {
			// Match "#~ msgid" / "#~msgid"
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "#~"))
			if strings.HasPrefix(rest, "msgid") && !strings.HasPrefix(rest, "msgid_plural") {
				count++
			}
		}
	}
	return count
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
		if _, err := UpsertPackagedTerms(runtimeScope, reg, application, module, lang, raw); err != nil {
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
	if err := runtimeScope.Session().Table(tableName).Where("module = ?", module).
		Unscoped().Delete(&i18nmodels.TranslationTerm{}).Error; err != nil {
		return err
	}
	if reg != nil {
		reg.StoreFor(application).InvalidateModule(module)
	}
	return nil
}
