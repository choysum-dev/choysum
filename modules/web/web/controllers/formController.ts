// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// FormController: drives OFormView (display/edit/create)
// VM: { mode, draft, original, loading, error }
// Methods: beginDisplay/beginEdit/beginCreate/reset/validate/submit/delete/provideToChildren

import { reactive, toRaw, provide } from 'vue';
import { buildUpdatePayload } from '@/core/utils/diff';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createStoreByModel } from '@/web/web/stores/registry';
import { buildBrowseContext } from '@/web/web/query/context';
import { buildPlan } from '@/web/web/query/planner';
import { execute } from '@/web/web/query/executor';
import { exportFieldSelection, pathsToFieldSelection, ensureRootId } from '@/web/web/query/utils/registry/field';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import type { RecordRow, DataSetSnapshot, FormViewModel, FormMode, IFormViewController } from '@/web/web/query/types';
import { createAbortableRequests, isCancellation, CancellationError } from '@/web/web/query/utils/abortable';
import { handoffCache, flashRead } from '@/web/web/query/utils/handoff';
import { getCurrentRequestContext } from '@/core/rpc/context';
import { getTokenProvider, getCSRFProvider } from '@/core/web/rpc/providers';

// FormMode & FormViewModel centralized in query/types.ts

function clone<T>(v: T): T {
  try {
    return JSON.parse(JSON.stringify(v));
  } catch {
    return v;
  }
}

function setByPath(obj: any, path: string, value: any) {
  const parts = path.split('.').filter(Boolean);
  let cur = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    const k = parts[i];
    if (cur[k] == null || typeof cur[k] !== 'object') cur[k] = {};
    cur = cur[k];
  }
  cur[parts[parts.length - 1]] = value;
}

function getByPath(obj: any, path: string) {
  const parts = path.split('.').filter(Boolean);
  let cur = obj;
  for (const k of parts) {
    if (cur == null) return undefined;
    cur = cur[k];
  }
  return cur;
}

type UploadOperation = 'create' | 'update';

type PrepareUploadReq = {
  ownerModel: string;
  fieldName: string;
  operation: UploadOperation;
  ownerRecordId?: string;
  businessRequestId: string;
  proposedFileName?: string;
  proposedContentType?: string;
  proposedSizeBytes?: number;
  checksumSha256?: string;
  originalFileName?: string;
  clientContentType?: string;
};

type PrepareUploadResp = {
  uploadId?: string;
  uploadTarget?: {
    method?: string;
    url?: string;
    headers?: Record<string, string>;
  };
};

type FinalizeUploadResp = {
  attachmentObjectId?: string;
};

type AttachmentContentServiceLike = {
  PrepareUpload(req: PrepareUploadReq): Promise<PrepareUploadResp>;
  FinalizeUpload(req: { uploadId: string; businessRequestId: string }): Promise<FinalizeUploadResp>;
  setContext?: (ctx: Record<string, string>) => void;
  withContext?: <T>(ctx: Record<string, string>, fn: () => Promise<T>) => Promise<T>;
};

type AttachmentResolutionContext = {
  operation: UploadOperation;
  ownerModel: string;
  ownerRecordId?: string;
  fieldName: string;
  service: AttachmentContentServiceLike;
  store: WebModelStore<any>;
};

type AttachmentResolution = { kind: 'omit' } | { kind: 'clear' } | { kind: 'set'; attachmentObjectId: string };
type AttachmentDownloadDisposition = 'inline' | 'attachment';

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

function normalizeText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text === '' ? undefined : text;
}

function normalizeAttachmentDownloadDisposition(value: unknown): AttachmentDownloadDisposition | undefined {
  const normalized = normalizeText(value)?.toLowerCase();
  if (normalized === 'inline' || normalized === 'attachment') {
    return normalized;
  }
  return undefined;
}

function resolveAttachmentDisplayFileName(raw: unknown): string | undefined {
  if (typeof File !== 'undefined' && raw instanceof File) {
    return normalizeText(raw.name);
  }

  if (!isRecord(raw)) {
    return undefined;
  }

  const fileLike = raw.file ?? raw.blob ?? raw.data;
  const fileLikeName = isRecord(fileLike) ? normalizeText(fileLike.name) : undefined;

  return (
    normalizeText(raw.displayFileName) ||
    normalizeText(raw.displayName) ||
    normalizeText(raw.fileName) ||
    normalizeText(raw.originalFileName) ||
    normalizeText(raw.proposedFileName) ||
    fileLikeName
  );
}

