// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package sync

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/po"
)

// Result summarizes a sync operation.
type Result struct {
	PoPath       string
	Kept         int
	Added        int
	Obsoleted    int
	TotalActive  int
	TotalObsolete int
}

// SyncModulePo merges module pot into lang.po (msgmerge semantics).
// Preserves msgstr for matching (msgctxt, msgid); adds new empty entries;
// marks missing pot entries obsolete without deleting msgstr history (D12a).
func SyncModulePo(moduleRoot, moduleName, langCode string) (*Result, error) {
	moduleRoot = filepath.Clean(moduleRoot)
	if moduleName == "" {
		moduleName = filepath.Base(moduleRoot)
	}
	langCode = strings.TrimSpace(langCode)
	if langCode == "" {
		return nil, fmt.Errorf("lang code is required")
	}

	i18nDir := filepath.Join(moduleRoot, "i18n")
	potPath := filepath.Join(i18nDir, moduleName+".pot")
	poPath := filepath.Join(i18nDir, langCode+".po")

	potData, err := os.ReadFile(potPath)
	if err != nil {
		return nil, fmt.Errorf("read pot: %w", err)
	}
	potEntries, err := po.Parse(bytes.NewReader(potData))
	if err != nil {
		return nil, fmt.Errorf("parse pot: %w", err)
	}

	var existing []po.Entry
	if raw, err := os.ReadFile(poPath); err == nil {
		existing, err = po.Parse(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("parse po: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	byKey := map[string]po.Entry{}
	var header *po.Entry
	for _, e := range existing {
		if po.IsHeader(e) {
			cp := e
			header = &cp
			continue
		}
		byKey[e.Key()] = e
	}

	var out []po.Entry
	if header != nil {
		out = append(out, *header)
	} else {
		out = append(out, defaultHeader(langCode))
	}

	result := &Result{PoPath: poPath}
	seen := map[string]struct{}{}

	for _, pot := range potEntries {
		if po.IsHeader(pot) {
			continue
		}
		if strings.TrimSpace(pot.Msgctxt) == "" {
			// Pot should always have msgctxt; skip invalid.
			continue
		}
		key := pot.Key()
		seen[key] = struct{}{}
		if old, ok := byKey[key]; ok {
			merged := pot
			merged.Msgstr = old.Msgstr
			merged.TranslatorComments = old.TranslatorComments
			merged.Obsolete = false
			// Drop fuzzy if msgstr still present and msgid unchanged.
			merged.Flags = withoutFlag(old.Flags, "fuzzy")
			if old.Msgstr == "" {
				merged.Flags = old.Flags
			}
			out = append(out, merged)
			result.Kept++
		} else {
			fresh := pot
			fresh.Msgstr = ""
			fresh.Obsolete = false
			out = append(out, fresh)
			result.Added++
		}
	}

	for key, old := range byKey {
		if _, ok := seen[key]; ok {
			continue
		}
		if po.IsHeader(old) {
			continue
		}
		old.Obsolete = true
		out = append(out, old)
		result.Obsoleted++
	}

	po.SortEntries(out)
	// Keep header first after sort (SortEntries puts non-obsolete first; header msgid "" sorts early).
	out = moveHeaderFirst(out)

	for _, e := range out {
		if po.IsHeader(e) {
			continue
		}
		if e.Obsolete {
			result.TotalObsolete++
		} else {
			result.TotalActive++
		}
	}

	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		return result, err
	}
	f, err := os.Create(poPath)
	if err != nil {
		return result, err
	}
	defer f.Close()
	if err := po.Write(f, out); err != nil {
		return result, err
	}
	return result, nil
}

func defaultHeader(lang string) po.Entry {
	return po.Entry{
		Msgid: "",
		Msgstr: "Content-Type: text/plain; charset=UTF-8\n" +
			"Content-Transfer-Encoding: 8bit\n" +
			"Language: " + lang + "\n",
	}
}

func withoutFlag(flags []string, name string) []string {
	out := make([]string, 0, len(flags))
	for _, f := range flags {
		if f != name {
			out = append(out, f)
		}
	}
	return out
}

func moveHeaderFirst(entries []po.Entry) []po.Entry {
	var header []po.Entry
	var rest []po.Entry
	for _, e := range entries {
		if po.IsHeader(e) {
			header = append(header, e)
			continue
		}
		rest = append(rest, e)
	}
	return append(header, rest...)
}
