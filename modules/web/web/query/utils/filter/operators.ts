// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { OperatorOption } from '../../types';
export type { OperatorOption } from '../../types';

/**
 * Default operator catalog for filter builders.
 */
export const OPERATORS: readonly OperatorOption[] = [
  { value: '=', label: '等于' },
  { value: '!=', label: '不等于' },
  { value: 'in', label: '包含于' },
  { value: 'not in', label: '不包含于' },
  { value: 'like', label: '包含' },
  { value: 'not like', label: '不包含' },
  { value: '>', label: '大于' },
  { value: '>=', label: '大于等于' },
  { value: '<', label: '小于' },
  { value: '<=', label: '小于等于' },
  { value: 'is', label: '是' },
  { value: 'is not', label: '不是' },
  { value: 'child_of', label: '子孙(child_of)' },
  { value: 'parent_of', label: '祖先(parent_of)' },
] as const;

/**
 * Resolves the display label for an operator.
 */
export function getOperatorLabel(op: string): string {
  const f = OPERATORS.find(o => o.value === op);
  return f ? f.label : op;
}

/**
 * Reports whether an operator compares against null-like values.
 */
export function isNullOperator(op?: string): boolean {
  if (!op) return false;
  return op === 'is' || op === 'is not';
}

/**
 * Reports whether an operator expects an explicit value.
 */
export function requiresValue(op?: string): boolean {
  if (!op) return false;
  return !isNullOperator(op);
}

/**
 * Returns operator options for a field type, optionally enabling tree operators.
 */
export function getOperatorOptions(metaType?: string, isTreeRelation?: boolean): OperatorOption[] {
  const base = [...OPERATORS];
  if ((metaType || '').toLowerCase() === 'manytoone') {
    // Reserved for field-type-specific customization.
  }
  if (isTreeRelation) {
    const tail = ['child_of', 'parent_of'];
    for (const t of tail) if (!base.find(o => o.value === t)) base.push({ value: t, label: t });
  }
  return base;
}

/**
 * Returns the default filter value for a field type.
 */
export function defaultValueFor(metaType?: string): any {
  if ((metaType || '').toLowerCase() === 'boolean') return false;
  return undefined;
}
