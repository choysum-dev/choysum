// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Request context stored as lightweight key-value pairs and encoded into W3C Baggage headers.
 * Used to carry business context such as tenant, locale, lang, and feature flags along the same trace.
 *
 * - `lang`: terminology language code (e.g. zh_CN) — used by `_t` / errors
 * - `locale`: format/region code (e.g. zh-CN) — dates/numbers; ≠ lang
 */
export type RequestContextKV = Record<string, string>;

/**
 * Global default context provider.
 * It may be either a static object or a dynamic function.
 */
let defaultProvider: (() => RequestContextKV) | undefined;

/**
 * Sets the global default request context provider.
 * The resolved context applies to all requests unless a scoped context overrides it.
 *
 * @param provider - Static context object or dynamic context function.
 *
 * @example
 * ```ts
 * // Static context.
 * setGlobalRequestContextProvider({ tenant: 'acme', locale: 'zh-CN' });
 *
 * // Dynamic context.
 * setGlobalRequestContextProvider(() => ({
 *   tenant: getCurrentTenant(),
 *   locale: navigator.language || 'zh-CN',
 * }));
 * ```
 */
export function setGlobalRequestContextProvider(provider: (() => RequestContextKV) | RequestContextKV): void {
  defaultProvider = typeof provider === 'function' ? provider : () => provider;
}

/**
 * Clears the global default context provider.
 */
export function clearGlobalRequestContextProvider(): void {
  defaultProvider = undefined;
}

/**
 * Stack of scoped contexts used by nested context execution.
 * Later frames override keys from earlier frames.
 */
const stack: RequestContextKV[] = [];

/**
 * Returns the effective request context by merging the global default with every scoped frame.
 * Precedence: stack top > ... > stack bottom > global default.
 *
 * @returns The merged context key-value pairs.
 *
 * @example
 * ```ts
 * setGlobalRequestContextProvider({ tenant: 'acme', locale: 'en' });
 *
 * await runWithRequestContext({ locale: 'zh-CN' }, async () => {
 *   const ctx = getCurrentRequestContext();
 *   // { tenant: 'acme', locale: 'zh-CN' }
 * });
 * ```
 */
export function getCurrentRequestContext(): RequestContextKV {
  const merged: RequestContextKV = {};

  // 1. Apply the global default first.
  if (defaultProvider) {
    try {
      const defaultCtx = defaultProvider();
      if (defaultCtx && typeof defaultCtx === 'object') {
        Object.assign(merged, defaultCtx);
      }
    } catch (error) {
      console.warn('[RequestContext] Failed to get default context:', error);
    }
  }

  // 2. Overlay stack entries in order so later frames win.
  for (const ctx of stack) {
    if (!ctx || typeof ctx !== 'object') continue;
    Object.assign(merged, ctx);
  }

  return merged;
}

/**
 * Runs an async function inside a scoped context overlay.
 * The overlay only applies to requests started within fn and is popped automatically afterward.
 *
 * @param ctx - Temporary context key-value pairs.
 * @param fn - Async function to execute.
 * @returns The return value from fn.
 *
 * @example
 * ```ts
 * // Add temporary context for a specific operation.
 * await runWithRequestContext({ reason: 'bulk-import' }, async () => {
 *   await userStore.CreateMany(users);
 *   await logStore.Create({ action: 'import' });
 * });
 *
 * // Nested scopes.
 * await runWithRequestContext({ tenant: 'acme' }, async () => {
 *   await runWithRequestContext({ feature: 'beta' }, async () => {
 *     const ctx = getCurrentRequestContext();
 *     // { tenant: 'acme', feature: 'beta', ...global default }
 *   });
 * });
 * ```
 */
export async function runWithRequestContext<T>(ctx: RequestContextKV, fn: () => Promise<T>): Promise<T> {
  // Push.
  stack.push(ctx || {});
  try {
    return await fn();
  } finally {
    // Pop.
    stack.pop();
  }
}

/**
 * Synchronous variant that runs a function inside a scoped context.
 *
 * @param ctx - Temporary context key-value pairs.
 * @param fn - Synchronous function to execute.
 * @returns The return value from fn.
 *
 * @example
 * ```ts
 * const result = runWithRequestContextSync({ debug: 'true' }, () => {
 *   return computeSomething();
 * });
 * ```
 */
export function runWithRequestContextSync<T>(ctx: RequestContextKV, fn: () => T): T {
  stack.push(ctx || {});
  try {
    return fn();
  } finally {
    stack.pop();
  }
}

/**
 * Returns the current stack depth for debugging.
 *
 * @returns The number of active scoped frames.
 */
export function getContextStackDepth(): number {
  return stack.length;
}

/**
 * Clears the scoped context stack.
 * Intended for tests and debugging only; production code should avoid calling it.
 */
export function clearContextStack(): void {
  stack.length = 0;
}
