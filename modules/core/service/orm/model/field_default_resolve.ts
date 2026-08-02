// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type FieldDefaultScopeRow = {
  Id?: string | null;
  Field?: string | null;
  UserId?: string | null;
  CompanyId?: string | null;
  Value?: unknown;
};

function scopeRank(userId: unknown, companyId: unknown): number {
  const hasUser = userId != null && String(userId).trim() !== '';
  const hasCompany = companyId != null && String(companyId).trim() !== '';
  if (hasUser && hasCompany) return 1;
  if (hasUser) return 2;
  if (hasCompany) return 3;
  return 4;
}

/**
 * Merge FieldDefault candidate rows into an effective field→value map.
 * Priority: user+company > user > company > global. Same-rank ties pick smallest Id and warn.
 */
export function resolveEffectiveFieldDefaults(
  rows: FieldDefaultScopeRow[],
  fieldNames?: string[]
): Record<string, unknown> {
  const allow = fieldNames && fieldNames.length ? new Set(fieldNames.map(String)) : undefined;
  const bestByField = new Map<string, { rank: number; id: string; value: unknown }>();

  for (const row of rows || []) {
    const field = String(row?.Field || '').trim();
    if (!field) continue;
    if (allow && !allow.has(field)) continue;

    const rank = scopeRank(row?.UserId, row?.CompanyId);
    const id = String(row?.Id || '');
    const prev = bestByField.get(field);
    if (!prev || rank < prev.rank || (rank === prev.rank && id && (!prev.id || id < prev.id))) {
      if (prev && rank === prev.rank && id && prev.id && id !== prev.id) {
        console.warn(
          `FIELD_DEFAULT_SCOPE_TIE field=${field} keepingId=${id < prev.id ? id : prev.id} droppingId=${id < prev.id ? prev.id : id}`
        );
      }
      bestByField.set(field, { rank, id, value: row?.Value });
    } else if (prev && rank === prev.rank && id && prev.id && id !== prev.id) {
      console.warn(`FIELD_DEFAULT_SCOPE_TIE field=${field} keepingId=${prev.id} droppingId=${id}`);
    }
  }

  const out: Record<string, unknown> = {};
  for (const [field, entry] of bestByField.entries()) {
    if (entry.value !== undefined) {
      out[field] = entry.value;
    }
  }
  return out;
}
