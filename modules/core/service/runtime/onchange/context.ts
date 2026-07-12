// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type {
  MessageLevel,
  OnchangeMessage,
  OnchangeCondition,
  OnchangeValue,
  SelectionCondition,
  MsgBuilder,
  ConditionBuilder,
  SelectionBuilder,
  ValBuilder,
  EmitArg,
  OnchangeContext,
} from './types';
import BaseModel from '../../orm/model/model';
import type { ObjectRecord } from '../../../utils/types';

function asRecord(value: unknown): ObjectRecord | undefined {
  return value !== null && typeof value === 'object' ? (value as ObjectRecord) : undefined;
}

function isMessageLevel(value: unknown): value is MessageLevel {
  return value === 'info' || value === 'warn' || value === 'error';
}

// ============================================================================
// Builder factories
// ============================================================================

/**
 * Create a message builder.
 * Supported signatures:
 * - msg('field', 'error', 'message')
 * - msg('field', 'error', { message: 'xxx', blocking: true, title: 'xxx' })
 * - msg('error', 'message')
 * - msg('error', 'message', { blocking: true, title: 'xxx' })
 */
export function makeMsg<T extends BaseModel>(): MsgBuilder<T> {
  return ((a: unknown, b: unknown, c?: unknown) => {
    // Signature 1: msg('field', 'error', 'message')
    if (typeof a === 'string' && isMessageLevel(b) && typeof c === 'string') {
      return {
        level: b,
        message: c,
        field: a as OnchangeMessage<T>['field'],
      };
    }

    // Signature 2: msg('field', 'error', { message, blocking, title })
    if (typeof a === 'string' && isMessageLevel(b) && c !== undefined && typeof c === 'object') {
      const options = asRecord(c);
      return {
        level: b,
        message: typeof options?.message === 'string' ? options.message : '',
        field: a as OnchangeMessage<T>['field'],
        blocking: typeof options?.blocking === 'boolean' ? options.blocking : undefined,
        title: typeof options?.title === 'string' ? options.title : undefined,
      };
    }

    // Signature 3: msg('error', 'message')
    if (isMessageLevel(a) && typeof b === 'string' && c === undefined) {
      return { level: a, message: b };
    }

    // Signature 4: msg('error', 'message', { blocking, title })
    if (isMessageLevel(a) && typeof b === 'string' && c !== undefined && typeof c === 'object') {
      const options = asRecord(c);
      return {
        level: a,
        message: b,
        blocking: typeof options?.blocking === 'boolean' ? options.blocking : undefined,
        title: typeof options?.title === 'string' ? options.title : undefined,
      };
    }

    throw new Error(`Invalid msg() signature: ${JSON.stringify([a, b, c])}`);
  }) as MsgBuilder<T>;
}

/**
 * Create a condition builder.
 * Signature: cond(field, condition)
 */
export function makeCondition<T extends BaseModel>(): ConditionBuilder<T> {
  return ((field: unknown, condition: unknown) => ({
    field: String(field) as OnchangeCondition<T>['field'],
    condition: condition as OnchangeCondition<T>['condition'],
  })) as ConditionBuilder<T>;
}

/**
 * Create a selection-condition builder.
 *
 * Signature: sel(field, selection, disabled?)
 *
 * @example
 * // Scenario 1: filter only while preserving explicit order.
 * ctx.sel('PaymentMethod', ['transfer', 'credit', 'cash', 'card'])
 *
 * // Scenario 2: filter plus disable.
 * ctx.sel('DiscountType', ['none', 'percentage', 'fixed', 'vip'], ['fixed', 'vip'])
 *
 * // Scenario 3: reorder with urgent options first.
 * ctx.sel('Priority', ['urgent', 'high', 'normal', 'low'])
 */
export function makeSelection(): SelectionBuilder {
  return (field: string, selection: string[], disabled?: string[]) => {
    const condition: SelectionCondition = {
      field,
      selection,
    };

    // Only add the disabled field when disabled items are present.
    if (disabled && disabled.length > 0) {
      condition.disabled = disabled;
    }

    return condition;
  };
}

