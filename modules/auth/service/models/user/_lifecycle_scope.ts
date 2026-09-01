// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxAndReq } from '@/core/service/api/context';
import { normalizeScopeId, uniqScopeIds, normalizePreferences as normalizeScopePreferences, buildScopePreferences } from '@/core/service/utils/normalization';

// Re-export core utilities for backward compat — other auth helpers import these from this module.
export { normalizeScopeId, uniqScopeIds, normalizeScopePreferences, buildScopePreferences };

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
    enabledCompanyIds: uniqScopeIds(enabledCompanyIds),
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
        ...payload,
        event: eventName,
        traceId,
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
