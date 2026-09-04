// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Condition, ConditionGroup, NamedFilter } from '@/web/web/query/types';

let _id = 0;

/**
 * Generates a stable local identifier for filter nodes.
 */
export function genId() {
  return `f_${Date.now().toString(36)}_${(++_id).toString(36)}`;
}

/**
 * Deep-clones a filter group while preserving node identifiers (editor drafts).
 * Avoids structuredClone so plain filter trees work in all Vitest environments.
 */
export function deepCloneFilter(f: ConditionGroup): ConditionGroup {
  return {
    id: f.id,
    logic: f.logic,
    ...(f.name !== undefined ? { name: f.name } : {}),
    children: (Array.isArray(f.children) ? f.children : []).map(ch => {
      if (isGroup(ch)) return deepCloneFilter(ch);
      const c = ch as Condition;
      return {
        id: c.id,
        field: c.field,
        operator: c.operator,
        value: cloneFilterValue(c.value),
      };
    }),
  } as ConditionGroup;
}

function cloneFilterValue(value: any): any {
  if (value == null || typeof value !== 'object') return value;
  if (value instanceof Date) return new Date(value.getTime());
  if (Array.isArray(value)) return value.map(cloneFilterValue);
  const out: Record<string, any> = {};
  for (const key of Object.keys(value)) {
    out[key] = cloneFilterValue(value[key]);
  }
  return out;
}

/**
 * Creates a single filter condition node.
 */
export function createCondition(field: string, operator: string, value: any): Condition {
  return { id: genId(), field, operator, value };
}

/**
 * Creates a condition group with the provided logical operator.
 */
export function createFilter(logic: 'And' | 'Or', children: Array<Condition | ConditionGroup> = []): ConditionGroup {
  return { id: genId(), logic, children } as ConditionGroup;
}

/**
 * Deep-clones a filter group and regenerates all node identifiers.
 */
export function cloneFilter(f: ConditionGroup): ConditionGroup {
  return {
    id: genId(),
    logic: f.logic,
    name: f.name,
    children: f.children.map((ch: Condition | ConditionGroup) => (isGroup(ch) ? cloneFilter(ch as ConditionGroup) : { ...(ch as Condition), id: genId() })),
  };
}

/**
 * Reports whether a filter node is a condition group.
 */
export function isGroup(node: Condition | ConditionGroup): node is ConditionGroup {
  return (node as ConditionGroup).children !== undefined;
}

/**
 * Reports whether a filter node is a single condition.
 */
export function isCondition(node: Condition | ConditionGroup): node is Condition {
  return (node as Condition).operator !== undefined && !(node as any).children;
}

/**
 * Convert a backend QueryCondition (leaf tuple or And/Or tree) into a UI ConditionGroup.
 */
function queryConditionToGroup(query: any, name?: string): ConditionGroup | null {
  if (query == null) return null;
  if (Array.isArray(query) && query.length === 3 && typeof query[0] === 'string' && typeof query[1] === 'string') {
    return {
      id: genId(),
      logic: 'And',
      ...(name ? { name } : {}),
      children: [{ id: genId(), field: query[0], operator: query[1], value: query[2] }],
    } as ConditionGroup;
  }
  if (typeof query !== 'object' || Array.isArray(query)) return null;
  if (query.logic && Array.isArray(query.children)) {
    return { ...(query as ConditionGroup), ...(name ? { name } : {}) };
  }
  const andParts = Array.isArray(query.And) ? query.And : null;
  const orParts = Array.isArray(query.Or) ? query.Or : null;
  const parts = andParts || orParts;
  if (!parts || parts.length === 0) return null;
  const children: Array<Condition | ConditionGroup> = [];
  for (const part of parts) {
    const sub = queryConditionToGroup(part);
    if (!sub) continue;
    if (sub.children.length === 1 && !sub.name) children.push(sub.children[0]);
    else children.push(sub);
  }
  if (children.length === 0) return null;
  return {
    id: genId(),
    logic: orParts ? 'Or' : 'And',
    ...(name ? { name } : {}),
    children,
  } as ConditionGroup;
}

/**
 * Normalizes named filters or groups into a ConditionGroup array.
 */
export function toFilters(input?: NamedFilter | NamedFilter[] | ConditionGroup | ConditionGroup[] | null): ConditionGroup[] {
  if (!input) return [];
  const arr = Array.isArray(input) ? input : [input];
  const out: ConditionGroup[] = [];
  for (const it of arr) {
    if (!it) continue;
    if (typeof it === 'object' && 'query' in it) {
      const nf = it as NamedFilter;
      if (!nf.name) continue;
      if (nf.query == null) continue;
      const group = queryConditionToGroup(nf.query, nf.name);
      if (!group) throw new Error('invalid_named_filter_query');
      out.push(group);
    } else if (isGroup(it as any)) {
      out.push(it as ConditionGroup);
    } else if (typeof it === 'object' && !Array.isArray(it)) {
      // Incomplete NamedFilter-like objects without query are skipped.
      continue;
    } else {
      throw new Error('invalid_named_filter_query');
    }
  }
  return out;
}

/**
 * Prunes incomplete draft filter nodes (C representation); does not invent backend query shapes.
 */
export function normalizeFilters(filters: ConditionGroup[] = []): ConditionGroup[] {
  /**
   * Normalizes a single condition group recursively.
   */
  function normalizeGroup(g: any): ConditionGroup | null {
    if (!g || typeof g !== 'object') return null;
    const logic: 'And' | 'Or' = g.logic === 'Or' || g.logic === 'OR' ? 'Or' : 'And';
    const rawChildren: any[] = Array.isArray(g.children) ? g.children : [];
    const children: Array<Condition | ConditionGroup> = [];

    for (const ch of rawChildren) {
      if (!ch) continue;
      if ((ch as any).children !== undefined) {
        const sub = normalizeGroup(ch);
        if (sub && sub.children.length > 0) children.push(sub);
      } else if ((ch as any).operator !== undefined || (ch as any).field !== undefined) {
        const c = ch as any;
        if (!c.field || !c.operator) continue;
        children.push({
          id: c.id || genId(),
          field: c.field,
          operator: c.operator,
          value: c.value,
        });
      }
    }

    if (children.length === 0) return null;
    return {
      id: g.id || genId(),
      logic,
      name: g.name,
      children,
    } as ConditionGroup;
  }

  const result: ConditionGroup[] = [];
  for (const f of filters || []) {
    const nf = normalizeGroup(f);
    if (nf) result.push(nf);
  }
  return result;
}
