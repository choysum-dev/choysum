// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package jsengine

import (
	"strconv"

	"github.com/zeebo/xxh3"
	xfmt "golang.org/x/exp/errors/fmt"
)

// DedupeInitScripts deduplicates init scripts by content hash while enforcing
// that the same FileName cannot map to different contents.
//
// Rules:
// - If two scripts have identical content, only the first is kept.
// - If two scripts share the same FileName but different content, returns error.
// - Order of remaining scripts is preserved.
func DedupeInitScripts(scripts []*JsScript) ([]*JsScript, error) {
	if len(scripts) == 0 {
		return scripts, nil
	}

	seenByHash := map[uint64]bool{}
	seenByName := map[string]uint64{}

	out := make([]*JsScript, 0, len(scripts))
	for _, s := range scripts {
		if s == nil {
			continue
		}
		h := xxh3.HashString(s.Content)
		if s.FileName != "" {
			if prev, ok := seenByName[s.FileName]; ok && prev != h {
				return nil, xfmt.Errorf("initScripts name conflict: %s (hash %s vs %s)", s.FileName, strconv.FormatUint(prev, 16), strconv.FormatUint(h, 16))
			}
			seenByName[s.FileName] = h
		}
		if seenByHash[h] {
			continue
		}
		seenByHash[h] = true
		out = append(out, s)
	}

	return out, nil
}
