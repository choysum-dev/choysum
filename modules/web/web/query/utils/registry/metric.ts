// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Metric registry per store with de-duplication & conflict detection
// - Multiple registrations of the same (field, agg[, alias]) increase refCount.
// - Alias has priority for key uniqueness; if alias absent use field__agg (count => __count).
// - Conflict: same key but different spec => keep first, ignore later & warn.

type StoreId = string;

export interface MetricSpec {
  field: string; // source field path
  agg: string; // aggregation op: sum | avg | min | max | count | count_distinct | ... (case-insensitive)
  alias?: string; // exposed name/alias (unique key if provided)
}

interface MetricEntry {
  spec: MetricSpec; // canonical spec stored
  refCount: number; // number of active registrations
  sources?: Set<string>; // optional logical sources (components) – future use
}

const metricMap: Map<StoreId, Map<string, MetricEntry>> = new Map();

function normalizeSpec(spec: MetricSpec): MetricSpec {
  return {
    field: String(spec.field).trim(),
    agg: String(spec.agg).trim().toLowerCase(),
    alias: spec.alias && spec.alias.trim() ? spec.alias.trim() : undefined,
  };
}

function keyOf(spec: MetricSpec): string {
  if (spec.alias) return spec.alias; // alias wins
  if (spec.agg === 'count') return '__count';
  return `${spec.field}__${spec.agg}`;
}

function equalsSpec(a: MetricSpec, b: MetricSpec): boolean {
  return a.field === b.field && a.agg === b.agg && (a.alias || '') === (b.alias || '');
}

export function registerMetric(storeId: StoreId, rawSpec: MetricSpec, sourceId?: string): void {
  const spec = normalizeSpec(rawSpec);
  let storeMetrics = metricMap.get(storeId);
  if (!storeMetrics) {
    storeMetrics = new Map();
    metricMap.set(storeId, storeMetrics);
  }
  const key = keyOf(spec);
  const existing = storeMetrics.get(key);
  if (!existing) {
    storeMetrics.set(key, { spec, refCount: 1, sources: sourceId ? new Set([sourceId]) : undefined });
    return;
  }
  // Same key exists; check for spec equality
  if (equalsSpec(spec, existing.spec)) {
    existing.refCount++;
    if (sourceId) existing.sources?.add(sourceId);
    return;
  }
  // Conflict: key collision with different spec – keep original, warn once
  if (!(existing as any)._conflictWarned) {
    // eslint-disable-next-line no-console
    console.warn('[MetricRegistry] metric conflict ignored', { storeId, key, existing: existing.spec, incoming: spec });
    (existing as any)._conflictWarned = true;
  }
  // Ignore new spec
}

export function unregisterMetric(storeId: StoreId, rawSpec: MetricSpec, sourceId?: string): void {
  const spec = normalizeSpec(rawSpec);
  const storeMetrics = metricMap.get(storeId);
  if (!storeMetrics) return;
  const key = keyOf(spec);
  const existing = storeMetrics.get(key);
  if (!existing) return; // no-op
  if (!equalsSpec(spec, existing.spec)) {
    // Mismatch on removal: likely conflict earlier; ignore silently
    return;
  }
  existing.refCount--;
  if (sourceId) existing.sources?.delete(sourceId);
  if (existing.refCount <= 0) storeMetrics.delete(key);
  if (storeMetrics.size === 0) metricMap.delete(storeId);
}

export function exportMetrics(storeId: StoreId): MetricSpec[] {
  const storeMetrics = metricMap.get(storeId);
  if (!storeMetrics) return [];
  // Stable sort by key for deterministic ordering
  return Array.from(storeMetrics.entries())
    .sort(([aKey], [bKey]) => (aKey < bKey ? -1 : aKey > bKey ? 1 : 0))
    .map(([, entry]) => entry.spec);
}

export function clearMetricsByStore(storeId: StoreId): void {
  metricMap.delete(storeId);
}

// Helper (optional): introspection for debugging
export function _debugMetrics(storeId: StoreId): Array<{ key: string; spec: MetricSpec; refCount: number }> {
  const out: Array<{ key: string; spec: MetricSpec; refCount: number }> = [];
  const storeMetrics = metricMap.get(storeId);
  if (!storeMetrics) return out;
  for (const [key, entry] of storeMetrics.entries()) out.push({ key, spec: entry.spec, refCount: entry.refCount });
  return out.sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0));
}
