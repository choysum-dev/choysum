// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc';
import type MetaApplicationModel from '@/meta/service/models/application';
import type MetaModelModel from '@/meta/service/models/model';

const MetaApplication = createServiceByModel<typeof MetaApplicationModel>('meta.MetaApplication');
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');

function moduleIdEmpty(row: any): boolean {
  const raw = row.ModuleId ?? row.module_id ?? row.ModuleID;
  if (raw == null) return true;
  if (raw === '') return true;
  if (typeof raw === 'object') {
    const id = (raw as any).Id ?? (raw as any).id;
    if (id == null) return true;
    return String(id).trim() === '';
  }
  return String(raw).trim() === '';
}

function rowId(row: any): string {
  if (row == null) return '';
  return String(row.Id ?? row.id ?? '').trim();
}

function rowUpdatedAt(row: any): number {
  let raw: any = row.UpdatedAt;
  if (raw == null) {
    raw = row.updated_at;
  }
  if (raw == null) return 0;
  if (raw === '') return 0;
  if (typeof raw === 'number') return raw;
  const ms = Date.parse(String(raw));
  if (!Number.isFinite(ms)) return 0;
  return ms;
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
      if (rowTs > bestTs) {
        best = row;
      } else if (rowTs === bestTs) {
        if (rowId(row) > rowId(best)) {
          best = row;
        }
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
  const pageSize = 500;
  const models: any[] = [];
  for (let offset = 0; ; offset += pageSize) {
    const page = await MetaModel.Search(
      {
        And: [
          ['Name', '=', modelName],
          ['Application', '=', appName],
        ],
      } as any,
      {
        fields: selectedFields,
        orderBy: { field: 'UpdatedAt', order: 'desc' },
        limit: pageSize,
        offset,
      } as any
    );
    const batch = (page as any[]) || [];
    models.push(...batch);
    if (batch.length < pageSize) break;
  }
  const rows = models.filter((m: any) => rowId(m));
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
