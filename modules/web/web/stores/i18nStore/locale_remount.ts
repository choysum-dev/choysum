// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Post-locale-change strategy (D9, frozen in S6).
 *
 * Supported production path: location.reload.
 * Soft remount stays experimental behind:
 * - localStorage `choysum.web.i18n.remountMode=remount`
 * - or `globalThis.__CHOYSUM_I18N_REMOUNT_MODE__ = 'remount'`
 *
 * Soft remount bounds (S6):
 * - Clears menu/global scoped model stores only.
 * - Does not guarantee RequestContext.lang / keep-alive view consistency.
 * - Without an explicit remount hook, falls back to reload (never a silent no-op).
 */

export type LocaleRemountMode = 'reload' | 'remount';

const FLAG_KEY = 'choysum.web.i18n.remountMode';

/** Resolve remount mode from localStorage / global flag (default reload). */
export function resolveLocaleRemountMode(): LocaleRemountMode {
  if (typeof globalThis === 'undefined') {
    return 'reload';
  }
  const g = globalThis as { localStorage?: Storage; __CHOYSUM_I18N_REMOUNT_MODE__?: string };
  const fromGlobal = String(g.__CHOYSUM_I18N_REMOUNT_MODE__ || '').trim().toLowerCase();
  if (fromGlobal === 'remount' || fromGlobal === 'reload') {
    return fromGlobal;
  }
  try {
    const fromStorage = String(g.localStorage?.getItem(FLAG_KEY) || '')
      .trim()
      .toLowerCase();
    if (fromStorage === 'remount' || fromStorage === 'reload') {
      return fromStorage;
    }
  } catch {
    // ignore storage access errors
  }
  return 'reload';
}

export type AfterLocaleChangeOptions = {
  mode?: LocaleRemountMode;
  /** Injected location.reload for tests. */
  reload?: () => void;
  /** Soft remount hook when mode=remount (clears scoped stores). */
  remount?: () => void | Promise<void>;
};

/**
 * Run after a successful setUiKey.
 * Default strategy is location.reload (D9 / S6 freeze).
 */
export async function afterLocaleChange(options?: AfterLocaleChangeOptions): Promise<void> {
  const mode = options?.mode || resolveLocaleRemountMode();
  if (mode === 'remount') {
    if (options?.remount) {
      await options.remount();
      return;
    }
    // Experimental remount without a hook must not silently no-op.
  }
  const reload =
    options?.reload ||
    (() => {
      if (typeof globalThis !== 'undefined' && (globalThis as { location?: Location }).location?.reload) {
        (globalThis as { location: Location }).location.reload();
      }
    });
  reload();
}

/**
 * Experimental soft remount: destroy menu/global scoped stores.
 * Callers that need full RPC/metadata consistency should keep using reload.
 */
export async function softLocaleRemount(): Promise<void> {
  try {
    const { useScopeManager } = await import('@/web/web/stores/storeScopeManager');
    const { menuScopeManager, globalScopeManager } = useScopeManager();
    menuScopeManager.destroyAll();
    globalScopeManager.destroyAll();
  } catch {
    // Best-effort; missing managers should not block locale change.
  }
}
