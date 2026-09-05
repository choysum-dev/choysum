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
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].genStart > hits[j].genStart
	})
	return hits[0].sourcePos, true
}

// RemapRange maps a generated diagnostic [start, start+length) into source
// coordinates. Length is recomputed from the remapped inclusive end.
func RemapRange(mappings []SpanMapping, generatedStart, generatedLength int) (sourceStart, sourceLength int, ok bool) {
	srcStart, ok := RemapOffset(mappings, generatedStart)
	if !ok {
		return 0, 0, false
	}
	if generatedLength <= 0 {
		return srcStart, 0, true
	}
	last := generatedStart + generatedLength - 1
	srcEnd, okEnd := RemapOffset(mappings, last)
	if !okEnd {
		return srcStart, 1, true
	}
	if srcEnd < srcStart {
		return srcStart, 0, true
	}
	return srcStart, srcEnd - srcStart + 1, true
}