function buildAttachmentWritePayload(raw: unknown, attachmentObjectId: string): string | Record<string, unknown> {
  const displayFileName = resolveAttachmentDisplayFileName(raw);
  const downloadDisposition = isRecord(raw) ? normalizeAttachmentDownloadDisposition(raw.downloadDisposition) : undefined;

  if (!displayFileName && !downloadDisposition) {
    return attachmentObjectId;
  }

  const payload: Record<string, unknown> = {
    attachmentObjectId,
  };
  if (displayFileName) {
    payload.displayFileName = displayFileName;
  }
  if (downloadDisposition) {
    payload.downloadDisposition = downloadDisposition;
  }
  return payload;
}

function buildBaggage(items: Record<string, string>): string {
  const pairs: string[] = [];
  for (const [key, value] of Object.entries(items || {})) {
    let normalizedKey = key.trim().toLowerCase();
    if (normalizedKey && !normalizedKey.startsWith('ctx.')) {
      normalizedKey = `ctx.${normalizedKey}`;
    }

    const normalizedValue = normalizeText(value);
    if (!normalizedKey || !normalizedValue) {
      continue;
    }

    pairs.push(`${encodeURIComponent(normalizedKey)}=${encodeURIComponent(normalizedValue)}`);
  }
  return pairs.join(',');
}

function isInternalDocumentUploadTarget(url: string): boolean {
  const trimmed = normalizeText(url);
  if (!trimmed) return false;
  if (trimmed.startsWith('/')) {
    return trimmed.startsWith('/_document/uploads/');
  }

  const locationHref = typeof globalThis !== 'undefined' ? globalThis.location?.href : undefined;
  const locationOrigin = typeof globalThis !== 'undefined' ? globalThis.location?.origin : undefined;
  if (!locationHref || !locationOrigin) {
    return false;
  }

  try {
    const parsed = new URL(trimmed, locationHref);
    return parsed.origin === locationOrigin && parsed.pathname.startsWith('/_document/uploads/');
  } catch {
    return false;
  }
}

async function applyInternalUploadAuthHeaders(headers: Headers): Promise<void> {
  if (!headers.has('x-xsrf-token')) {
    const csrfProvider = getCSRFProvider();
    if (csrfProvider) {
      try {
        const csrfToken = normalizeText(await csrfProvider.getCSRFToken());
        if (csrfToken) {
          headers.set('X-XSRF-TOKEN', csrfToken);
        }
      } catch {
        // Keep best-effort behavior consistent with RPC CSRF interceptor.
      }
    }
  }

  if (!headers.has('authorization')) {
    const tokenProvider = getTokenProvider();
    if (tokenProvider) {
      try {
        const needRefresh = await tokenProvider.shouldRefreshToken?.();
        if (needRefresh) {
          await tokenProvider.refreshToken();
        }
      } catch {
        // Keep best-effort behavior consistent with RPC auth interceptor.
      }

      try {
        const token = normalizeText(await tokenProvider.getToken());
        if (token) {
          headers.set('Authorization', `Bearer ${token}`);
        }
      } catch {
        // Fall back to existing headers/cookies if token provider is unavailable at runtime.
      }
    }
  }

  if (!headers.has('baggage')) {
    try {
      const baggage = buildBaggage(getCurrentRequestContext());
      if (baggage) {
        headers.set('baggage', baggage);
      }
    } catch {
      // Ignore context propagation failures and let server-side auth decide.
    }
  }
}

async function describeUploadError(response: Response): Promise<string | undefined> {
  let text = '';
  try {
    text = (await response.text()).trim();
  } catch {
    return undefined;
  }

  if (!text) return undefined;

  try {
    const payload = JSON.parse(text) as {
      code?: unknown;
      message?: unknown;
      metadata?: Record<string, unknown>;
    };
    const code = normalizeText(payload?.code);
    const message = normalizeText(payload?.message);
    const reason = normalizeText(payload?.metadata?.reason);
    const stage = normalizeText(payload?.metadata?.stage);

    const details = [code, message, reason ? `reason=${reason}` : undefined, stage ? `stage=${stage}` : undefined].filter(Boolean);
    if (details.length > 0) {
      return details.join(' | ');
    }
  } catch {
    // Keep raw text fallback when response body is not JSON.
  }

  return text;
}

