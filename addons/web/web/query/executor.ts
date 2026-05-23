// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Execute plans against the store API and produce a DataSetSnapshot
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { PlanBundle, QueryPlan } from './planner';
import type { AttachmentFieldDescriptor } from './types';
import { DataSetSnapshot, RecordRow, GroupRow } from './types';
import { exportFieldSelection, pathsToFieldSelection, ensureRootId } from './utils/registry/field';
import { exportMetrics } from './utils/registry/metric';
import { createStoreByModel } from '../stores/registry';
import { getTokenProvider } from '@/core/web/rpc/providers';

/**
 * Hydrates ManyToOneRef fields in bulk.
 * Replaces string ids in the result set with { Id, DisplayName } objects.
 */
async function hydrateManyToOneRefs(store: WebModelStore<any>, items: any[]) {
  if (!items || items.length === 0) return;

  try {
    const meta = store.fieldsMetadata || {};
    // Keep field names so records can be updated in place.
    const refFields = Object.entries(meta).filter(([, f]) => f.type === 'ManyToOneRef' && f.relationModel);

    if (refFields.length === 0) return;

    // 1. Collect ids into Map<TargetModel, Set<Id>>.
    const batch = new Map<string, Set<string>>();

    for (const item of items) {
      if (!item) continue;
      for (const [fieldName, f] of refFields) {
        const val = item[fieldName];
        // Only process string ids so objects are not hydrated twice.
        if (val && typeof val === 'string') {
          const target = f.relationModel!;
          if (!batch.has(target)) batch.set(target, new Set());
          batch.get(target)!.add(val);
        }
      }
    }

    // 2. Query target services in bulk.
    const lookups = new Map<string, Map<string, any>>(); // TargetModel -> (Id -> Object)

    await Promise.all(
      Array.from(batch.entries()).map(async ([model, ids]) => {
        if (ids.size === 0) return;
        try {
          const targetStore = createStoreByModel(model);
          const cond: any = ['Id', 'in', Array.from(ids)];
          const results = await targetStore.Search(cond, { fields: ['Id', 'DisplayName'] });

          const map = new Map<string, any>();
          for (const r of results) {
            map.set(r.Id, r);
          }
          lookups.set(model, map);
        } catch (e) {
          console.warn(`[Executor] Failed to hydrate ManyToOneRef for target '${model}'`, e);
        }
      })
    );

    // 3. Fill hydrated objects back into each record.
    for (const item of items) {
      if (!item) continue;
      for (const [fieldName, f] of refFields) {
        const val = item[fieldName];
        if (val && typeof val === 'string') {
          const target = f.relationModel!;
          const map = lookups.get(target);
          if (map && map.has(val)) {
            item[fieldName] = map.get(val);
          }
        }
      }
    }
  } catch (e) {
    console.error('[Executor] Error hydrating ManyToOneRef fields', e);
  }
}

/**
 * Hydrates ManyToManyRef fields in bulk.
 * Replaces id lists with target-model objects that at least contain Id and DisplayName.
 */
async function hydrateManyToManyRefs(store: WebModelStore<any>, items: any[]) {
  if (!items || items.length === 0) return;

  try {
    const meta = store.fieldsMetadata || {};
    const refFields = Object.entries(meta).filter(([, f]) => f.type === 'ManyToManyRef' && f.relationModel);
    if (refFields.length === 0) return;

    const batch = new Map<string, Set<string>>();

    for (const item of items) {
      if (!item) continue;
      for (const [fieldName, f] of refFields) {
        const val = item[fieldName];
        if (!Array.isArray(val)) continue;
        for (const entry of val) {
          const id = typeof entry === 'string' ? entry : (entry?.Id ?? entry?.id);
          if (!id) continue;
          const target = f.relationModel!;
          if (!batch.has(target)) batch.set(target, new Set());
          batch.get(target)!.add(String(id));
        }
      }
    }

    const lookups = new Map<string, Map<string, any>>();

    await Promise.all(
      Array.from(batch.entries()).map(async ([model, ids]) => {
        if (ids.size === 0) return;
        try {
          const targetStore = createStoreByModel(model);
          const selectionPaths = exportFieldSelection(targetStore.storeId) || [];
          const selection = ensureRootId(pathsToFieldSelection(selectionPaths) ?? ['DisplayName']);
          const results = await targetStore.Search(['Id', 'in', Array.from(ids)] as any, { fields: selection });

          const map = new Map<string, any>();
          for (const r of results || []) {
            const key = String(r?.Id ?? r?.id);
            map.set(key, r);
          }
          lookups.set(model, map);
        } catch (e) {
          console.warn(`[Executor] Failed to hydrate ManyToManyRef for target '${model}'`, e);
        }
      })
    );

    for (const item of items) {
      if (!item) continue;
      for (const [fieldName, f] of refFields) {
        const val = item[fieldName];
        if (!Array.isArray(val)) continue;
        const target = f.relationModel!;
        const map = lookups.get(target);
        if (!map) continue;
        item[fieldName] = val.map((entry: any) => {
          const id = typeof entry === 'string' ? entry : (entry?.Id ?? entry?.id);
          if (!id) return entry;
          return map.get(String(id)) ?? entry;
        });
      }
    }
  } catch (e) {
    console.error('[Executor] Error hydrating ManyToManyRef fields', e);
  }
}