/**
 * Create a value builder.
 *
 * Signature: val(key, value)
 *
 * @deprecated Prefer direct `this.Field = value` assignment in onchange handlers
 * registered with {@link OnchangeHandlerMeta.signature} set to `'instanceNoArgs'`.
 * `ctx.val` is preserved for backward compatibility with legacy `method(ctx)` handlers.
 */
export function makeVal<T extends BaseModel>(): ValBuilder<T> {
  return ((key: PropertyKey, value: unknown) => ({ [key]: value })) as ValBuilder<T>;
}

// ============================================================================
// Normalization and helpers
// ============================================================================

/**
 * Normalize message arrays.
 *
 * @param input - A message or an array of messages.
 * @returns The normalized message array.
 *
 * @example
 * normalizeMessages('warning message') // [{ level: 'warn', message: 'warning message' }]
 * normalizeMessages([{ level: 'error', message: 'error' }]) // [{ level: 'error', message: 'error' }]
 */
export function normalizeMessages(input: unknown): OnchangeMessage[] {
  if (!input) return [];

  const arr = Array.isArray(input) ? input : [input];
  const out: OnchangeMessage[] = [];

  for (const item of arr) {
    if (!item) continue;

    // Strings default to warn level.
    if (typeof item === 'string') {
      out.push({ level: 'warn', message: item });
      continue;
    }

    const record = asRecord(item);
    if (!record) continue;

    // Objects must provide both message and level.
    if (typeof record.message === 'string' && typeof record.level === 'string') {
      out.push({
        level: record.level as MessageLevel,
        message: record.message,
        field: record.field as OnchangeMessage['field'],
        blocking: typeof record.blocking === 'boolean' ? record.blocking : undefined,
        title: typeof record.title === 'string' ? record.title : undefined,
      });
    }
  }

  return out;
}

/**
 * Normalize condition arrays by standardizing field and condition keys.
 *
 * @param input - A condition or an array of conditions.
 * @returns The normalized condition array.
 */
export function normalizeCondition<T extends BaseModel = BaseModel>(input: unknown): OnchangeCondition<T>[] {
  if (!input) return [];

  const arr = Array.isArray(input) ? input : [input];
  const out: OnchangeCondition<T>[] = [];

  for (const item of arr) {
    if (!item) continue;

    const record = asRecord(item);
    if (!record) continue;

    // Entries must be objects that contain field and condition.
    if (typeof record.field === 'string' && 'condition' in record) {
      out.push({
        field: record.field as OnchangeCondition<T>['field'],
        condition: record.condition as OnchangeCondition<T>['condition'],
      } as OnchangeCondition<T>);
    }
  }

  return out;
}

/**
 * Normalize selection-condition arrays.
 *
 * @param input - A selection condition or an array of selection conditions.
 * @returns The normalized selection-condition array.
 *
 * @remarks
 * Validation rules:
 * - The entry must be an object.
 * - It must include a non-empty string field.
 * - It must include a non-empty selection array.
 * - disabled is optional and must be an array when provided.
 */
export function normalizeSelection(input: unknown): SelectionCondition[] {
  if (!input) return [];

  const arr = Array.isArray(input) ? input : [input];
  const out: SelectionCondition[] = [];

  for (const item of arr) {
    const record = asRecord(item);
    if (!record) continue;

    // field must be a non-empty string.
    if (typeof record.field !== 'string' || !record.field) continue;

    // selection must be a non-empty array.
    if (!Array.isArray(record.selection) || record.selection.length === 0) continue;

    const selection = record.selection.filter(item => typeof item === 'string') as string[];
    if (selection.length === 0) continue;

    const condition: SelectionCondition = {
      field: record.field,
      selection,
    };

    // Optional disabled array.
    if (Array.isArray(record.disabled) && record.disabled.length > 0) {
      condition.disabled = record.disabled.filter(item => typeof item === 'string') as string[];
    }

    out.push(condition);
  }

  return out;
}

/**
 * Apply a patch to an object, supporting dotted paths such as a.b.c.
 *
 * @param target - Target object.
 * @param patch - Patch object to apply.
 *
 * @example
 * applyValuePatch(obj, { 'Lines.0.Quantity': 10 })
 * // obj.Lines[0].Quantity = 10
 */