function isBlobLike(value: unknown): value is Blob {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as { arrayBuffer?: unknown; size?: unknown };
  return typeof candidate.arrayBuffer === 'function' && typeof candidate.size === 'number';
}

function looksLikeUploadEnvelope(value: unknown): boolean {
  if (isBlobLike(value)) return true;
  if (!isRecord(value)) return false;
  if (Array.isArray(value)) return true;
  if ('attachmentObjectId' in value) return true;
  if ('file' in value || 'blob' in value || 'data' in value) return true;
  const kind = normalizeText(value.kind)?.toLowerCase();
  return kind === 'set' || kind === 'clear' || kind === 'noop';
}

function attachmentFieldNames(store: WebModelStore<any>): string[] {
  const metas = (store as any)?.fieldsMetadata as Record<string, { type?: string }> | undefined;
  if (!metas) return [];
  return Object.entries(metas)
    .filter(([, meta]) => {
      const t = String(meta?.type || '').toLowerCase();
      return t === 'binary' || t === 'image';
    })
    .map(([fieldName]) => fieldName);
}

function hasAttachmentFieldMutation(payload: Record<string, unknown>, fields: string[]): boolean {
  if (!payload || !fields.length) return false;
  return fields.some(fieldName => Object.prototype.hasOwnProperty.call(payload, fieldName));
}

function resolveOwnerModel(store: WebModelStore<any>): string {
  const fullModelName = store.fullModelName;
  if (!fullModelName) {
    throw new Error('[Attachment] store.fullModelName is required.');
  }
  return fullModelName;
}

function newBusinessRequestId(fieldName: string): string {
  const base =
    (typeof globalThis !== 'undefined' && typeof globalThis.crypto !== 'undefined' && typeof globalThis.crypto.randomUUID === 'function'
      ? globalThis.crypto.randomUUID()
      : `${Date.now()}_${Math.random().toString(36).slice(2)}`) || `${Date.now()}`;
  const safeField = fieldName.replace(/[^a-zA-Z0-9_.-]/g, '_');
  return `attach.${safeField}.${base}`;
}

async function sha256Hex(blob: Blob): Promise<string | undefined> {
  if (typeof globalThis === 'undefined') return undefined;
  const subtle = globalThis.crypto?.subtle;
  if (!subtle || typeof subtle.digest !== 'function') return undefined;
  try {
    const bytes = await blob.arrayBuffer();
    const digest = await subtle.digest('SHA-256', bytes);
    const hash = Array.from(new Uint8Array(digest))
      .map(v => v.toString(16).padStart(2, '0'))
      .join('');
    return hash || undefined;
  } catch {
    return undefined;
  }
}

function resolveAttachmentContentService(store: WebModelStore<any>): AttachmentContentServiceLike {
  const service = createStoreByModel('document.AttachmentContent') as unknown as AttachmentContentServiceLike;
  if (!service || typeof service.PrepareUpload !== 'function' || typeof service.FinalizeUpload !== 'function') {
    throw new Error(
      "[Attachment] document.AttachmentContent service is unavailable. Ensure module 'document' is installed and web dist artifacts are up to date."
    );
  }

  const context = typeof store.getContext === 'function' ? store.getContext() : {};
  if (context && Object.keys(context).length > 0 && typeof service.setContext === 'function') {
    service.setContext(context);
  }
  return service;
}

async function runWithStoreContext<T>(store: WebModelStore<any>, service: AttachmentContentServiceLike, fn: () => Promise<T>): Promise<T> {
  const context = typeof store.getContext === 'function' ? store.getContext() : {};
  if (context && Object.keys(context).length > 0 && typeof service.withContext === 'function') {
    return await service.withContext(context, fn);
  }
  return await fn();
}