function normalizeIdText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text ? text : undefined;
}

function isInternalStorageBindingContentUrl(rawUrl: string): boolean {
  const url = normalizeIdText(rawUrl);
  if (!url) return false;

  if (url.startsWith('/_document/bindings/')) {
    return url.includes('/content');
  }

  const locationHref = typeof globalThis !== 'undefined' ? globalThis.location?.href : undefined;
  const locationOrigin = typeof globalThis !== 'undefined' ? globalThis.location?.origin : undefined;
  if (!locationHref || !locationOrigin) return false;

  try {
    const parsed = new URL(url, locationHref);
    return parsed.origin === locationOrigin && parsed.pathname.startsWith('/_document/bindings/') && parsed.pathname.includes('/content');
  } catch {
    return false;
  }
}

function appendStorageQueryToken(rawUrl: string | undefined, token: string | undefined): string | undefined {
  const url = normalizeIdText(rawUrl);
  if (!url) return undefined;
  if (!token || !isInternalStorageBindingContentUrl(url)) return url;

  if (url.startsWith('/')) {
    try {
      const parsed = new URL(url, 'http://choysum.local');
      if (!parsed.searchParams.has('token')) {
        parsed.searchParams.set('token', token);
      }
      return `${parsed.pathname}${parsed.search}${parsed.hash}`;
    } catch {
      return url;
    }
  }

  const locationHref = typeof globalThis !== 'undefined' ? globalThis.location?.href : undefined;
  if (!locationHref) return url;

  try {
    const parsed = new URL(url, locationHref);
    if (!parsed.searchParams.has('token')) {
      parsed.searchParams.set('token', token);
    }
    return parsed.toString();
  } catch {
    return url;
  }
}

async function resolveStorageQueryToken(): Promise<string | undefined> {
  const tokenProvider = getTokenProvider();
  if (!tokenProvider) return undefined;

  try {
    const needRefresh = await tokenProvider.shouldRefreshToken?.();
    if (needRefresh) {
      await tokenProvider.refreshToken();
    }
  } catch {
    // Best-effort refresh, keep current token when refresh fails.
  }

  try {
    return normalizeIdText(await tokenProvider.getToken());
  } catch {
    return undefined;
  }
}

function normalizeRelationProjectionPayload(items: any[]) {
  if (!Array.isArray(items) || items.length === 0) return;
  // Normalize $rel$_xxx relation payloads into PascalCase fields used by the frontend.
  try {
    for (const it of items) {
      if (!it || typeof it !== 'object') continue;
      const keys = Object.keys(it).filter(k => k.startsWith('$rel$_'));
      for (const k of keys) {
        const raw = k.substring('$rel$_'.length); // e.g. user_id
        // Convert snake_case to PascalCase while preserving Id casing.
        const pascal = raw
          .split('_')
          .map(seg => (seg.toLowerCase() === 'id' ? 'Id' : seg.charAt(0).toUpperCase() + seg.slice(1)))
          .join('');
        if (!pascal) continue;
        const v = (it as any)[k];
        let parsed = v;
        if (typeof v === 'string') {
          try {
            parsed = JSON.parse(v);
          } catch {}
        }
        // Promote parsed object keys to PascalCase as well.
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          const norm: any = {};
          for (const [rk, rv] of Object.entries(parsed)) {
            const segs = rk.split('_');
            const camel = segs.map(s => (s.toLowerCase() === 'id' ? 'Id' : s.charAt(0).toUpperCase() + s.slice(1))).join('');
            norm[camel] = rv;
          }
          (it as any)[pascal] = norm;
        } else {
          (it as any)[pascal] = parsed;
        }
        // Keep the raw $rel$_ field for debugging if desired.
        // delete (it as any)[k];
      }
    }
  } catch {}
}