export function applyValuePatch(target: ObjectRecord, patch: ObjectRecord) {
  const setByPath = (obj: ObjectRecord, path: string, value: unknown) => {
    const segs = path.split('.').filter(Boolean);
    if (!segs.length) return;

    let current = obj;
    for (let i = 0; i < segs.length - 1; i++) {
      const segment = segs[i];
      const next = asRecord(current[segment]);
      if (!next) {
        const created: ObjectRecord = {};
        current[segment] = created;
        current = created;
      } else {
        current = next;
      }
    }

    current[segs[segs.length - 1]] = value;
  };

  for (const [key, value] of Object.entries(patch)) {
    setByPath(target, key, value);
  }
}

// ============================================================================
// Context creation
// ============================================================================

/**
 * Create a type-safe Onchange context.
 *
 * @remarks
 * **Migration note**: `ctx.val` is a legacy helper for emitting value patches
 * via `ctx.emit(ctx.val('Field', value))`. When using the recommended
 * `instanceNoArgs` handler signature, prefer direct `this.Field = value` and
 * `return { messages, condition, selection }` instead.
 *
 * @param params - Context parameters.
 * @param params.draft - Mutable draft record.
 * @param params.changed - Read-only set of changed fields.
 * @param params.pushMessages - Message dispatch callback.
 * @param params.pushCondition - Condition dispatch callback.
 * @param params.pushSelection - Selection-condition dispatch callback.
 * @param params.applyValue - Value-application callback.
 * @returns The Onchange context object.
 */
export function createOnchangeContext<T extends BaseModel>(params: {
  draft: T;
  changed: Set<string>;
  pushMessages: (m: OnchangeMessage<T>[]) => void;
  pushCondition: (q: OnchangeCondition<T>[]) => void;
  pushSelection: (s: SelectionCondition[]) => void;
  applyValue: (v: OnchangeValue<T>) => void;
}): OnchangeContext<T> {
  const msg = makeMsg<T>();
  const cond = makeCondition<T>();
  const sel = makeSelection();
  const val = makeVal<T>();

  /**
   * Unified emit dispatcher.
   *
   * Supported input types:
   * - OnchangeMessage<T>
   * - OnchangeCondition<T>
   * - SelectionCondition
   * - OnchangeValue<T>
   * - Arrays of the types above
   */
  const emit = (payload: EmitArg<T>) => {
    const arr = Array.isArray(payload) ? payload : [payload];

    const ms: OnchangeMessage<T>[] = [];
    const qs: OnchangeCondition<T>[] = [];
    const sels: SelectionCondition[] = [];
    let vp: ObjectRecord | null = null;

    const isMessagePayload = (value: unknown): value is OnchangeMessage<T> => {
      const record = asRecord(value);
      return !!record && typeof record.level === 'string' && typeof record.message === 'string';
    };

    const isSelectionPayload = (value: unknown): value is SelectionCondition => {
      const record = asRecord(value);
      return !!record && typeof record.field === 'string' && Array.isArray(record.selection);
    };

    const isConditionPayload = (value: unknown): value is OnchangeCondition<T> => {
      const record = asRecord(value);
      return !!record && typeof record.field === 'string' && 'condition' in record && !('selection' in record);
    };

    for (const it of arr) {
      if (!it) continue;

      // Message payloads expose level and message.
      if (isMessagePayload(it)) {
        ms.push(it);
        continue;
      }

      // Condition payloads expose field and condition.
      if (isConditionPayload(it)) {
        qs.push(it);
        continue;
      }

      // Selection payloads expose field and selection.
      if (isSelectionPayload(it)) {
        sels.push(it);
        continue;
      }

      // Remaining object payloads are treated as value patches.
      const record = asRecord(it);
      if (record) {
        vp = Object.assign(vp ?? {}, record);
      }
    }

    // Dispatch in batches to preserve atomic behavior.
    if (vp) params.applyValue(vp as OnchangeValue<T>);
    if (ms.length) params.pushMessages(ms);
    if (qs.length) params.pushCondition(qs);
    if (sels.length) params.pushSelection(sels);
  };

  return {
    msg,
    cond,
    sel,
    val,
    draft: params.draft,
    changed: params.changed,
    emit,
  };
}