async function uploadToTarget(fieldName: string, target: NonNullable<PrepareUploadResp['uploadTarget']>, body: Blob): Promise<void> {
  const url = normalizeText(target?.url);
  if (!url) {
    throw new Error(`[Attachment] ${fieldName}: upload target url is empty.`);
  }

  const method = (normalizeText(target?.method) || 'PUT').toUpperCase();
  if (method !== 'PUT') {
    throw new Error(`[Attachment] ${fieldName}: unsupported upload target method '${method}'.`);
  }

  const headers = new Headers();
  const rawHeaders = target?.headers || {};
  for (const [k, v] of Object.entries(rawHeaders)) {
    const key = normalizeText(k);
    const value = normalizeText(v);
    if (!key || !value) continue;
    headers.set(key, value);
  }

  const internalUploadTarget = isInternalDocumentUploadTarget(url);
  if (internalUploadTarget) {
    await applyInternalUploadAuthHeaders(headers);
  }

  const requestInit: RequestInit = {
    method,
    headers,
    body,
  };
  if (internalUploadTarget) {
    requestInit.credentials = 'include';
  }

  const response = await fetch(url, requestInit);

  if (!response.ok) {
    const detail = await describeUploadError(response);
    throw new Error(`[Attachment] ${fieldName}: upload failed with HTTP ${response.status}${detail ? ` (${detail})` : ''}.`);
  }
}

async function resolveUploadInputToObjectId(
  fieldName: string,
  blob: Blob,
  recordLike: Record<string, unknown> | undefined,
  ctx: AttachmentResolutionContext
): Promise<string> {
  const operation = ctx.operation;
  const ownerModel = ctx.ownerModel;
  const ownerRecordId = normalizeText(ctx.ownerRecordId);

  const preferredFileName =
    normalizeText(recordLike?.proposedFileName) ||
    normalizeText(recordLike?.fileName) ||
    normalizeText(recordLike?.originalFileName) ||
    (typeof File !== 'undefined' && blob instanceof File ? normalizeText(blob.name) : undefined);

  const contentType =
    normalizeText(recordLike?.proposedContentType) ||
    normalizeText(recordLike?.contentType) ||
    normalizeText(recordLike?.clientContentType) ||
    normalizeText((blob as any)?.type) ||
    'application/octet-stream';

  const businessRequestId = normalizeText(recordLike?.businessRequestId) || newBusinessRequestId(fieldName);
  const checksumSha256 = normalizeText(recordLike?.checksumSha256) || (await sha256Hex(blob));

  const prepareReq: PrepareUploadReq = {
    ownerModel,
    fieldName,
    operation,
    ownerRecordId,
    businessRequestId,
    proposedFileName: preferredFileName,
    proposedContentType: contentType,
    proposedSizeBytes: Number.isFinite((blob as any)?.size) ? Number((blob as any)?.size) : undefined,
    checksumSha256,
    originalFileName: preferredFileName,
    clientContentType: contentType,
  };

  const prepared = await runWithStoreContext(ctx.store, ctx.service, async () => {
    return await ctx.service.PrepareUpload(prepareReq);
  });

  const uploadId = normalizeText(prepared?.uploadId);
  const uploadTarget = prepared?.uploadTarget;
  if (!uploadId || !uploadTarget) {
    throw new Error(`[Attachment] ${fieldName}: PrepareUpload returned invalid upload target.`);
  }

  await uploadToTarget(fieldName, uploadTarget, blob);

  const finalized = await runWithStoreContext(ctx.store, ctx.service, async () => {
    return await ctx.service.FinalizeUpload({
      uploadId,
      businessRequestId,
    });
  });

  const attachmentObjectId = normalizeText(finalized?.attachmentObjectId);
  if (!attachmentObjectId) {
    throw new Error(`[Attachment] ${fieldName}: FinalizeUpload did not return attachmentObjectId.`);
  }

  return attachmentObjectId;
}