function decodeBinaryImageFields(store: WebModelStore<any>, items: any[]) {
  if (!Array.isArray(items) || items.length === 0) return;

  const fieldsMeta = store.fieldsMetadata || {};
  const descriptorFields = Object.entries(fieldsMeta).filter(([, meta]) => {
    const t = String((meta as any)?.type || '').toLowerCase();
    return t === 'binary' || t === 'image';
  });
  if (!descriptorFields.length) return;

  const ownerModel = normalizeIdText((store as any).fullModelName);

  for (const item of items) {
    if (!item || typeof item !== 'object') continue;

    const ownerRecordId = normalizeIdText((item as any).Id ?? (item as any).id);

    for (const [fieldName, meta] of descriptorFields) {
      const raw = (item as any)[fieldName];
      if (raw == null) continue;

      const fieldType = String((meta as any)?.type || '').toLowerCase() === 'image' ? 'image' : 'binary';

      let attachmentBindingId: string | undefined;
      let fileName: string | undefined;
      let displayName: string | undefined;
      let previewUrl: string | undefined;

      if (typeof raw === 'string') {
        attachmentBindingId = normalizeIdText(raw);
      } else if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
        const r = raw as Record<string, unknown>;
        attachmentBindingId = normalizeIdText(r.attachmentBindingId ?? r.bindingId ?? r.Id ?? r.id);
        fileName = normalizeIdText(r.fileName ?? r.displayFileName ?? r.originalFileName);
        displayName = normalizeIdText(r.displayName ?? r.name ?? r.fileName ?? r.displayFileName);
        previewUrl = normalizeIdText(r.previewUrl ?? r.url ?? r.downloadUrl);
      }

      if (!attachmentBindingId) continue;

      const descriptor: AttachmentFieldDescriptor = {
        kind: 'attachment',
        fieldType,
        fieldName,
        attachmentBindingId,
      };
      if (ownerModel) descriptor.ownerModel = ownerModel;
      if (ownerRecordId) descriptor.ownerRecordId = ownerRecordId;
      if (fileName) descriptor.fileName = fileName;
      if (displayName) descriptor.displayName = displayName;
      if (previewUrl) descriptor.previewUrl = previewUrl;

      (item as any)[fieldName] = descriptor;
    }
  }
}

type AttachmentDescriptorEnrichDeps = {
  createStoreByModel: typeof createStoreByModel;
};

function isAttachmentDescriptor(value: unknown): value is AttachmentFieldDescriptor {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const candidate = value as AttachmentFieldDescriptor;
  return candidate.kind === 'attachment' && !!normalizeIdText(candidate.attachmentBindingId);
}

type AttachmentBatchDescribeItem = {
  attachmentBindingId?: unknown;
  descriptor?: {
    fileName?: unknown;
    mimeType?: unknown;
    downloadUrl?: unknown;
  };
  displayName?: unknown;
  previewUrl?: unknown;
};

