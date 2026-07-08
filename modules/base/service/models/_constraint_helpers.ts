// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Write back constraint fields from self to ctx.values.
 *
 * For each field in `fields`, if the field key is present in ctx.values
 * (via hasOwnProperty), copy the value from self to values.
 *
 * Options:
 *  - forceOnCreate: when true and ctx.mode === 'create', write all listed
 *    fields unconditionally (used by currency, sequence).
 *  - triggerFields: when set, write `targetField` to values if ANY of the
 *    triggerFields are present in values (used for compound triggers like
 *    CompanyScopeKey triggered by CompanyId or CompanyScopeKey changes).
 */
export function writeConstraintFields(
  self: Record<string, any>,
  ctx: { values?: Record<string, any>; mode?: string },
  fields: string[],
  opts?: { forceOnCreate?: boolean; triggerFields?: string[]; targetField?: string }
): void {
  const values = (ctx?.values || {}) as Record<string, any>;
  const isCreate = opts?.forceOnCreate && String(ctx?.mode || '') === 'create';

  for (const field of fields) {
    if (isCreate || Object.prototype.hasOwnProperty.call(values, field)) {
      values[field] = (self as any)[field];
    }
  }

  if (opts?.triggerFields && opts?.targetField) {
    const triggered =
      isCreate || opts.triggerFields.some(f => Object.prototype.hasOwnProperty.call(values, f));
    if (triggered) {
      values[opts.targetField] = (self as any)[opts.targetField];
    }
  }
}