async function resolveAttachmentFieldValue(raw: unknown, ctx: AttachmentResolutionContext): Promise<AttachmentResolution> {
  if (raw === undefined) return { kind: 'omit' };
  if (raw === null) return { kind: 'clear' };

  const text = typeof raw === 'string' ? normalizeText(raw) : undefined;
  if (text) {
    return { kind: 'set', attachmentObjectId: text };
  }

  if (Array.isArray(raw)) {
    throw new Error(`[Attachment] ${ctx.fieldName}: array payload is not supported for binary/image fields.`);
  }

  if (isBlobLike(raw)) {
    const attachmentObjectId = await resolveUploadInputToObjectId(ctx.fieldName, raw, undefined, ctx);
    return { kind: 'set', attachmentObjectId };
  }

  if (!isRecord(raw)) {
    throw new Error(`[Attachment] ${ctx.fieldName}: expected attachmentObjectId string | null | omitted.`);
  }

  const kind = normalizeText(raw.kind)?.toLowerCase();
  if (kind === 'noop') return { kind: 'omit' };
  if (kind === 'clear') return { kind: 'clear' };

  const attachmentObjectId = normalizeText(raw.attachmentObjectId);
  if (attachmentObjectId) {
    return { kind: 'set', attachmentObjectId };
  }

  const fileLike = raw.file ?? raw.blob ?? raw.data;
  if (isBlobLike(fileLike)) {
    const objectId = await resolveUploadInputToObjectId(ctx.fieldName, fileLike, raw, ctx);
    return { kind: 'set', attachmentObjectId: objectId };
  }

  if (kind === 'set') {
    throw new Error(`[Attachment] ${ctx.fieldName}: kind='set' requires attachmentObjectId or file/blob payload.`);
  }

  throw new Error(`[Attachment] ${ctx.fieldName}: invalid attachment payload.`);
}

async function normalizeAttachmentFieldsInPayload(
  store: WebModelStore<any>,
  payload: Record<string, unknown>,
  options: {
    operation: UploadOperation;
    ownerModel: string;
    ownerRecordId?: string;
    fields: string[];
  }
): Promise<Record<string, unknown>> {
  if (!options.fields.length) return payload;

  const nextPayload: Record<string, unknown> = { ...payload };
  const service = resolveAttachmentContentService(store);

  for (const fieldName of options.fields) {
    if (!Object.prototype.hasOwnProperty.call(nextPayload, fieldName)) continue;

    const sourceValue = nextPayload[fieldName];

    const resolved = await resolveAttachmentFieldValue(sourceValue, {
      operation: options.operation,
      ownerModel: options.ownerModel,
      ownerRecordId: options.ownerRecordId,
      fieldName,
      service,
      store,
    });

    if (resolved.kind === 'omit') {
      delete nextPayload[fieldName];
      continue;
    }
    if (resolved.kind === 'clear') {
      nextPayload[fieldName] = null;
      continue;
    }
    nextPayload[fieldName] = buildAttachmentWritePayload(sourceValue, resolved.attachmentObjectId);
  }

  return nextPayload;
}

async function preNormalizeDraftForDiff(
  store: WebModelStore<any>,
  payload: Record<string, unknown>,
  options: {
    ownerModel: string;
    ownerRecordId?: string;
    fields: string[];
  }
): Promise<Record<string, unknown>> {
  if (!options.fields.length) return payload;
  const candidatePayload: Record<string, unknown> = { ...payload };
  const candidateFields = options.fields.filter(fieldName => {
    if (!Object.prototype.hasOwnProperty.call(candidatePayload, fieldName)) return false;
    return looksLikeUploadEnvelope(candidatePayload[fieldName]);
  });
  if (!candidateFields.length) return candidatePayload;

  return await normalizeAttachmentFieldsInPayload(store, candidatePayload, {
    operation: 'update',
    ownerModel: options.ownerModel,
    ownerRecordId: options.ownerRecordId,
    fields: candidateFields,
  });
}

