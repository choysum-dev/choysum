// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package scripts

import (
	"encoding/json"
	"fmt"
)

type Result struct {
	Ok        bool           `json:"ok"`
	Message   string         `json:"message,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	Code      string         `json:"code,omitempty"`
	Retriable bool           `json:"retriable,omitempty"`
}

func ParseResult(raw any) (Result, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return Result{}, fmt.Errorf("HOOK_EXCEPTION: marshal result: %w", err)
	}
	var result Result
	if err := json.Unmarshal(b, &result); err != nil {
		return Result{}, fmt.Errorf("HOOK_EXCEPTION: parse result: %w", err)
	}
	return result, nil
}
