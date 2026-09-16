// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId, normalizeRefIdList } from '@/core/service/utils/normalization';
import type { Insertable } from '@/core/service/api/input';
import RoleUiResource from './role_ui_resource';

type RoleAccessRow = {
  Id?: unknown;
  AccessUiResourceIds?: string[];
};

type UiResourceRow = Record<string, unknown> & {
  Id?: string;
  Mode?: string;
  MetaApplicationId?: string | null;
  MetaUiResourceId?: string | null;
};

const hasOwn = (obj: Record<string, unknown>, key: string): boolean => Object.prototype.hasOwnProperty.call(obj, key);

function isAllowResourceScope(row: unknown): boolean {
  const rec = row && typeof row === 'object' ? (row as Record<string, unknown>) : {};
  const mode = String(rec.Mode ?? 'allow')
    .trim()
    .toLowerCase();
  const uiResourceId = normalizeRefId(rec.MetaUiResourceId);
  const appId = normalizeRefId(rec.MetaApplicationId);
  return mode === 'allow' && !!uiResourceId && appId == null;
}

function makeAllowResourceEntries(ids: string[]): UiResourceRow[] {
  return ids.map(id => ({
    Mode: 'allow',
    MetaApplicationId: null,
    MetaUiResourceId: id,
  }));
}

function extractUiResourcesArray(v: unknown): unknown[] | null {
  if (Array.isArray(v)) return v;
  if (v && typeof v === 'object' && Array.isArray((v as { replace?: unknown }).replace)) {
    return (v as { replace: unknown[] }).replace;
  }
  return null;
}

async function loadUiResourcesForRole(roleId: string): Promise<UiResourceRow[]> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    },
    { fields: ['Id', 'Mode', 'MetaApplicationId', 'MetaUiResourceId'] }
  );

  return (rows || []).map(row => {
    const rec = row as unknown as Record<string, unknown>;
    return {
      ...rec,
      Id: normalizeRefId(rec.Id) ?? undefined,
      Mode: String(rec.Mode ?? 'allow')
        .trim()
        .toLowerCase(),
      MetaApplicationId: normalizeRefId(rec.MetaApplicationId),
      MetaUiResourceId: normalizeRefId(rec.MetaUiResourceId),
    };
  });
}

function mergeAccessIntoUiResources(baseRows: UiResourceRow[], accessIds: string[]): UiResourceRow[] {
  const preserved = (baseRows || []).filter(row => !isAllowResourceScope(row));
  const allowRows = makeAllowResourceEntries(accessIds);
  return [...preserved, ...allowRows];
}

/**
 * Rewrite AccessUiResourceIds into UiResources for create payloads.
 */
export async function applyAccessWriteTransformOnCreate(values: Record<string, unknown>): Promise<string[] | null> {
  if (!hasOwn(values, 'AccessUiResourceIds')) return null;

  const accessIds = normalizeRefIdList(values.AccessUiResourceIds);
  const incomingUiRows = extractUiResourcesArray(values.UiResources);
  const baseRows = Array.isArray(incomingUiRows) ? (incomingUiRows as UiResourceRow[]) : [];

  values.UiResources = mergeAccessIntoUiResources(baseRows, accessIds);
  delete values.AccessUiResourceIds;
  return accessIds;
}

/**
 * Rewrite AccessUiResourceIds into UiResources for update payloads.
 */
export async function applyAccessWriteTransformOnUpdate(values: Record<string, unknown>, roleId: string): Promise<string[] | null> {
  if (!hasOwn(values, 'AccessUiResourceIds')) return null;

  const accessIds = normalizeRefIdList(values.AccessUiResourceIds);
  const incomingUiRows = extractUiResourcesArray(values.UiResources);
  const baseRows = Array.isArray(incomingUiRows) ? (incomingUiRows as UiResourceRow[]) : await loadUiResourcesForRole(roleId);

  values.UiResources = mergeAccessIntoUiResources(baseRows, accessIds);
  delete values.AccessUiResourceIds;
  return accessIds;
}

