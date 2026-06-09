// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxRoot, type Context } from './source';
import { asObjectRecord } from '../../../utils/object';

// Symbol keys for request-scoped overrides and frozen cache values.
const CTX_OVERRIDE_KEY = Symbol.for('choysum.ctx.override');
const CTX_FROZEN_KEY = Symbol.for('choysum.ctx.frozen');

// Process-level override stack used only when no request context is available, such as scripts or background tasks.
const processLevelCtxStack: Context[] = [];

type ContextCarrier = Record<PropertyKey, unknown> & { ctx?: Record<string, unknown> };

function asContextCarrier(value: unknown): ContextCarrier | undefined {
  const record = asObjectRecord(value);
  return record ? (record as ContextCarrier) : undefined;
}

// Deep-freeze iteratively to avoid recursive stack overflows.
function deepFreeze<T>(obj: T): T {
  if (!obj || typeof obj !== 'object' || Object.isFrozen(obj)) return obj;
  const stack: object[] = [obj as object];
  while (stack.length) {
    const cur = stack.pop();
    if (!cur || typeof cur !== 'object' || Object.isFrozen(cur)) continue;
    Object.freeze(cur);
    const curRecord = asObjectRecord(cur);
    if (!curRecord) continue;
    for (const key of Object.keys(curRecord)) {
      const val = curRecord[key];
      if (val && typeof val === 'object' && !Object.isFrozen(val)) {
        stack.push(val);
      }
    }
  }
  return obj;
}

/**
 * Returns the effective readonly business context for the current execution scope.
 */
export function getReadonlyCtx(): Context {
  const jsCtx = asContextCarrier(getJsCtxRoot());

  if (jsCtx?.[CTX_OVERRIDE_KEY]) {
    return jsCtx[CTX_OVERRIDE_KEY] as Context;
  }

  if (jsCtx) {
    let frozen = jsCtx[CTX_FROZEN_KEY] as Context | undefined;
    if (!frozen) {
      const base = asObjectRecord(jsCtx.ctx) || {};
      frozen = deepFreeze({ ...(base || {}) }) as Context;
      jsCtx[CTX_FROZEN_KEY] = frozen;
    }
    return frozen;
  }

  if (processLevelCtxStack.length > 0) {
    return processLevelCtxStack[processLevelCtxStack.length - 1];
  }

  return deepFreeze({} as Record<string, unknown>) as Context;
}

/**
 * Reads a single value from the effective readonly context.
 */
export function getCtxValue<T = unknown>(key: string): T | undefined {
  const ctx = getReadonlyCtx() as Record<string, unknown>;
  return ctx ? (ctx[key] as T) : undefined;
}

function normalizeContextString(value: unknown): string | undefined {
  const normalized = typeof value === 'string' ? value.trim() : String(value ?? '').trim();
  return normalized || undefined;
}

function normalizeContextStringArray(value: unknown): string[] {
  const values = Array.isArray(value) ? value : value ? [value] : [];
  return Array.from(new Set(values.map(item => normalizeContextString(item)).filter((item): item is string => Boolean(item))));
}

/**
 * Returns the active company Id from context.
 */
export function getActiveCompanyId(): string | undefined {
  return normalizeContextString(getCtxValue('activeCompanyId') ?? getCtxValue('ActiveCompanyId'));
}

/**
 * Returns the enabled company Id set from context.
 */
export function getEnabledCompanyIds(): string[] {
  return normalizeContextStringArray(
    getCtxValue('enabledCompanyIds') ?? getCtxValue('EnabledCompanyIds') ?? getCtxValue('activeCompanyId') ?? getCtxValue('ActiveCompanyId')
  );
}

/**
 * Returns the current language from context.
 */
export function getContextLang(): string | undefined {
  return normalizeContextString(getCtxValue('lang') ?? getCtxValue('language') ?? getCtxValue('Language'));
}

/**
 * Returns the current timezone from context.
 */
export function getContextTimezone(): string | undefined {
  return normalizeContextString(getCtxValue('tz') ?? getCtxValue('timezone') ?? getCtxValue('Timezone'));
}

/**
 * Runs a function with a temporary business-context override.
 */
export function withContext<R>(ctx: Partial<Context> | (() => Partial<Context>), fn: () => R, opts?: { merge?: boolean }): R {
  const jsCtx = asContextCarrier(getJsCtxRoot());
  const base = getReadonlyCtx() as Record<string, unknown>;
  const source = typeof ctx === 'function' ? ctx() || {} : ctx || {};
  const sourceRecord = source as Record<string, unknown>;

  const merged = (opts?.merge ?? true) ? { ...base, ...sourceRecord } : { ...sourceRecord };
  const frozen = deepFreeze({ ...merged }) as Context;

  if (jsCtx) {
    const prev = jsCtx[CTX_OVERRIDE_KEY] as Context | undefined;
    jsCtx[CTX_OVERRIDE_KEY] = frozen;

    try {
      const result = fn();
      if (result instanceof Promise) {
        return result.finally(() => {
          if (prev) jsCtx[CTX_OVERRIDE_KEY] = prev;
          else delete jsCtx[CTX_OVERRIDE_KEY];
        }) as unknown as R;
      }
      if (prev) jsCtx[CTX_OVERRIDE_KEY] = prev;
      else delete jsCtx[CTX_OVERRIDE_KEY];
      return result;
    } catch (error) {
      if (prev) jsCtx[CTX_OVERRIDE_KEY] = prev;
      else delete jsCtx[CTX_OVERRIDE_KEY];
      throw error;
    }
  }

  processLevelCtxStack.push(frozen);
  try {
    const result = fn();
    if (result instanceof Promise) {
      return result.finally(() => {
        processLevelCtxStack.pop();
      }) as unknown as R;
    }
    processLevelCtxStack.pop();
    return result;
  } catch (error) {
    processLevelCtxStack.pop();
    throw error;
  }
}
