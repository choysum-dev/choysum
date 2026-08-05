// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';

const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');

function moduleIdEmpty(row: any): boolean {
  const raw = row?.ModuleId ?? row?.module_id ?? row?.ModuleID;
  if (raw == null || raw === '') return true;
  if (typeof raw === 'object') {
    const id = (raw as any)?.Id ?? (raw as any)?.id;
    return id == null || String(id).trim() === '';
  }
  return String(raw).trim() === '';
}

function rowId(row: any): string {
  return String(row?.Id ?? row?.id ?? '').trim();
}

function rowUpdatedAt(row: any): number {
  const raw = row?.UpdatedAt ?? row?.updated_at;
  if (raw == null || raw === '') return 0;
  if (typeof raw === 'number') return raw;
  const ms = Date.parse(String(raw));
  return Number.isFinite(ms) ? ms : 0;
}

/** Align with Go pickEffectiveAmong: empty ModuleId first, then newest UpdatedAt, then larger Id. */
function pickEffectiveAmong(rows: any[]): any {
  let best = rows[0];
  for (let i = 1; i < rows.length; i++) {
    const row = rows[i];
    const empty = moduleIdEmpty(row);
    const bestEmpty = moduleIdEmpty(best);
    if (empty && !bestEmpty) {
      best = row;
      continue;
    }
    if (empty === bestEmpty) {
      const rowTs = rowUpdatedAt(row);
      const bestTs = rowUpdatedAt(best);
      if (rowTs > bestTs || (rowTs === bestTs && rowId(row) > rowId(best))) {
        best = row;
      }
    }
  }
  return best;
}

/**
 * Resolve the single effective MetaModel id for (application, name).
 * Prefers empty ModuleId (E2 projection) over legacy declaration shells.
 */
export async function resolveEffectiveModelId(appName: string, modelName: string): Promise<string> {
  const hit = await resolveEffectiveModelRow(appName, modelName, ['Id', 'ModuleId', 'UpdatedAt']);
  return String(hit?.Id || '').trim();
}

/**
 * Resolve the effective MetaModel row (optional extra fields).
 * Always fetches Id/ModuleId/UpdatedAt so shell vs effective selection stays correct
 * even when callers omit those columns from `fields`.
 */
export async function resolveEffectiveModelRow(
  appName: string,
  modelName: string,
  fields: string[] = ['Id', 'ModuleId', 'UpdatedAt', 'CompanyField']
): Promise<any | undefined> {
  const selectedFields = Array.from(new Set(['Id', 'ModuleId', 'UpdatedAt', ...fields]));
  const models = await MetaModel.Search(
    {
      And: [
        ['Name', '=', modelName],
        ['Application', '=', appName],
      ],
    } as any,
    {
      fields: selectedFields,
      orderBy: { field: 'UpdatedAt', order: 'desc' },
      limit: 50,
    } as any
  );
  const rows = (models || []).filter((m: any) => rowId(m));
  if (rows.length === 0) return undefined;
  if (rows.length === 1) return rows[0];
  return pickEffectiveAmong(rows);
}

/**
 * Resolve meta.MetaApplication id by name (single row).
 */
export async function resolveEffectiveApplicationId(appName: string): Promise<string> {
  const apps = await MetaApplication.Search(['Name', '=', appName] as any, {
    fields: ['Id', 'UpdatedAt'],
    orderBy: { field: 'UpdatedAt', order: 'desc' },
    limit: 1,
  } as any);
  return String((apps as any)?.[0]?.Id || '').trim();
}
