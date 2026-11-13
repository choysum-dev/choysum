// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Inline helpers under test to avoid backend import dependencies.
function normalizeScopeId(value: unknown): string {
  if (value == null) return '';
  if (typeof value === 'object') return String((value as any)?.Id ?? '').trim();
  return String(value ?? '').trim();
}

function uniqScopeIds(ids: string[]): string[] {
  return Array.from(new Set((ids || []).map(v => normalizeScopeId(v)).filter(Boolean)));
}

function buildAllowedCompanyIds(user: any): string[] {
  return uniqScopeIds([
    normalizeScopeId((user as any)?.CompanyId),
    ...(Array.isArray((user as any)?.CompanyIds) ? (user as any).CompanyIds : []),
  ]);
}

function readStoredEnabledCompanyIds(user: any): string[] {
  const raw = (user as any)?.EnabledCompanyIds;
  if (!Array.isArray(raw)) return [];
  return uniqScopeIds(raw.map(normalizeScopeId).filter(Boolean));
}

function computeTokenCompanyScope(user: any) {
  const allowedCompanyIds = buildAllowedCompanyIds(user);
  const storedActive = normalizeScopeId((user as any)?.ActiveCompanyId);
  const storedEnabled = readStoredEnabledCompanyIds(user);

  let activeCompanyId = '';
  if (storedActive && allowedCompanyIds.includes(storedActive)) {
    activeCompanyId = storedActive;
  } else {
    const fallback = normalizeScopeId((user as any)?.CompanyId);
    if (fallback && allowedCompanyIds.includes(fallback)) activeCompanyId = fallback;
    else if (allowedCompanyIds.length) activeCompanyId = allowedCompanyIds[0];
  }

  let enabledCompanyIds = storedEnabled.filter((cid: string) => allowedCompanyIds.includes(cid));
  if (!enabledCompanyIds.length && activeCompanyId) enabledCompanyIds = [activeCompanyId];
  if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) {
    enabledCompanyIds = [activeCompanyId, ...enabledCompanyIds];
  }
  if (!enabledCompanyIds.length && allowedCompanyIds.length) enabledCompanyIds = [allowedCompanyIds[0]];

  return { allowedCompanyIds, activeCompanyId: activeCompanyId || undefined, enabledCompanyIds };
}

function normalizeRequestedEnabledCompanyIds(enabledCompanyIds: any): string[] | null {
  if (!Array.isArray(enabledCompanyIds)) return null;
  return enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
}

function validateSwitchCompanyScopeInput(user: any, activeCompanyId: string, enabledCompanyIds: any) {
  const active = normalizeScopeId(activeCompanyId);
  const allowed = buildAllowedCompanyIds(user);
  const storedEnabled = readStoredEnabledCompanyIds(user);

  if (!active) {
    return { ok: false, code: 'active_empty', active, enabled: null, allowed };
  }

  let enabled: string[];

  if (enabledCompanyIds === undefined || enabledCompanyIds === null) {
    enabled = storedEnabled.length ? storedEnabled : [active];
  } else if (Array.isArray(enabledCompanyIds)) {
    enabled = enabledCompanyIds.map(normalizeScopeId).filter(Boolean);
  } else {
    return { ok: false, code: 'enabled_type', active, enabled: null, allowed };
  }

  enabled = uniqScopeIds(enabled);

  // Match production (_lifecycle_scope.ts): authorize active before enabled loop.
  if (!allowed.includes(active)) {
    return { ok: false, code: 'active_outside_allowed', active, enabled, allowed };
  }

  for (const cid of enabled) {
    if (!allowed.includes(cid)) {
      return { ok: false, code: 'enabled_unauthorized', active, enabled, allowed, companyId: cid };
    }
  }

  if (!enabled.includes(active)) {
    return { ok: false, code: 'active_not_in_enabled', active, enabled, allowed };
  }

  return { ok: true, active, enabled, allowed };
}

test('buildAllowedCompanyIds: returns deduplicated non-empty ids from CompanyId and CompanyIds', () => {
  const user = { CompanyId: 'C1', CompanyIds: ['C2', 'C3'] };
  expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2', 'C3']);
});

