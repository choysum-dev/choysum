// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsCtxAndReq } from '@/core/service/api/context';
import { normalizeScopeId, uniqScopeIds } from '@/core/service/utils/normalization';

export { normalizeScopeId, uniqScopeIds };

export type UserCompanyScopeSource = {
  CompanyId?: unknown;
  CompanyIds?: unknown;
  ActiveCompanyId?: unknown;
  EnabledCompanyIds?: unknown;
};

export type TokenCompanyScope = {
  allowedCompanyIds: string[];
  activeCompanyId?: string;
  enabledCompanyIds: string[];
};

export type SwitchScopeValidationErrorCode =
  | 'active_empty'
  | 'enabled_type'
  | 'enabled_unauthorized'
  | 'active_outside_allowed'
  | 'active_not_in_enabled';

export type SwitchScopeValidationResult =
  | {
      ok: true;
      active: string;
      enabled: string[];
      allowed: string[];
    }
  | {
      ok: false;
      code: SwitchScopeValidationErrorCode;
      active: string;
      enabled: string[] | null;
      allowed: string[];
      companyId?: string;
    };

/** Read stored EnabledCompanyIds as normalized id list. */
export function readStoredEnabledCompanyIds(user: UserCompanyScopeSource): string[] {
  const raw = user?.EnabledCompanyIds;
  if (!Array.isArray(raw)) return [];
  return uniqScopeIds(raw.map(normalizeScopeId).filter(Boolean));
}

/**
 * Compute allowed company ids from User.CompanyId and User.CompanyIds.
 */
export function buildAllowedCompanyIds(user: UserCompanyScopeSource): string[] {
  return uniqScopeIds([
    normalizeScopeId(user?.CompanyId),
    ...(Array.isArray(user?.CompanyIds) ? user.CompanyIds : []),
  ]);
}

/**
 * Build token metadata company scope from User work-context fields.
 */
export function computeTokenCompanyScope(user: UserCompanyScopeSource): TokenCompanyScope {
  const allowedCompanyIds = buildAllowedCompanyIds(user);
  const storedActive = normalizeScopeId(user?.ActiveCompanyId);
  const storedEnabled = readStoredEnabledCompanyIds(user);

  let activeCompanyId = '';
  if (storedActive && allowedCompanyIds.includes(storedActive)) {
    activeCompanyId = storedActive;
  } else {
    const fallback = normalizeScopeId(user?.CompanyId);
    if (fallback && allowedCompanyIds.includes(fallback)) activeCompanyId = fallback;
    else if (allowedCompanyIds.length) activeCompanyId = allowedCompanyIds[0];
  }

  let enabledCompanyIds = storedEnabled.filter((cid: string) => allowedCompanyIds.includes(cid));
  if (!enabledCompanyIds.length && activeCompanyId) enabledCompanyIds = [activeCompanyId];
  if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) {
    enabledCompanyIds = [activeCompanyId, ...enabledCompanyIds];
  }
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
export function normalizeRequestedEnabledCompanyIds(enabledCompanyIds: unknown): string[] | null {
  if (!Array.isArray(enabledCompanyIds)) return null;
  return enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
}

/**
 * Validate switch-company inputs against allowed company scope.
 * When EnabledCompanyIds is omitted, defaults to stored EnabledCompanyIds or [active].
 */
export function validateSwitchCompanyScopeInput(
  user: UserCompanyScopeSource,
  activeCompanyId: string,
  enabledCompanyIds: unknown
): SwitchScopeValidationResult {
  const active = normalizeScopeId(activeCompanyId);
  const allowed = buildAllowedCompanyIds(user);
  const storedEnabled = readStoredEnabledCompanyIds(user);

  if (!active) {
    return {
      ok: false,
      code: 'active_empty',
      active,
      enabled: null,
      allowed,
    };
  }

  let enabled: string[];

  if (enabledCompanyIds === undefined || enabledCompanyIds === null) {
    enabled = storedEnabled.length ? storedEnabled : [active];
  } else if (Array.isArray(enabledCompanyIds)) {
    enabled = enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
  } else {
    return {
      ok: false,
      code: 'enabled_type',
      active,
      enabled: null,
      allowed,
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
    };
  }

  return {
    ok: true,
    active,
    enabled,
    allowed,
  };
}

/**
 * Create an audit emitter that enforces at-most-once output for one switch flow.
 */
export function createSwitchCompanyScopeAuditEmitter(eventName: string): {
  emitOnce: (payload: Record<string, unknown>) => void;
  wasEmitted: () => boolean;
} {
  let emitted = false;

  const emit = (payload: Record<string, unknown>): void => {
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
    emitOnce(payload: Record<string, unknown>) {
      if (emitted) return;
      emitted = true;
      emit(payload);
    },
    wasEmitted() {
      return emitted;
    },
  };
}
