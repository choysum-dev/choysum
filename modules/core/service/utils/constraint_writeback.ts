// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ObjectRecord } from '../../utils/types';

export type ConstraintWritebackOptions = {
  forceOnCreate?: boolean;
  triggerFields?: string[];
  targetField?: string;
};

/**
 * Writes selected fields from constraint self object back to ctx.values.
 *
 * Fields are written when either:
 * - ctx.mode is create and forceOnCreate is enabled, or
 * - the field exists on ctx.values (hasOwnProperty).
 *
 * When triggerFields and targetField are set, targetField is written when
 * any trigger field exists on ctx.values, or forceOnCreate is active.
 */
export function writeConstraintFields(
  self: ObjectRecord,
  ctx: { values?: ObjectRecord; mode?: string },
  fields: string[],
  opts?: ConstraintWritebackOptions
): void {
  const values = (ctx?.values || {}) as ObjectRecord;
  const isCreate = opts?.forceOnCreate && String(ctx?.mode || '') === 'create';

  for (const field of fields) {
    if (isCreate || Object.prototype.hasOwnProperty.call(values, field)) {
      values[field] = self[field];
    }
  }

  if (opts?.triggerFields && opts?.targetField) {
    const triggered = isCreate || opts.triggerFields.some(field => Object.prototype.hasOwnProperty.call(values, field));
    if (triggered) {
      values[opts.targetField] = self[opts.targetField];
    }
  }
}
