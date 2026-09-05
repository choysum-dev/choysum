// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import "sort"

// RemapOffset maps a generated (service script) offset to a source .vue offset.
// Segments use half-open ranges [GeneratedStart, GeneratedEnd).
// Mappings with Verification == false are skipped.
// When multiple segments cover pos, the one with the greatest GeneratedStart wins.
func RemapOffset(mappings []SpanMapping, generatedPos int) (sourcePos int, ok bool) {
	type hit struct {
		sourcePos int
		genStart  int
	}
	var hits []hit
	for _, m := range mappings {
		if m.Verification == false {
			continue
		}
		if generatedPos < m.GeneratedStart || generatedPos >= m.GeneratedEnd {
			continue
		}
		genLen := m.GeneratedEnd - m.GeneratedStart
		srcLen := m.SourceEnd - m.SourceStart
		if genLen <= 0 || srcLen <= 0 {
			hits = append(hits, hit{sourcePos: m.SourceStart, genStart: m.GeneratedStart})
			continue
		}
		delta := generatedPos - m.GeneratedStart
		if delta >= srcLen {
			delta = srcLen - 1
		}
		hits = append(hits, hit{sourcePos: m.SourceStart + delta, genStart: m.GeneratedStart})
	}
	if len(hits) == 0 {
		return 0, false
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].genStart > hits[j].genStart
	})
	return hits[0].sourcePos, true
}
