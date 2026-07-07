// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxAndReq } from '@/core/service/api/context';
import { maybeId } from './_user_authz_shared';

export type TokenCompanyScope = {
  allowedCompanyIds: string[];
  activeCompanyId?: string;
  enabledCompanyIds: string[];
};

export type SwitchScopeValidationErrorCode = 'active_empty' | 'enabled_type' | 'enabled_unauthorized' | 'active_outside_allowed' | 'active_not_in_enabled';

export type SwitchScopeValidationResult =
  | {
      ok: true;
      active: string;
      enabled: string[];
      allowed: string[];
      prefs: Record<string, any>;
    }
  | {
      ok: false;
      code: SwitchScopeValidationErrorCode;
      active: string;
      enabled: string[] | null;
      allowed: string[];
      prefs: Record<string, any>;
      companyId?: string;
    };

/**
 * Normalize a company id-like value to a trimmed string.
 */
export function normalizeScopeId(value: any): string {
  const id = maybeId(value);
  if (id) return String(id).trim();
  return String(value ?? '').trim();
}

/**
 * Return a stable unique copy of ids while preserving the first-seen order.
 */
export function uniqScopeIds(ids: string[]): string[] {
  return Array.from(new Set((ids || []).map(v => normalizeScopeId(v)).filter(Boolean)));
}

/**
 * Normalize a dynamic preferences value into a plain JSON object.
 */
export function normalizeScopePreferences(value: any): Record<string, any> {
  if (!value) return {};

  if (typeof value === 'string') {
    const s = value.trim();
    if (!s) return {};
    try {
      const parsed = JSON.parse(s);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
    } catch {
      return {};
    }
  }

  if (typeof value === 'object') {
    try {
      const snapshot = JSON.parse(JSON.stringify(value));
      if (snapshot && typeof snapshot === 'object' && !Array.isArray(snapshot)) return snapshot;
    } catch {
      // fallthrough
    }
    return value;
  }

  return {};
}

/**
 * Compute allowed company ids from User.CompanyId and User.CompanyIds.
 */
export function buildAllowedCompanyIds(user: any): string[] {
  return uniqScopeIds([normalizeScopeId((user as any)?.CompanyId), ...(Array.isArray((user as any)?.CompanyIds) ? (user as any).CompanyIds : [])]);
}

/**
 * Build token metadata company scope from user state and preferences.
 */
export function computeTokenCompanyScope(user: any): TokenCompanyScope {
  const allowedCompanyIds = buildAllowedCompanyIds(user);
  const prefs: any = normalizeScopePreferences((user as any)?.Preferences);
  const prefActive = typeof prefs?.activeCompanyId === 'string' ? normalizeScopeId(prefs.activeCompanyId) : '';
  const prefEnabled = Array.isArray(prefs?.enabledCompanyIds) ? prefs.enabledCompanyIds.map(normalizeScopeId).filter(Boolean) : [];

  let activeCompanyId = '';
  if (prefActive && allowedCompanyIds.includes(prefActive)) {
    activeCompanyId = prefActive;
  } else {
    const fallback = normalizeScopeId((user as any)?.CompanyId);
    if (fallback && allowedCompanyIds.includes(fallback)) activeCompanyId = fallback;
    else if (allowedCompanyIds.length) activeCompanyId = allowedCompanyIds[0];
  }

  let enabledCompanyIds = prefEnabled.filter((cid: string) => allowedCompanyIds.includes(cid));
  if (!enabledCompanyIds.length && activeCompanyId) enabledCompanyIds = [activeCompanyId];
  if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) enabledCompanyIds = [activeCompanyId, ...enabledCompanyIds];
  if (!enabledCompanyIds.length && allowedCompanyIds.length) enabledCompanyIds = [allowedCompanyIds[0]];

  return {
    allowedCompanyIds,
    activeCompanyId: activeCompanyId || undefined,
    enabledCompanyIds,
  };
}

/**
 * Normalize the optional enabled-company request payload for audit fallback paths.
 */
export function normalizeRequestedEnabledCompanyIds(enabledCompanyIds: any): string[] | null {
  if (!Array.isArray(enabledCompanyIds)) return null;
  return enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
}

/**
 * Validate switch-company inputs against allowed company scope.
 */
export function validateSwitchCompanyScopeInput(user: any, activeCompanyId: string, enabledCompanyIds: any): SwitchScopeValidationResult {
  const active = normalizeScopeId(activeCompanyId);
  const prefs = normalizeScopePreferences((user as any)?.Preferences);
  const allowed = buildAllowedCompanyIds(user);

  if (!active) {
    return {
      ok: false,
      code: 'active_empty',
      active,
      enabled: null,
      allowed,
      prefs,
    };
  }

  const prefEnabled = Array.isArray(prefs?.enabledCompanyIds) ? prefs.enabledCompanyIds.map(normalizeScopeId).filter(Boolean) : [];
  let enabled: string[];

  if (enabledCompanyIds === undefined || enabledCompanyIds === null) {
    enabled = prefEnabled.length ? prefEnabled : [active];
  } else if (Array.isArray(enabledCompanyIds)) {
    enabled = enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
  } else {
    return {
      ok: false,
      code: 'enabled_type',
      active,
      enabled: null,
      allowed,
      prefs,
    };
  }

  enabled = uniqScopeIds(enabled);

  for (const cid of enabled) {
    if (!allowed.includes(cid)) {
      return {
        ok: false,
        code: 'enabled_unauthorized',
        active,
        enabled,
        allowed,
        prefs,
        companyId: cid,
      };
    }
  }

  if (!allowed.includes(active)) {
    return {
      ok: false,
      code: 'active_outside_allowed',
      active,
      enabled,
      allowed,
      prefs,
    };
  }

  if (!enabled.includes(active)) {
    return {
      ok: false,
      code: 'active_not_in_enabled',
      active,
      enabled,
      allowed,
      prefs,
    };
  }

  return {
    ok: true,
    active,
    enabled,
    allowed,
    prefs,
  };
}

/**
 * Merge company scope preferences while preserving unrelated preference fields.
 */
export function buildScopePreferences(basePrefs: Record<string, any>, active: string, enabled: string[]): Record<string, any> {
  const base = basePrefs && typeof basePrefs === 'object' && !Array.isArray(basePrefs) ? basePrefs : {};
  return {
    ...base,
    activeCompanyId: active,
    enabledCompanyIds: enabled,
  };
}

/**
 * Create an audit emitter that enforces at-most-once output for one switch flow.
 */
export function createSwitchCompanyScopeAuditEmitter(eventName: string): {
  emitOnce: (payload: Record<string, any>) => void;
  wasEmitted: () => boolean;
} {
  let emitted = false;

  const emit = (payload: Record<string, any>): void => {
    try {
      const { req } = getJsCtxAndReq();
      const traceId = typeof req?.traceId === 'string' ? req.traceId : '';
      const out = {
        event: eventName,
        traceId,
        ...payload,
      };
      if (payload?.ok === false) {
        console.warn(`[AUDIT] ${JSON.stringify(out)}`);
      } else {
        console.info(`[AUDIT] ${JSON.stringify(out)}`);
      }
    } catch {
      try {
        if (payload?.ok === false) {
          console.warn(`[AUDIT] ${eventName}`);
        } else {
          console.info(`[AUDIT] ${eventName}`);
        }
      } catch {
        // ignore
      }
    }
  };

  return {
    emitOnce(payload: Record<string, any>) {
      if (emitted) return;
      emitted = true;
      emit(payload);
    },
    wasEmitted() {
      return emitted;
    },
  };
}