test('buildAllowedCompanyIds: deduplicates when CompanyId appears in CompanyIds', () => {
  const user = { CompanyId: 'C1', CompanyIds: ['C1', 'C2'] };
  expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2']);
});

test('buildAllowedCompanyIds: handles missing CompanyIds', () => {
  const user = { CompanyId: 'C1' };
  expect(buildAllowedCompanyIds(user)).toEqual(['C1']);
});

test('buildAllowedCompanyIds: filters empty and whitespace ids', () => {
  const user = { CompanyId: '  C1  ', CompanyIds: ['', '  ', 'C2'] };
  expect(buildAllowedCompanyIds(user)).toEqual(['C1', 'C2']);
});

test('computeTokenCompanyScope: picks active from stored fields when within allowed scope', () => {
  const user = {
    CompanyId: 'C1',
    CompanyIds: ['C1', 'C2'],
    ActiveCompanyId: 'C2',
    EnabledCompanyIds: ['C2'],
  };
  const scope = computeTokenCompanyScope(user);
  expect(scope.activeCompanyId).toBe('C2');
  expect(scope.enabledCompanyIds).toEqual(['C2']);
});

test('computeTokenCompanyScope: falls back to CompanyId when stored active is outside allowed', () => {
  const user = {
    CompanyId: 'C1',
    CompanyIds: ['C1'],
    ActiveCompanyId: 'C99',
    EnabledCompanyIds: ['C99'],
  };
  const scope = computeTokenCompanyScope(user);
  expect(scope.activeCompanyId).toBe('C1');
});

test('computeTokenCompanyScope: returns empty scope for user with no companies', () => {
  const scope = computeTokenCompanyScope({});
  expect(scope.activeCompanyId).toBeUndefined();
  expect(scope.enabledCompanyIds).toEqual([]);
});

test('normalizeRequestedEnabledCompanyIds: returns null for non-array', () => {
  expect(normalizeRequestedEnabledCompanyIds(undefined)).toBeNull();
  expect(normalizeRequestedEnabledCompanyIds('C1')).toBeNull();
});

test('normalizeRequestedEnabledCompanyIds: returns normalized and filtered array', () => {
  expect(normalizeRequestedEnabledCompanyIds(['  C1  ', '', 'C2'])).toEqual(['C1', 'C2']);
});

const baseUser = { CompanyId: 'C1', CompanyIds: ['C1', 'C2'] };

test('validateSwitchCompanyScopeInput: rejects empty active company id', () => {
  const result = validateSwitchCompanyScopeInput(baseUser, '', undefined);
  expect(result.ok).toBe(false);
  if (!result.ok) expect(result.code).toBe('active_empty');
});

test('validateSwitchCompanyScopeInput: rejects non-array enabled type', () => {
  const result = validateSwitchCompanyScopeInput(baseUser, 'C1', 'not-an-array');
  expect(result.ok).toBe(false);
  if (!result.ok) expect(result.code).toBe('enabled_type');
});

test('validateSwitchCompanyScopeInput: rejects enabled company outside allowed scope', () => {
  const result = validateSwitchCompanyScopeInput(baseUser, 'C1', ['C99']);
  expect(result.ok).toBe(false);
  if (!result.ok) expect(result.code).toBe('enabled_unauthorized');
});

test('validateSwitchCompanyScopeInput: rejects active outside allowed scope', () => {
  const result = validateSwitchCompanyScopeInput(baseUser, 'C99', undefined);
  expect(result.ok).toBe(false);
  if (!result.ok) expect(result.code).toBe('active_outside_allowed');
});

test('validateSwitchCompanyScopeInput: accepts valid active and enabled', () => {
  const result = validateSwitchCompanyScopeInput(baseUser, 'C2', ['C1', 'C2']);
  expect(result.ok).toBe(true);
  if (result.ok) {
    expect(result.active).toBe('C2');
    expect(result.enabled).toEqual(['C1', 'C2']);
  }
});

test('validateSwitchCompanyScopeInput: omit enabled defaults to stored EnabledCompanyIds', () => {
  const user = { ...baseUser, EnabledCompanyIds: ['C2'] };
  const result = validateSwitchCompanyScopeInput(user, 'C2', undefined);
  expect(result.ok).toBe(true);
  if (result.ok) expect(result.enabled).toEqual(['C2']);
});
