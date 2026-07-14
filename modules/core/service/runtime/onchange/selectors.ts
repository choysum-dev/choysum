// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Precise row-selector parsing with multi-hop support.
// Supported selector syntax on collection segments:
//   - By position: Lines[2]
//   - By Id: Lines(id=abcd)
// Multi-hop examples: Lines[2].Batches[0].Qty or Lines(id=...).Batches(id=...).Qty
// Paths without selectors are not handled specially here and continue through the existing flow.

export type RowSelector = { kind: 'all' } | { kind: 'pos'; positions: Set<number> } | { kind: 'id'; ids: Set<string> };

export interface ParsedChangedSelectors {
  // Normalized trigger seeds without selectors, used by OnchangeEngine and compute.
  normalizedSeeds: Set<string>;
  // Collection paths that require cascading, including multi-hop dotted keys.
  collectionRoots: Set<string>;
  // Field signals, keyed by collection path and storing leaf fields.
  fieldSignals: Map<string, Set<string>>;
  // Row selector for each collection path.
  selectors: Map<string, RowSelector>;
}

/** Parse one segment such as Name, Name[2], or Name(id=...). */
function parseSegment(seg: string): { name: string; pos?: number; id?: string } | null {
  if (!seg) return null;
  // Both selector syntaxes are accepted; if both appear, prefer id.
  // Examples: Batches[0], Batches(id=d3x...), Batches.
  const m = seg.match(/^([A-Za-z_][A-Za-z0-9_]*)(?:\[(\d+)\])?(?:\((?:id|Id)=([^)]+)\))?$/);
  if (!m) return { name: seg };
  const name = m[1];
  const pos = m[2] != null ? parseInt(m[2], 10) : undefined;
  const id = m[3] != null ? String(m[3]) : undefined;
  return { name, pos: typeof pos === 'number' && Number.isFinite(pos) ? pos : undefined, id };
}

function mergeSelector(dst: RowSelector | undefined, add: RowSelector): RowSelector {
  if (!dst) return add;
  if (dst.kind === 'all' || add.kind === 'all') return { kind: 'all' };
  if (dst.kind === 'pos' && add.kind === 'pos') {
    const s = new Set<number>(dst.positions);
    add.positions.forEach(p => s.add(p));
    return { kind: 'pos', positions: s };
  }
  if (dst.kind === 'id' && add.kind === 'id') {
    const s = new Set<string>(dst.ids);
    add.ids.forEach(p => s.add(p));
    return { kind: 'id', ids: s };
  }
  // Merge mixed selector kinds by promoting them to all for a simpler implementation.
  return { kind: 'all' };
}

/**
 * Parse changed paths while only handling entries that include selectors.
 * Paths without selectors still follow the existing flow.
 * Returns:
 *  - normalizedSeeds: equivalent paths with selectors removed, plus their collection roots.
 *  - collectionRoots: all collection dotted keys encountered, including top-level and multi-hop ones.
 *  - fieldSignals: mapping from collection dotted key to leaf field names.
 *  - selectors: mapping from collection dotted key to row selector.
 */
export function parseChangedSelectors(changed: string[]): ParsedChangedSelectors {
  const normalizedSeeds = new Set<string>();
  const collectionRoots = new Set<string>();
  const fieldSignals = new Map<string, Set<string>>();
  const selectors = new Map<string, RowSelector>();

  for (const raw of changed || []) {
    if (!raw || (!raw.includes('[') && !raw.includes('('))) {
      // Non-selector entries stay on the existing path; copy them here for unified merging.
      if (raw) normalizedSeeds.add(raw);
      continue;
    }

    const segsRaw = raw.split('.').filter(Boolean);
    if (!segsRaw.length) continue;

    const parsed = segsRaw.map(parseSegment);
    if (parsed.some(x => !x)) {
      // Parsing failed, so fall back conservatively to the original value.
      normalizedSeeds.add(raw);
      continue;
    }

    const names = parsed.map(x => x!.name);
    const normalized = names.join('.');
    // Normalized leaf path used to trigger compute fields and handlers.
    normalizedSeeds.add(normalized);

    // Collect selectors. Every segment with a selector is treated as a collection segment.
    const collectionKeys: string[] = [];
    for (let i = 0; i < parsed.length; i++) {
      const p = parsed[i]!;
      if (p.pos == null && p.id == null) continue;
      const key = names.slice(0, i + 1).join('.');
      collectionKeys.push(key);
      collectionRoots.add(key);
      const addSel: RowSelector = p.id != null ? { kind: 'id', ids: new Set([String(p.id)]) } : { kind: 'pos', positions: new Set([p.pos!]) };
      const prev = selectors.get(key);
      selectors.set(key, mergeSelector(prev, addSel));
    }

    // Root-only collection access without explicit selectors is intentionally left to the existing path.
    // Field signals attach to the nearest collection path.
    if (parsed.length >= 2) {
      const leaf = parsed[parsed.length - 1]!;
      const hasLeafSelector = leaf.pos != null || leaf.id != null;
      // Without metadata, leaf segments without selectors are treated as field signals.
      if (!hasLeafSelector) {
        // Find the nearest collection key, which is the last segment carrying a selector.
        let attachKey: string | null = null;
        for (let i = parsed.length - 2; i >= 0; i--) {
          const p = parsed[i]!;
          if (p.pos != null || p.id != null) {
            attachKey = names.slice(0, i + 1).join('.');
            break;
          }
        }
        if (attachKey) {
          if (!fieldSignals.has(attachKey)) fieldSignals.set(attachKey, new Set<string>());
          fieldSignals.get(attachKey)!.add(leaf.name);
        }
      }
    }

    // Add the top-level collection root during normalization so Lines[2].UnitPrice also seeds Lines.
    if (collectionKeys.length) {
      const topRoot = collectionKeys[0].split('.')[0];
      if (topRoot) {
        collectionRoots.add(topRoot);
        normalizedSeeds.add(topRoot);
      }
      // Intermediate collections in multi-hop paths also act as roots during recursion.
      for (const ck of collectionKeys) {
        collectionRoots.add(ck);
        normalizedSeeds.add(ck);
      }
    }
  }

  return { normalizedSeeds, collectionRoots, fieldSignals, selectors };
}
