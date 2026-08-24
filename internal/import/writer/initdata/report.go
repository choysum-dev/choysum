// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initdata

import (
	"fmt"
	"strings"

	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// LoadErrorToMessage maps loader errors to import report messages (appendix E.1).
func LoadErrorToMessage(err *dataloader.LoadError) importpkg.Message {
	if err == nil {
		return importpkg.Message{}
	}
	text := strings.TrimSpace(err.Message)
	if err.Cause != nil {
		causeText := err.Cause.Error()
		if text != "" {
			text = fmt.Sprintf("%s: %s", text, causeText)
		} else {
			text = causeText
		}
	}
	if suffix := loadErrorContextSuffix(err); suffix != "" {
		if text != "" {
			text = fmt.Sprintf("%s (%s)", text, suffix)
		} else {
			text = suffix
		}
	}
	row := 0
	if err.RecordIndex >= 0 {
		row = err.RecordIndex + 1
	}
	return importpkg.Message{
		Type:      importpkg.MessageError,
		Row:       row,
		Field:     err.FieldPath,
		Code:      err.Code,
		Text:      text,
		RecordRef: err.Ref,
	}
}

func loadErrorContextSuffix(err *dataloader.LoadError) string {
	parts := make([]string, 0, 4)
	if file := strings.TrimSpace(err.FilePath); file != "" {
		parts = append(parts, "file="+file)
	}
	if module := strings.TrimSpace(err.Module); module != "" {
		parts = append(parts, "module="+module)
	}
	if name := strings.TrimSpace(err.Name); name != "" {
		parts = append(parts, "name="+name)
	}
	if model := strings.TrimSpace(err.Model); model != "" {
		if app := strings.TrimSpace(err.Application); app != "" {
			parts = append(parts, "model="+app+"."+model)
		} else {
			parts = append(parts, "model="+model)
		}
	}
	return strings.Join(parts, ", ")
}
