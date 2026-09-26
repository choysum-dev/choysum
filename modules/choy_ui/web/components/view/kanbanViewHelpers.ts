// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for ChoyKanbanView: lane keys, card identity, and move payloads.
 */

export type ChoyKanbanCard = {
  id: string;
  title: string;
  subtitle?: string;
  laneKey: string;
  payload?: Record<string, unknown>;
};

export type ChoyKanbanLane = {
  key: string;
  label: string;
  cards: ChoyKanbanCard[];
};

export type ChoyKanbanMove = {
  cardId: string;
  fromLaneKey: string;
  toLaneKey: string;
  /**
   * For `applyChoyKanbanMove` input: drop target index before the card is removed.
   * For `card-move` emit: index within the destination lane after the move.
   */
  toIndex: number;
};

/** Stable string key for a lane (empty → `"__unset__"`). */
export function normalizeChoyKanbanLaneKey(raw: unknown): string {
  const s = String(raw ?? '').trim();
  return s || '__unset__';
}

/** Prefer Id / id; fall back to index-based synthetic id. */
export function resolveChoyKanbanCardId(raw: Record<string, unknown>, index: number): string {
  const id = raw.Id ?? raw.id;
  if (id != null && String(id).trim() !== '') {
    return String(id);
  }
  return `__row_${index}`;
}

/**
 * Group flat rows into lanes by `laneField`. Unknown lane keys are kept under
 * their normalized key with a generated label.
 */
export function groupRowsIntoChoyKanbanLanes(
  rows: Record<string, unknown>[],
  opts: {
    laneField: string;
    laneDefs?: Array<{ key: string; label: string }>;
    titleField?: string;
    subtitleField?: string;
  },
): ChoyKanbanLane[] {
  const titleField = opts.titleField || 'Title';
  const subtitleField = opts.subtitleField;
  const defs = opts.laneDefs ?? [];
  const byKey = new Map<string, ChoyKanbanLane>();

  for (const def of defs) {
    const key = normalizeChoyKanbanLaneKey(def.key);
    byKey.set(key, { key, label: def.label || key, cards: [] });
  }

  rows.forEach((row, index) => {
    const laneKey = normalizeChoyKanbanLaneKey(row[opts.laneField]);
    let lane = byKey.get(laneKey);
    if (!lane) {
      lane = { key: laneKey, label: laneKey === '__unset__' ? 'Unset' : laneKey, cards: [] };
      byKey.set(laneKey, lane);
    }
    const titleRaw = row[titleField];
    lane.cards.push({
      id: resolveChoyKanbanCardId(row, index),
      title: titleRaw == null || titleRaw === '' ? resolveChoyKanbanCardId(row, index) : String(titleRaw),
      subtitle:
        subtitleField && row[subtitleField] != null && row[subtitleField] !== ''
          ? String(row[subtitleField])
          : undefined,
      laneKey,
      payload: row,
    });
  });

  if (defs.length) {
    const ordered: ChoyKanbanLane[] = [];
    const seen = new Set<string>();
    for (const def of defs) {
      const key = normalizeChoyKanbanLaneKey(def.key);
      // Duplicate / equivalent defs must not push the same lane twice.
      if (seen.has(key)) continue;
      const lane = byKey.get(key);
      if (lane) {
        ordered.push(lane);
        seen.add(key);
      }
    }
    for (const [key, lane] of byKey) {
      if (!seen.has(key)) ordered.push(lane);
    }
    return ordered;
  }

  return [...byKey.values()];
}

/**
 * Apply a card move immutably. Returns null when the card or lanes are missing,
 * or when from/to are identical with no index change.
 */
export function applyChoyKanbanMove(lanes: ChoyKanbanLane[], move: ChoyKanbanMove): ChoyKanbanLane[] | null {
  const fromIdx = lanes.findIndex(l => l.key === move.fromLaneKey);
  const toIdx = lanes.findIndex(l => l.key === move.toLaneKey);
  if (fromIdx < 0 || toIdx < 0) return null;

  const next = lanes.map(l => ({ ...l, cards: [...l.cards] }));
  const fromLane = next[fromIdx]!;
  const toLane = next[toIdx]!;
  const cardIndex = fromLane.cards.findIndex(c => c.id === move.cardId);
  if (cardIndex < 0) return null;

  const [card] = fromLane.cards.splice(cardIndex, 1);
  const moved: ChoyKanbanCard = { ...card!, laneKey: move.toLaneKey };
  // Drop targets use pre-removal indexes. Same-lane downward moves must subtract
  // one after the splice so the card lands before the hovered target, not after.
  const sameLane = fromIdx === toIdx;
  const rawInsertAt = sameLane && cardIndex < move.toIndex ? move.toIndex - 1 : move.toIndex;
  const insertAt = Math.max(0, Math.min(rawInsertAt, toLane.cards.length));
  if (sameLane && insertAt === cardIndex) {
    return null;
  }
  toLane.cards.splice(insertAt, 0, moved);
  return next;
}
