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
  if (raw == null) return false;
  if (Array.isArray(raw)) return raw.length > 0;
  if (typeof raw === 'string') {
    const text = raw.trim();
    if (!text || text === '[]' || text === 'null') return false;
    try {
      const parsed = JSON.parse(text);
      return Array.isArray(parsed) && parsed.length > 0;
    } catch {
      return text.length > 0;
    }
  }
  return false;
}

/**
 * Copy Preferences.activeCompanyId / enabledCompanyIds into first-class columns
 * when those columns are still empty. No-ops when the leftover Preferences column
 * is already gone.
 */
export async function migratePreferencesCompanyScopeToColumns(): Promise<void> {
  const db = resolveDb();
  if (!db?.query || !db?.execute) return;

  let rows: AuthUserScopeRow[] = [];
  try {
    const raw = await db.query(
      'SELECT id, preferences, active_company_id, enabled_company_ids FROM auth_user WHERE preferences IS NOT NULL',
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

    const nextActive = rowActiveCompanyId(row) || bag.activeCompanyId || '';
    const keepEnabled = hasStoredEnabled(rowEnabledCompanyIds(row));
    const nextEnabled = keepEnabled
      ? null
      : bag.enabledCompanyIds && bag.enabledCompanyIds.length
        ? bag.enabledCompanyIds
        : null;

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
