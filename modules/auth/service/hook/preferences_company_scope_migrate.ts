// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

type ChoysumDb = {
  execute?: (sql: string, params: string) => Promise<unknown>;
  query?: (sql: string, params: string) => Promise<string>;
};

type AuthUserScopeRow = {
  id?: string;
  Id?: string;
  preferences?: unknown;
  Preferences?: unknown;
  active_company_id?: unknown;
  ActiveCompanyId?: unknown;
  enabled_company_ids?: unknown;
  EnabledCompanyIds?: unknown;
  company_id?: unknown;
  CompanyId?: unknown;
  company_ids?: unknown;
  CompanyIds?: unknown;
};

function resolveDb(): ChoysumDb | undefined {
  return (globalThis as { $choysum?: { db?: ChoysumDb } }).$choysum?.db;
}

function rowId(row: AuthUserScopeRow): string {
  return String(row.id ?? row.Id ?? '').trim();
}

function rowPreferences(row: AuthUserScopeRow): unknown {
  return row.preferences ?? row.Preferences;
}

function rowActiveCompanyId(row: AuthUserScopeRow): string {
  return String(row.active_company_id ?? row.ActiveCompanyId ?? '').trim();
}

function rowEnabledCompanyIds(row: AuthUserScopeRow): unknown {
  return row.enabled_company_ids ?? row.EnabledCompanyIds;
}

function parseIdList(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.map(id => String(id ?? '').trim()).filter(Boolean);
  }
  if (typeof raw === 'string') {
    const text = raw.trim();
    if (!text || text === '[]' || text === 'null') return [];
    try {
      const parsed = JSON.parse(text);
      if (Array.isArray(parsed)) {
        return parsed.map(id => String(id ?? '').trim()).filter(Boolean);
      }
    } catch {
      // fall through
    }
    return text ? [text] : [];
  }
  return [];
}

/** Allowed company set from primary CompanyId ∪ CompanyIds JSON column. */
function buildAllowedFromRow(row: AuthUserScopeRow): string[] {
  const primary = String(row.company_id ?? row.CompanyId ?? '').trim();
  const secondary = parseIdList(row.company_ids ?? row.CompanyIds);
  const out: string[] = [];
  const seen = new Set<string>();
  for (const id of [primary, ...secondary]) {
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

function parsePreferencesBag(raw: unknown): { activeCompanyId?: string; enabledCompanyIds?: string[] } | null {
  if (raw == null) return null;
  let value: unknown = raw;
  if (typeof raw === 'string') {
    const text = raw.trim();
    if (!text || text === '{}' || text === 'null') return null;
    try {
      value = JSON.parse(text);
    } catch {
      return null;
    }
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
  const bag = value as Record<string, unknown>;
  const activeCompanyId = String(bag.activeCompanyId ?? '').trim() || undefined;
  const enabledRaw = bag.enabledCompanyIds;
  const enabledCompanyIds = Array.isArray(enabledRaw)
    ? enabledRaw.map(id => String(id ?? '').trim()).filter(Boolean)
    : undefined;
  if (!activeCompanyId && (!enabledCompanyIds || !enabledCompanyIds.length)) return null;
  return { activeCompanyId, enabledCompanyIds };
}

function hasStoredEnabled(raw: unknown): boolean {
  return parseIdList(raw).length > 0;
}

/**
 * Copy Preferences.activeCompanyId / enabledCompanyIds into first-class columns
 * when those columns are still empty. Values outside CompanyId ∪ CompanyIds are
 * dropped so a stale preference cannot seed an unauthorized active company.
 * No-ops when the leftover Preferences column is already gone.
 */
export async function migratePreferencesCompanyScopeToColumns(): Promise<void> {
  const db = resolveDb();
  if (!db?.query || !db?.execute) return;

  let rows: AuthUserScopeRow[] = [];
  try {
    const raw = await db.query(
      'SELECT id, preferences, active_company_id, enabled_company_ids, company_id, company_ids FROM auth_user WHERE preferences IS NOT NULL',
      '[]',
    );
    const parsed = JSON.parse(String(raw || '[]'));
    rows = Array.isArray(parsed) ? parsed : [];
  } catch (err) {
    // Preferences column already dropped, or table unavailable on this dialect.
    console.warn('[auth] preferences company-scope migrate skipped', err);
    return;
  }

  for (const row of rows) {
    const id = rowId(row);
    if (!id) continue;
    const bag = parsePreferencesBag(rowPreferences(row));
    if (!bag) continue;

    const allowed = buildAllowedFromRow(row);
    const allowedSet = new Set(allowed);
    const bagActive =
      bag.activeCompanyId && allowedSet.has(bag.activeCompanyId) ? bag.activeCompanyId : '';
    const bagEnabled = (bag.enabledCompanyIds || []).filter(cid => allowedSet.has(cid));

    const nextActive = rowActiveCompanyId(row) || bagActive || '';
    const keepEnabled = hasStoredEnabled(rowEnabledCompanyIds(row));
    const nextEnabled = keepEnabled ? null : bagEnabled.length ? bagEnabled : null;

    if (!nextActive && !nextEnabled) continue;

    const hadActive = Boolean(rowActiveCompanyId(row));
    try {
      if (!hadActive && nextActive && nextEnabled) {
        await db.execute(
          'UPDATE auth_user SET active_company_id = ?, enabled_company_ids = ? WHERE id = ?',
          JSON.stringify([nextActive, JSON.stringify(nextEnabled), id]),
        );
      } else if (!hadActive && nextActive) {
        await db.execute(
          'UPDATE auth_user SET active_company_id = ? WHERE id = ?',
          JSON.stringify([nextActive, id]),
        );
      } else if (nextEnabled) {
        await db.execute(
          'UPDATE auth_user SET enabled_company_ids = ? WHERE id = ?',
          JSON.stringify([JSON.stringify(nextEnabled), id]),
        );
      }
    } catch (err) {
      console.warn('[auth] preferences company-scope migrate row failed', id, err);
    }
  }
}
