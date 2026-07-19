// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';
import type { OperatorOption } from '../../types';
export type { OperatorOption } from '../../types';

const { _t } = createTranslate('web', { scope: 'web/query/utils/filter/operators' });

/**
 * Default operator catalog for filter builders.
 * Labels use literal `_t('…')` msgids so extract can collect them.
 */
function resolveOperatorOptions(): OperatorOption[] {
  return [
    { value: '=', label: _t('Equals') },
    { value: '!=', label: _t('Not equals') },
    { value: 'in', label: _t('In') },
    { value: 'not in', label: _t('Not in') },
    { value: 'like', label: _t('Contains') },
    { value: 'not like', label: _t('Does not contain') },
    { value: '>', label: _t('Greater than') },
    { value: '>=', label: _t('Greater than or equal') },
    { value: '<', label: _t('Less than') },
    { value: '<=', label: _t('Less than or equal') },
    { value: 'is', label: _t('Is') },
    { value: 'is not', label: _t('Is not') },
    { value: 'child_of', label: _t('Descendants (child_of)') },
    { value: 'parent_of', label: _t('Ancestors (parent_of)') },
  ];
}

/** @deprecated Prefer `getOperatorOptions()` so labels stay reactive to locale changes. */
export const OPERATORS: readonly OperatorOption[] = resolveOperatorOptions();

/**
 * Resolves the display label for an operator.
 */
export function getOperatorLabel(op: string): string {
  switch (op) {
    case '=':
      return _t('Equals');
    case '!=':
      return _t('Not equals');
    case 'in':
      return _t('In');
    case 'not in':
      return _t('Not in');
    case 'like':
      return _t('Contains');
    case 'not like':
      return _t('Does not contain');
    case '>':
      return _t('Greater than');
    case '>=':
      return _t('Greater than or equal');
    case '<':
      return _t('Less than');
    case '<=':
      return _t('Less than or equal');
    case 'is':
      return _t('Is');
    case 'is not':
      return _t('Is not');
    case 'child_of':
      return _t('Descendants (child_of)');
    case 'parent_of':
      return _t('Ancestors (parent_of)');
    default:
      return op;
  }
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
  const base = resolveOperatorOptions();
  if ((metaType || '').toLowerCase() === 'manytoone') {
    // Reserved for field-type-specific customization.
  }
  if (isTreeRelation) {
    for (const t of ['child_of', 'parent_of'] as const) {
      if (!base.find(o => o.value === t)) {
        base.push({ value: t, label: getOperatorLabel(t) });
      }
    }
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
