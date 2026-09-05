// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

// RemapOffset maps a generated (service script) offset to a source .vue offset.
// Segments use half-open ranges [GeneratedStart, GeneratedEnd).
// Mappings with Verification == false are skipped.
// When multiple segments cover pos, the one with the greatest GeneratedStart wins.
func RemapOffset(mappings []SpanMapping, generatedPos int) (sourcePos int, ok bool) {
	bestGenStart := -1
	bestSource := 0
	found := false
	for _, m := range mappings {
		if m.Verification == false {
			continue
		}
		if generatedPos < m.GeneratedStart || generatedPos >= m.GeneratedEnd {
			continue
		}
		genLen := m.GeneratedEnd - m.GeneratedStart
		srcLen := m.SourceEnd - m.SourceStart
		pos := m.SourceStart
		if genLen > 0 && srcLen > 0 {
			delta := generatedPos - m.GeneratedStart
			if delta >= srcLen {
				delta = srcLen - 1
			}
			pos = m.SourceStart + delta
		}
		if !found || m.GeneratedStart > bestGenStart {
			found = true
			bestGenStart = m.GeneratedStart
			bestSource = pos
		}
	}
	if !found {
		return 0, false
	}
	return bestSource, true
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
