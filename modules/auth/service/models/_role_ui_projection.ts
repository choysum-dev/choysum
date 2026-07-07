// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';
import RoleUiResource from './role_ui_resource';

const hasOwn = (obj: Record<string, any>, key: string): boolean => Object.prototype.hasOwnProperty.call(obj, key);

function normalizeIdList(v: any): string[] {
  if (v == null) return [];
  const arr = Array.isArray(v) ? v : [v];
  const set = new Set<string>();
  for (const it of arr) {
    const id = normalizeRefId(it);
    if (id) set.add(id);
  }
  return Array.from(set);
}

function readRoleId(row: any): string | null {
  return normalizeRefId(row?.RoleId);
}

function isAllowResourceScope(row: any): boolean {
  const mode = String((row as any)?.Mode ?? 'allow')
    .trim()
    .toLowerCase();
  const uiResourceId = normalizeRefId((row as any)?.IrUiResourceId);
  const appId = normalizeRefId((row as any)?.IrApplicationId);
  return mode === 'allow' && !!uiResourceId && appId == null;
}

function makeAllowResourceEntries(ids: string[]): Array<Record<string, any>> {
  return ids.map(id => ({
    Mode: 'allow',
    IrApplicationId: null,
    IrUiResourceId: id,
  }));
}

function extractUiResourcesArray(v: any): any[] | null {
  if (Array.isArray(v)) return v;
  if (v && typeof v === 'object' && Array.isArray((v as any).replace)) return (v as any).replace;
  return null;
}

async function loadUiResourcesForRole(roleId: string): Promise<Array<Record<string, any>>> {
  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', '=', roleId]],
    } as any,
    { fields: ['Id', 'Mode', 'IrApplicationId', 'IrUiResourceId'] as any } as any
  );

  return (rows || []).map((row: any) => ({
    ...(row || {}),
    Id: normalizeRefId(row?.Id) ?? undefined,
    Mode: String((row as any)?.Mode ?? 'allow')
      .trim()
      .toLowerCase(),
    IrApplicationId: normalizeRefId((row as any)?.IrApplicationId),
    IrUiResourceId: normalizeRefId((row as any)?.IrUiResourceId),
  }));
}

function mergeAccessIntoUiResources(baseRows: Array<Record<string, any>>, accessIds: string[]): Array<Record<string, any>> {
  const preserved = (baseRows || []).filter(row => !isAllowResourceScope(row));
  const allowRows = makeAllowResourceEntries(accessIds);
  return [...preserved, ...allowRows];
}

/**
 * Rewrite AccessUiResourceIds into UiResources for create payloads.
 */
export async function applyAccessWriteTransformOnCreate(values: Record<string, any>): Promise<string[] | null> {
  if (!hasOwn(values, 'AccessUiResourceIds')) return null;

  const accessIds = normalizeIdList(values.AccessUiResourceIds);
  const incomingUiRows = extractUiResourcesArray(values.UiResources);
  const baseRows = Array.isArray(incomingUiRows) ? incomingUiRows : [];

  values.UiResources = mergeAccessIntoUiResources(baseRows, accessIds);
  delete values.AccessUiResourceIds;
  return accessIds;
}

/**
 * Rewrite AccessUiResourceIds into UiResources for update payloads.
 */
export async function applyAccessWriteTransformOnUpdate(values: Record<string, any>, roleId: string): Promise<string[] | null> {
  if (!hasOwn(values, 'AccessUiResourceIds')) return null;

  const accessIds = normalizeIdList(values.AccessUiResourceIds);
  const incomingUiRows = extractUiResourcesArray(values.UiResources);
  const baseRows = Array.isArray(incomingUiRows) ? incomingUiRows : await loadUiResourcesForRole(roleId);

  values.UiResources = mergeAccessIntoUiResources(baseRows, accessIds);
  delete values.AccessUiResourceIds;
  return accessIds;
}

/**
 * Persist allow/resource UI grants so they stay aligned with AccessUiResourceIds.
 */
export async function syncAllowResourceGrants(roleId: string, accessIds: string[]): Promise<void> {
  const targetIds = normalizeIdList(accessIds);
  const targetSet = new Set<string>(targetIds);

  const rows = await loadUiResourcesForRole(roleId);
  const allowRows = rows.filter(row => isAllowResourceScope(row));

  const existingByResource = new Map<string, string>();
  for (const row of allowRows) {
    const id = normalizeRefId((row as any).Id);
    const resourceId = normalizeRefId((row as any).IrUiResourceId);
    if (!id || !resourceId) continue;
    existingByResource.set(resourceId, id);
  }

  const deleteIds = Array.from(existingByResource.entries())
    .filter(([resourceId]) => !targetSet.has(resourceId))
    .map(([, rowId]) => rowId);

  if (deleteIds.length === 1) {
    await RoleUiResource.DeleteById(deleteIds[0]);
  } else if (deleteIds.length > 1) {
    await RoleUiResource.Delete(['Id', 'in', deleteIds] as any);
  }

  const createRows = targetIds
    .filter(resourceId => !existingByResource.has(resourceId))
    .map(resourceId => ({
      RoleId: { Id: roleId } as any,
      Mode: 'allow',
      IrApplicationId: null,
      IrUiResourceId: resourceId,
    }));

  if (createRows.length) {
    await RoleUiResource.CreateMany(createRows as any);
  }
}

async function buildAccessMap(roleIds: string[]): Promise<Map<string, string[]>> {
  const out = new Map<string, string[]>();
  if (!roleIds.length) return out;

  const rows = await RoleUiResource.Search(
    {
      And: [['RoleId', 'in', roleIds]],
    } as any,
    { fields: ['RoleId', 'Mode', 'IrApplicationId', 'IrUiResourceId'] as any, limit: Math.max(1000, roleIds.length * 200) } as any
  );

  const map = new Map<string, Set<string>>();
  for (const row of rows || []) {
    const roleId = readRoleId(row as any);
    if (!roleId) continue;
    if (!isAllowResourceScope(row)) continue;
    const resourceId = normalizeRefId((row as any)?.IrUiResourceId);
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
export async function hydrateAccessUiResourceIds(records: any[]): Promise<void> {
  if (!Array.isArray(records) || records.length === 0) return;
  const roleIds = Array.from(
    new Set(
      records
        .map(row => normalizeRefId((row as any)?.Id))
        .filter(Boolean)
        .map(String)
    )
  );
  if (!roleIds.length) return;

  const accessMap = await buildAccessMap(roleIds);
  for (const row of records) {
    const roleId = normalizeRefId((row as any)?.Id);
    if (!roleId) continue;
    (row as any).AccessUiResourceIds = accessMap.get(roleId) || [];
  }
}

/**
 * Check whether a field selection asks for AccessUiResourceIds.
 */
export function wantsAccessField(selection: any): boolean {
  if (selection == null) return false;
  if (typeof selection === 'string') return selection === 'AccessUiResourceIds';
  if (Array.isArray(selection)) return selection.some(it => wantsAccessField(it));
  if (typeof selection === 'object' && Array.isArray((selection as any).fields)) {
    return wantsAccessField((selection as any).fields);
  }
  return false;
}