async function enrichBinaryImageDescriptors(
  store: WebModelStore<any>,
  items: any[],
  deps: AttachmentDescriptorEnrichDeps = { createStoreByModel }
): Promise<void> {
  if (!Array.isArray(items) || items.length === 0) return;

  const fieldsMeta = store.fieldsMetadata || {};
  const descriptorFieldNames = Object.entries(fieldsMeta)
    .filter(([, meta]) => {
      const t = String((meta as any)?.type || '').toLowerCase();
      return t === 'binary' || t === 'image';
    })
    .map(([fieldName]) => fieldName);
  if (!descriptorFieldNames.length) return;

  const descriptors: AttachmentFieldDescriptor[] = [];
  const bindingIds = new Set<string>();

  for (const item of items) {
    if (!item || typeof item !== 'object') continue;
    for (const fieldName of descriptorFieldNames) {
      const value = (item as any)[fieldName];
      if (!isAttachmentDescriptor(value)) continue;
      descriptors.push(value);
      bindingIds.add(value.attachmentBindingId);
    }
  }

  if (!descriptors.length || !bindingIds.size) return;

  let bindingStore: WebModelStore<any>;
  try {
    bindingStore = deps.createStoreByModel('document.AttachmentBinding');
  } catch {
    return;
  }

  const batchDescribe = (bindingStore as any)?.BatchDescribe;
  if (typeof batchDescribe !== 'function') {
    return;
  }

  let describeItems: AttachmentBatchDescribeItem[] = [];
  try {
    const response = await batchDescribe({
      attachmentBindingIds: Array.from(bindingIds),
    });
    describeItems = Array.isArray((response as any)?.items) ? ((response as any).items as AttachmentBatchDescribeItem[]) : [];
  } catch (e) {
    console.warn('[Executor] Failed to enrich binary/image descriptors via document.AttachmentBinding.BatchDescribe', e);
    return;
  }

  const describeByBindingId = new Map<
    string,
    {
      fileName?: string;
      displayName?: string;
      previewUrl?: string;
      mimeType?: string;
      downloadUrl?: string;
    }
  >();

  const queryToken = await resolveStorageQueryToken();

  for (const item of describeItems) {
    const bindingId = normalizeIdText(item?.attachmentBindingId);
    if (!bindingId) continue;

    const descriptorRaw = item?.descriptor;
    const fileName = normalizeIdText(descriptorRaw?.fileName);
    const mimeType = normalizeIdText(descriptorRaw?.mimeType);
    const downloadUrl = normalizeIdText(descriptorRaw?.downloadUrl);
    const displayName = normalizeIdText(item?.displayName) || fileName;
    const previewUrl = normalizeIdText(item?.previewUrl);

    describeByBindingId.set(bindingId, {
      fileName,
      displayName,
      previewUrl,
      mimeType,
      downloadUrl,
    });
  }

  for (const descriptor of descriptors) {
    const described = describeByBindingId.get(descriptor.attachmentBindingId);

    if (!descriptor.fileName) {
      descriptor.fileName = described?.fileName;
    }

    if (!descriptor.displayName) {
      descriptor.displayName = described?.displayName || descriptor.fileName;
    }

    if (descriptor.previewUrl) {
      descriptor.previewUrl = appendStorageQueryToken(descriptor.previewUrl, queryToken);
    }

    const canPreview =
      descriptor.fieldType === 'image' ||
      String(described?.mimeType || '')
        .toLowerCase()
        .startsWith('image/');
    if (!descriptor.previewUrl && canPreview) {
      descriptor.previewUrl = appendStorageQueryToken(
        described?.previewUrl || described?.downloadUrl || `/_document/bindings/${descriptor.attachmentBindingId}/content`,
        queryToken
      );
    }
  }
}

function withSelections(plan: QueryPlan, storeId: string): QueryPlan {
  if (plan.kind === 'search' || plan.kind === 'browse') {
    const paths = exportFieldSelection(storeId);
    const fieldSel = pathsToFieldSelection(paths);
    // Search and browse plans do not need metrics and must always include the root Id.
    return { ...plan, params: { ...plan.params, fields: ensureRootId(fieldSel) } };
  }
  if (plan.kind === 'readGroup' || plan.kind === 'readGroupCount') {
    const metrics = exportMetrics(storeId);
    // Convert MetricSpec values into the ReadGroupOptions.fields structure.
    const fieldAggs = metrics.map(m => ({ field: m.field, agg: m.agg, alias: m.alias }));
    // Drop legacy metrics and use fields instead.
    const { metrics: _oldMetrics, ...rest } = plan.params || {};
    return { ...plan, params: { ...rest, fields: fieldAggs } };
  }
  return plan;
}

async function call(plan: QueryPlan, store: WebModelStore<any>): Promise<any> {
  switch (plan.kind) {
    case 'search': {
      const { condition, fields, orderBy, limit, offset } = plan.params || {};
      const options = { fields, orderBy, limit, offset };
      return store.Search!(condition ?? [], options);
    }
    case 'count': {
      const { condition } = plan.params || {};
      return store.Count!(condition ?? []);
    }
    case 'readGroup': {
      const { groupby, condition, fields, orderBy, limit, offset } = plan.params || {};
      const opts = { fields, orderBy, limit, offset };
      const gbArray: any[] = Array.isArray(groupby) ? groupby : groupby ? [groupby] : [];
      const cond = Array.isArray(condition) || !condition ? (condition ?? []) : condition;
      return store.ReadGroup!(gbArray, cond, opts);
    }
    case 'readGroupCount': {
      const { groupby, condition, fields } = plan.params || {};
      const gbArray: any[] = Array.isArray(groupby) ? groupby : groupby ? [groupby] : [];
      const cond = Array.isArray(condition) || !condition ? (condition ?? []) : condition;
      const opts = { fields };
      return store.ReadGroupCount!(gbArray, cond, opts);
    }
    case 'browse': {
      const { id, fields } = plan.params || {};
      return store.Browse!(id, fields);
    }
  }
}

function toRecordRows(items: any[]): RecordRow[] {
  return (items || []).map(rec => ({ kind: 'record', key: String(rec?.Id ?? rec?.id ?? Math.random()), payload: rec }));
}