export function createFormController(store: WebModelStore<any>): IFormViewController {
  const vm = reactive<FormViewModel>({
    mode: 'display',
    draft: null,
    original: null,
    loading: false,
    error: null,
    result: null,
  });

  const aborts = createAbortableRequests();
  let loadSeq = 0;

  async function beginDisplay(recordId: string) {
    const seq = ++loadSeq;
    vm.loading = true;
    vm.error = null;
    vm.mode = 'display';

    // Handoff Optimization: Check if we have a fresh object passed from a previous action
    const cached = flashRead(recordId);
    if (cached) {
      vm.original = clone(cached);
      vm.draft = clone(cached);
      // Construct synthetic snapshot for consistency
      const snap: DataSetSnapshot = {
        kind: 'search',
        rows: [{ kind: 'record', key: recordId, payload: cached, raw: cached } as RecordRow],
        total: 1,
        ts: Date.now(),
      };
      vm.result = snap;
      store.state.result = snap;
      vm.loading = false;
      return cached;
    }

    try {
      // Wait once for field registration so the request can use a reduced field selection.
      try {
        if (!exportFieldSelection((store as any).storeId)?.length) {
          await awaitFieldSelection(store as any, { maxTries: 5, requireNonEmpty: false });
        }
      } catch {}
      const ctx = buildBrowseContext(store, recordId);
      const bundle = buildPlan(ctx);
      const snap = await aborts.execute('form.load', async signal => {
        const r = await execute(bundle, store, 'form', { signal });
        if (signal.aborted) throw new CancellationError();
        return r;
      });
      const payload = (snap.rows?.[0] as RecordRow | undefined)?.payload ?? null;
      if (seq !== loadSeq) throw new CancellationError('Superseded');
      vm.original = payload ? clone(payload) : null;
      vm.draft = payload ? clone(payload) : null;
      vm.result = snap;
      store.state.result = snap;
      return payload;
    } catch (e) {
      if (isCancellation(e)) return; // swallow
      vm.error = e;
      throw e;
    } finally {
      if (seq === loadSeq) vm.loading = false;
    }
  }

  function beginEdit() {
    if (!vm.original) return;
    vm.mode = 'edit';
    vm.draft = clone(vm.original);
  }

  /**
   * Enter create mode and prefetch server DefaultGet into the draft (PR-FD-4 / D11).
   * Opens immediately with seed; merges server defaults with seed priority; failures keep seed.
   */
  async function beginCreate(initial?: any): Promise<void> {
    const seq = ++loadSeq;
    const seed = clone(initial || {});
    vm.mode = 'create';
    vm.original = null;
    vm.draft = seed;
    vm.result = null;
    vm.error = null;

    try {
      if (typeof (store as any).DefaultGet !== 'function') return;
      const defaults = await (store as any).DefaultGet(seed);
      if (seq !== loadSeq) return;

      const server =
        defaults && typeof defaults === 'object' && !Array.isArray(defaults)
          ? ({ ...(defaults as object) } as Record<string, unknown>)
          : {};
      // Seed wins (D2). `clone` drops undefined keys, so explicit null / values are preserved.
      vm.draft = { ...server, ...(seed as Record<string, unknown>) };
    } catch (e) {
      if (seq !== loadSeq) return;
      console.warn('[FormController] DefaultGet prefetch failed', e);
      vm.draft = seed;
    }
  }

  function reset() {
    vm.draft = vm.original ? clone(vm.original) : null;
  }

  async function validate(): Promise<{ valid: boolean; errors: any[] }> {
    return { valid: true, errors: [] };
  }

  async function submit(): Promise<any> {
    vm.loading = true;
    vm.error = null;
    try {
      const { valid, errors } = await validate();
      if (!valid) throw { message: 'Validation failed', errors };
      const payload = toRaw(vm.draft) as Record<string, unknown>;
      let result: any = null;
      let newId: string | undefined;
      const ownerModel = resolveOwnerModel(store);
      const attachmentFields = attachmentFieldNames(store);

      // Use the currently registered view fields to request fresh data from the backend.
      const rawPaths = exportFieldSelection(store.storeId);
      const returnFields = ensureRootId(pathsToFieldSelection(rawPaths));

      if (vm.mode === 'create') {
        const normalizedPayload = await normalizeAttachmentFieldsInPayload(store, payload, {
          operation: 'create',
          ownerModel,
          fields: attachmentFields,
        });

        // Create(payload, returnFields) -> T
        const createRes = await store.Create(normalizedPayload, returnFields);
        result = createRes;

        if (createRes && typeof createRes === 'object') {
          // Reuse the returned record directly instead of querying again.
          const record = createRes as any;
          newId = record.Id ?? record.id;
          if (newId) {
            vm.original = clone(record);
            vm.draft = clone(record);
            // The outer route decides when to leave create mode, but the data is ready now.
            handoffCache.set(String(newId), record);
          }
        } else if (typeof createRes === 'string') {
          // Compatibility fallback for the legacy id-only response shape.
          newId = String(createRes);
          if (newId) {
            const draftWithId = { ...(normalizedPayload || {}), Id: newId };
            vm.original = clone(draftWithId);
            vm.draft = clone(draftWithId);
          }
        }
      } else {
        const id = vm.original?.Id ?? vm.original?.id;
        const ownerRecordId = normalizeText(id);
        if (!ownerRecordId) {
          throw new Error('Update failed: record id is empty.');
        }

        const original = toRaw(vm.original);
        const fieldsMeta = (store as any)?.fieldsMetadata;
        const draftForDiff = await preNormalizeDraftForDiff(store, payload || {}, {
          ownerModel,
          ownerRecordId,
          fields: attachmentFields,
        });
        const patch = buildUpdatePayload(original || {}, draftForDiff || {}, fieldsMeta) as Record<string, unknown>;
        const normalizedPatch = await normalizeAttachmentFieldsInPayload(store, patch || {}, {
          operation: 'update',
          ownerModel,
          ownerRecordId,
          fields: attachmentFields,
        });
        const touchedAttachmentFields = hasAttachmentFieldMutation(normalizedPatch, attachmentFields);

        // UpdateById(id, patch, returnFields) -> Partial<T>
        // Use UpdateById and apply the returned object directly.
        const updateRes = await store.UpdateById(id, normalizedPatch, returnFields);
        result = updateRes;

        if (updateRes && typeof updateRes === 'object') {
          // Merge updated fields into original and draft state.
          const updated = { ...original, ...updateRes };
          vm.original = clone(updated);
          vm.draft = clone(updated);
          vm.mode = 'display';
          const recordId = updated.Id ?? updated.id;
          if (recordId) {
            if (touchedAttachmentFields) {
              try {
                await beginDisplay(String(recordId));
              } catch (refreshError) {
                console.warn('[FormController] post-submit attachment refresh failed', refreshError);
                vm.error = null;
                vm.original = clone(updated);
                vm.draft = clone(updated);
                vm.mode = 'display';
              }
            }
            handoffCache.set(String(recordId), (vm.original as any) || updated);
          }
        } else {
          // Fallback to a reload when the backend does not return an updated object.
          newId = id ? String(id) : undefined;
          if (newId) {
            await beginDisplay(String(newId));
            vm.mode = 'display';
          }
        }
      }
      return result;
    } catch (e) {
      vm.error = e;
      throw e;
    } finally {
      vm.loading = false;
    }
  }

  async function remove(): Promise<any> {
    vm.loading = true;
    vm.error = null;
    try {
      const id = vm.original?.Id ?? vm.original?.id;
      if (id == null) return null;
      const res = await store.DeleteById(id);
      vm.original = null;
      vm.draft = null;
      vm.mode = 'display';
      vm.result = null;
      return res;
    } catch (e) {
      vm.error = e;
      throw e;
    } finally {
      vm.loading = false;
    }
  }

  function provideToChildren(key = 'form-root') {
    const api = {
      get mode() {
        return vm.mode;
      },
      get draft() {
        return vm.draft;
      },
      get original() {
        return vm.original;
      },
      setField(path: string, value: any) {
        if (!vm.draft) vm.draft = {};
        setByPath(vm.draft, path, value);
      },
      getField(path: string) {
        return vm.draft ? getByPath(vm.draft, path) : undefined;
      },
      onChange(_cb: (nextDraft: any) => void) {
        // Reserved for a future event queue.
      },
    };
    provide(key, api);
    return api;
  }

  const api: IFormViewController = {
    vm,
    beginDisplay,
    beginEdit,
    beginCreate,
    reset,
    validate,
    submit,
    delete: remove,
    provideToChildren,
  };
  return api;
}

export type FormController = ReturnType<typeof createFormController>;

// Test hooks: lock attachment orchestration protocol without coupling to full controller lifecycle.
export const __resolveAttachmentFieldValueForTest = resolveAttachmentFieldValue;
export const __normalizeAttachmentFieldsInPayloadForTest = normalizeAttachmentFieldsInPayload;
