// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, test, expect } from 'vitest';

// Inline helpers under test to avoid backend import dependencies.
function normalizeScopeId(value: unknown): string {
  if (value == null) return '';
  if (typeof value === 'object')
    return String((value as any)?.Id ?? (value as any)?.id ?? '').trim();
  return String(value ?? '').trim();
}

function uniqScopeIds(ids: string[]): string[] {
  return Array.from(new Set((ids || []).map(v => normalizeScopeId(v)).filter(Boolean)));
}

function buildAllowedCompanyIds(user: any): string[] {
  return uniqScopeIds([normalizeScopeId((user as any)?.CompanyId), ...(Array.isArray((user as any)?.CompanyIds) ? (user as any).CompanyIds : [])]);
}

function normalizeScopePreferences(value: unknown): Record<string, unknown> {
  if (!value) return {};
  if (typeof value === 'string') {
    const s = value.trim();
    if (!s) return {};
    try {
      const parsed = JSON.parse(s);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
    } catch {
      return {};
    }
  }
  if (typeof value === 'object') {
    try {
      const snapshot = JSON.parse(JSON.stringify(value));
      if (snapshot && typeof snapshot === 'object' && !Array.isArray(snapshot)) return snapshot as Record<string, unknown>;
    } catch {}
    return value as Record<string, unknown>;
  }
  return {};
}

function computeTokenCompanyScope(user: any) {
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

  return { allowedCompanyIds, activeCompanyId: activeCompanyId || undefined, enabledCompanyIds };
}

function normalizeRequestedEnabledCompanyIds(enabledCompanyIds: any): string[] | null {
  if (!Array.isArray(enabledCompanyIds)) return null;
  return enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
}

function validateSwitchCompanyScopeInput(user: any, activeCompanyId: string, enabledCompanyIds: any) {
  const active = normalizeScopeId(activeCompanyId);
  const prefs = normalizeScopePreferences((user as any)?.Preferences);
  const allowed = buildAllowedCompanyIds(user);

  if (!active) {
    return { ok: false, code: 'active_empty', active, enabled: null, allowed, prefs };
  }

  const prefEnabled = Array.isArray(prefs?.enabledCompanyIds) ? prefs.enabledCompanyIds.map(normalizeScopeId).filter(Boolean) : [];
  let enabled: string[];

  if (enabledCompanyIds === undefined || enabledCompanyIds === null) {
    enabled = prefEnabled.length ? prefEnabled : [active];
  } else if (Array.isArray(enabledCompanyIds)) {
    enabled = enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
  } else {
    return { ok: false, code: 'enabled_type', active, enabled: null, allowed, prefs };
  }

  enabled = uniqScopeIds(enabled);

  for (const cid of enabled) {
    if (!allowed.includes(cid)) {
      return { ok: false, code: 'enabled_unauthorized', active, enabled, allowed, prefs, companyId: cid };
    }
  }

  if (!allowed.includes(active)) {
    return { ok: false, code: 'active_outside_allowed', active, enabled, allowed, prefs };
  }

  if (!enabled.includes(active)) {
    return { ok: false, code: 'active_not_in_enabled', active, enabled, allowed, prefs };
  }

  return { ok: true, active, enabled, allowed, prefs };
}

describe('buildAllowedCompanyIds', () => {
  test('returns deduplicated non-empty ids from CompanyId and CompanyIds', () => {
    const user = { CompanyId: 'C1', CompanyIds: ['C2', 'C3'] };
    expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2', 'C3']);
  });

  test('deduplicates when CompanyId appears in CompanyIds', () => {
    const user = { CompanyId: 'C1', CompanyIds: ['C1', 'C2'] };
    expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2']);
  });

  test('handles missing CompanyIds', () => {
    const user = { CompanyId: 'C1' };
    expect(buildAllowedCompanyIds(user)).toEqual(['C1']);
  });

  test('filters empty and whitespace ids', () => {
    const user = { CompanyId: '  C1  ', CompanyIds: ['', '  ', 'C2'] };
    expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2']);
  });
});

describe('computeTokenCompanyScope', () => {
  test('picks active from preferences when within allowed scope', () => {
    const user = { CompanyId: 'C1', CompanyIds: ['C1', 'C2'], Preferences: { activeCompanyId: 'C2', enabledCompanyIds: ['C2'] } };
    const scope = computeTokenCompanyScope(user);
    expect(scope.activeCompanyId).toBe('C2');
    expect(scope.enabledCompanyIds).toEqual(['C2']);
  });

  test('falls back to CompanyId when pref active is outside allowed', () => {
    const user = { CompanyId: 'C1', CompanyIds: ['C1'], Preferences: { activeCompanyId: 'C99', enabledCompanyIds: ['C99'] } };
    const scope = computeTokenCompanyScope(user);
    expect(scope.activeCompanyId).toBe('C1');
  });

  test('returns empty scope for user with no companies', () => {
    const scope = computeTokenCompanyScope({});
    expect(scope.activeCompanyId).toBeUndefined();
    expect(scope.enabledCompanyIds).toEqual([]);
  });
});

describe('normalizeRequestedEnabledCompanyIds', () => {
  test('returns null for non-array', () => {
    expect(normalizeRequestedEnabledCompanyIds(undefined)).toBeNull();
    expect(normalizeRequestedEnabledCompanyIds('C1')).toBeNull();
  });

  test('returns normalized and filtered array', () => {
    expect(normalizeRequestedEnabledCompanyIds(['  C1  ', '', 'C2'])).toEqual(['C1', 'C2']);
  });
});

describe('validateSwitchCompanyScopeInput', () => {
  const baseUser = { CompanyId: 'C1', CompanyIds: ['C1', 'C2'] };

  test('rejects empty active company id', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, '', undefined);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.code).toBe('active_empty');
  });

  test('rejects non-array enabled type', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C1', 'not-an-array');
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.code).toBe('enabled_type');
  });

  test('rejects enabled company outside allowed scope', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C1', ['C99']);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.code).toBe('enabled_unauthorized');
  });

  test('rejects active outside allowed scope (caught by enabled check when enabled defaults to active)', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C99', undefined);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.code).toBe('enabled_unauthorized');
  });

  test('rejects active not in enabled', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C2', ['C1']);
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.code).toBe('active_not_in_enabled');
  });

  test('accepts valid switch with explicit enabled', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C1', ['C1', 'C2']);
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.enabled).toEqual(['C1', 'C2']);
  });

  test('accepts valid switch with omitted enabled', () => {
    const result = validateSwitchCompanyScopeInput(baseUser, 'C2', undefined);
    expect(result.ok).toBe(true);
    if (result.ok) expect(result.enabled).toEqual(['C2']);
  });
});