function toGroupRows(groups: any[]): GroupRow[] {
  return (groups || []).map((g, i): GroupRow => {
    const isNew = g && typeof g === 'object' && 'keys' in g && 'labels' in g;
    if (isNew) {
      const aliases = Object.keys((g as any).keys || {}).sort();
      const composite = aliases.map(a => String((g as any).keys[a])).join('|');
      const firstAlias = aliases[0];
      const label = ((g as any).labels && (g as any).labels[firstAlias]) || composite || String(i);
      const stableKey = composite || String(i);
      const childrenRaw = Array.isArray((g as any).children) ? (g as any).children : [];
      const children: GroupRow[] = childrenRaw.length ? toGroupRows(childrenRaw) : [];
      return {
        kind: 'group',
        key: stableKey,
        depth: (g as any).depth ?? 0,
        label,
        count: typeof (g as any).count === 'number' ? (g as any).count : undefined,
        metrics: (g as any).metrics || {},
        __condition: (g as any).condition,
        keys: (g as any).keys || {},
        labels: (g as any).labels || {},
        children: children.length ? children : undefined,
        raw: g,
      };
    }
    const derivedKey = (g as any).key || (g as any).id || ((g as any).__condition ? JSON.stringify((g as any).__condition) : String(i));
    let label: string | undefined = (g as any).label ?? (g as any).value ?? (g as any).name;
    if (!label) label = derivedKey.includes('{') ? String(i) : String(derivedKey);
    return {
      kind: 'group',
      key: derivedKey,
      depth: (g as any).depth ?? 0,
      label,
      count: typeof (g as any).count === 'number' ? (g as any).count : typeof (g as any).__count === 'number' ? (g as any).__count : undefined,
      metrics: (g as any).metrics,
      __condition: (g as any).__condition ?? (g as any).condition,
      children: Array.isArray((g as any).children) ? toGroupRows((g as any).children) : undefined,
      raw: g,
    } as GroupRow;
  });
}

export async function execute(bundle: PlanBundle, store: WebModelStore<any>, uiView?: string, options?: { signal?: AbortSignal }): Promise<DataSetSnapshot> {
  const storeId: string = store.storeId;
  const main = withSelections(bundle.main, storeId);
  const aux = bundle.auxiliary?.map(p => withSelections(p, storeId)) ?? [];
  // Removed debug logging for execution plans

  // Early abort check
  if (options?.signal?.aborted) {
    throw Object.assign(new Error('AbortError'), { name: 'AbortError', code: 'ABORT_ERR' });
  }

  const [mainRes, ...auxRes] = await Promise.all([call(main, store), ...aux.map(p => call(p, store))]);

  if (main.kind === 'browse') {
    const rec = Array.isArray(mainRes) ? mainRes[0] : mainRes;
    // Keep browse decoding consistent with search decoding.
    if (rec) {
      await hydrateManyToOneRefs(store, [rec]);
      await hydrateManyToManyRefs(store, [rec]);
      decodeBinaryImageFields(store, [rec]);
      await enrichBinaryImageDescriptors(store, [rec]);
      normalizeRelationProjectionPayload([rec]);
    }
    const rows = rec ? toRecordRows([rec]) : [];
    return { kind: 'search', rows, total: rows.length, planHash: main.hash, ts: Date.now(), uiView };
  }
  if (main.kind === 'readGroup') {
    const groups = Array.isArray(mainRes) ? mainRes : (mainRes?.groups ?? []);
    const total = typeof auxRes[0] === 'number' ? auxRes[0] : (mainRes?.total ?? undefined);
    const rows = toGroupRows(groups);
    return { kind: 'group', rows, total, planHash: main.hash, ts: Date.now(), uiView };
  }
  // search
  const items = Array.isArray(mainRes) ? mainRes : (mainRes?.items ?? []);

  // Hydrate reference fields before normalization for top-level fields.
  await hydrateManyToOneRefs(store, items);
  await hydrateManyToManyRefs(store, items);

  // Decode binary and image fields into descriptors for field-view components.
  decodeBinaryImageFields(store, items);
  await enrichBinaryImageDescriptors(store, items);

  normalizeRelationProjectionPayload(items);
  const total = typeof auxRes[0] === 'number' ? auxRes[0] : (mainRes?.total ?? items.length);
  const rows = toRecordRows(items);
  return { kind: 'search', rows, total, planHash: main.hash, ts: Date.now(), uiView };
}

export const __decodeBinaryImageFieldsForTest = decodeBinaryImageFields;
export const __enrichBinaryImageDescriptorsForTest = enrichBinaryImageDescriptors;