/**
 * Persist allow/resource UI grants so they stay aligned with AccessUiResourceIds.
 */
export async function syncAllowResourceGrants(roleId: string, accessIds: string[]): Promise<void> {
  const targetIds = normalizeRefIdList(accessIds);
  const targetSet = new Set<string>(targetIds);

  const rows = await loadUiResourcesForRole(roleId);
  const allowRows = rows.filter(row => isAllowResourceScope(row));

  const existingByResource = new Map<string, string>();
  for (const row of allowRows) {
    const id = normalizeRefId(row.Id);
    const resourceId = normalizeRefId(row.MetaUiResourceId);
    if (!id || !resourceId) continue;
    existingByResource.set(resourceId, id);
  }

  const deleteIds = Array.from(existingByResource.entries())
    .filter(([resourceId]) => !targetSet.has(resourceId))
    .map(([, rowId]) => rowId);

  if (deleteIds.length === 1) {
    await RoleUiResource.DeleteById(deleteIds[0]);
  } else if (deleteIds.length > 1) {
    await RoleUiResource.Delete(['Id', 'in', deleteIds]);
  }

  const createRows = targetIds
    .filter(resourceId => !existingByResource.has(resourceId))
    .map(resourceId => ({
      RoleId: { Id: roleId },
      Mode: 'allow',
      MetaApplicationId: null,
      MetaUiResourceId: resourceId,
    }));

  if (createRows.length) {
    await RoleUiResource.CreateMany(createRows as Array<Partial<Insertable<RoleUiResource>>>);
  }
}

async function buildAccessMap(roleIds: string[]): Promise<Map<string, string[]>> {
  const out = new Map<string, string[]>();
  if (!roleIds.length) return out;

  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', 'in', roleIds]],
    },
    { fields: ['RoleId', 'Mode', 'MetaApplicationId', 'MetaUiResourceId'], limit: Math.max(1000, roleIds.length * 200) }
  );

  const map = new Map<string, Set<string>>();
  for (const row of rows || []) {
    const rec = row as unknown as Record<string, unknown>;
    const roleId = normalizeRefId(rec.RoleId);
    if (!roleId) continue;
    if (!isAllowResourceScope(row)) continue;
    const resourceId = normalizeRefId(rec.MetaUiResourceId);
    if (!resourceId) continue;
    if (!map.has(roleId)) map.set(roleId, new Set<string>());
    map.get(roleId)!.add(resourceId);
  }

  for (const roleId of roleIds) {
    out.set(roleId, Array.from(map.get(roleId) || []));
  }
  return out;
}

/**
 * Hydrate AccessUiResourceIds onto result rows when callers request the field.
 */
export async function hydrateAccessUiResourceIds(records: RoleAccessRow[]): Promise<void> {
  if (!Array.isArray(records) || records.length === 0) return;
  const roleIds = Array.from(
    new Set(
      records
        .map(row => normalizeRefId(row?.Id))
        .filter(Boolean)
        .map(String)
    )
  );
  if (!roleIds.length) return;

  const accessMap = await buildAccessMap(roleIds);
  for (const row of records) {
    const roleId = normalizeRefId(row?.Id);
    if (!roleId) continue;
    row.AccessUiResourceIds = accessMap.get(roleId) || [];
  }
}

/**
 * Check whether a field selection asks for AccessUiResourceIds.
 */
export function wantsAccessField(selection: unknown): boolean {
  if (selection == null) return false;
  if (typeof selection === 'string') return selection === 'AccessUiResourceIds';
  if (Array.isArray(selection)) return selection.some(it => wantsAccessField(it));
  if (typeof selection === 'object' && Array.isArray((selection as { fields?: unknown }).fields)) {
    return wantsAccessField((selection as { fields: unknown }).fields);
  }
  return false;
}
