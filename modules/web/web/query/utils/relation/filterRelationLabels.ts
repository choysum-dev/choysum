// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Condition, ConditionGroup } from '@/web/web/query/types';
import { isNullOperator } from '@/web/web/query/utils/filter/operators';
import { isGroup } from '@/web/web/query/utils/filter/structures';
import { getFieldMetadataView } from '@/web/web/stores/modelStore';

const RELATION_PREVIEW_TYPES = new Set(['manytoone', 'manytooneref']);

export function relationLabelCacheKey(relationModel: string, id: string): string {
  return `${relationModel}::${id}`;
}

/** Normalize filter leaf values to a non-empty id string, or undefined. */
export function extractRelationId(value: unknown): string | undefined {
  if (value == null) return undefined;
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed || undefined;
  }
  if (typeof value === 'object' && !Array.isArray(value)) {
    const id = (value as { Id?: unknown }).Id;
    if (id == null) return undefined;
    const trimmed = String(id).trim();
    return trimmed || undefined;
  }
  return undefined;
}

function metaTypeOf(meta: { type?: unknown } | undefined): string {
  return String(meta?.type || '').toLowerCase();
}

function relationModelOf(meta: unknown): string | undefined {
  const model = getFieldMetadataView(meta as any).relationModel;
  const trimmed = typeof model === 'string' ? model.trim() : '';
  return trimmed || undefined;
}

function isRelationPreviewMeta(meta: { type?: unknown; relationModel?: unknown } | undefined): boolean {
  if (!meta) return false;
  if (!RELATION_PREVIEW_TYPES.has(metaTypeOf(meta))) return false;
  return !!relationModelOf(meta);
}

/**
 * Walk a filter draft and collect target-model → id sets for manytoone / manytooneref leaves.
 * Skips is / is not; expands in / not in arrays.
 */
export function collectRelationIdsFromGroup(
  root: ConditionGroup | null | undefined,
  fieldsMetadata: Record<string, { type?: unknown; relationModel?: unknown } | undefined> | null | undefined
): Map<string, Set<string>> {
  const batch = new Map<string, Set<string>>();
  const meta = fieldsMetadata || {};

  const add = (model: string, id: string) => {
    if (!batch.has(model)) batch.set(model, new Set());
    batch.get(model)!.add(id);
  };

  const walkCond = (c: Condition) => {
    if (isNullOperator(c.operator)) return;
    const fieldMeta = meta[c.field];
    if (!isRelationPreviewMeta(fieldMeta)) return;
    const model = relationModelOf(fieldMeta)!;
    const raw = c.value;
    if (Array.isArray(raw)) {
      for (const entry of raw) {
        const id = extractRelationId(entry);
        if (id) add(model, id);
      }
      return;
    }
    const id = extractRelationId(raw);
    if (id) add(model, id);
  };

  const walk = (g: ConditionGroup) => {
    const children = Array.isArray(g.children) ? g.children : [];
    for (const ch of children) {
      if (isGroup(ch)) walk(ch);
      else walkCond(ch as Condition);
    }
  };

  if (root) walk(root);
  return batch;
}

/** Stable signature for watch/debounce of a collected id batch. */
export function relationIdBatchSignature(batch: Map<string, Set<string>>): string {
  const parts: string[] = [];
  for (const model of Array.from(batch.keys()).sort()) {
    const ids = Array.from(batch.get(model) || []).sort();
    parts.push(`${model}=${ids.join(',')}`);
  }
  return parts.join('|');
}

/**
 * Normalize relation preview values to id / id[], then map through resolveLabel.
 * Objects without a usable Id become null (matches prior manytoone unwrap).
 */
export function formatRelationPreviewValue(raw: unknown, resolveLabel: (id: string) => string): unknown {
  const toLabeled = (entry: unknown): unknown => {
    const id = extractRelationId(entry);
    if (id) return resolveLabel(id);
    if (entry && typeof entry === 'object' && !Array.isArray(entry) && 'Id' in (entry as object)) {
      return null;
    }
    return entry;
  };
  if (Array.isArray(raw)) return raw.map(toLabeled);
  return toLabeled(raw);
}
