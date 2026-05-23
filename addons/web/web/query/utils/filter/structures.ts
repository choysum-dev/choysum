// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Condition, ConditionGroup, NamedFilter } from '@/web/web/query/types';

let _id = 0;

/**
 * Generates a stable local identifier for filter nodes.
 */
function genId() {
  return `f_${Date.now().toString(36)}_${(++_id).toString(36)}`;
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
 * Normalizes named filters or groups into a ConditionGroup array.
 */
export function toFilters(input?: NamedFilter | NamedFilter[] | ConditionGroup | ConditionGroup[] | null): ConditionGroup[] {
  if (!input) return [];
  const arr = Array.isArray(input) ? input : [input];
  const out: ConditionGroup[] = [];
  for (const it of arr) {
    if (!it) continue;
    if ((it as any).query) {
      const nf = it as NamedFilter;
      if (!nf.name) continue;
      const q = nf.query as any;
      if (Array.isArray(q) && q.length === 3) {
        out.push({ id: genId(), logic: 'And', name: nf.name, children: [{ id: genId(), field: q[0], operator: q[1], value: q[2] }] } as any);
      }
    } else if (isGroup(it as any)) out.push(it as ConditionGroup);
  }
  return out;
}

/**
 * Cleans filter groups by discarding invalid nodes and empty groups.
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
