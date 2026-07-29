// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, ModelMetadata } from '../../metadata';
import { isDecimalLikeField } from '../../metadata/decimal_like';
import { getActiveCompanyId, getCtxValue } from '../../../runtime/context/scope';
import { isBigdecimalEnvelope, isDecimal, normalizeDecimalByMeta } from '@/core/utils/decimal';
import { asObjectRecord, hasOwnKey } from '../../../../utils/object';
import type { Entity } from '../types';
import type { ObjectRecord, UnknownRecord } from '../../../../utils/types';
import { sanitizeHtmlForWrite } from '../../utils/html_sanitize';

export type CompanyDependentWriteMode = 'create' | 'update';

export type CompanyValueMap = Record<string, unknown>;

function hasOwn(map: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(map, key);
}

function maybeJsonFast(str: string): boolean {
  const c = str.charCodeAt(0);
  return c === 123 || c === 91 || c === 34 || c === 45 || (c >= 48 && c <= 57) || c === 116 || c === 102 || c === 110;
}

function parseJsonObjectLike(v: unknown): unknown {
  if (typeof v === 'string') {
    const s = v.trim();
    if (!s) return {};
    if (!maybeJsonFast(s)) return s;
    try {
      return JSON.parse(s);
    } catch {
      return s;
    }
  }
  return v;
}

/**
 * Object-shaped scalars that must not be treated as company maps:
 * Date, Decimal, $bigdecimal envelopes, and ManyToOne `{ Id }` payloads.
 */
export function isCompanyDependentScalarEnvelope(value: unknown): boolean {
  if (value == null || typeof value !== 'object' || Array.isArray(value)) return false;
  if (value instanceof Date) return true;
  if (isDecimal(value) || isBigdecimalEnvelope(value)) return true;
  // Arrays already rejected; remaining objects are always object-records.
  const record = value as Record<string, unknown>;
  if (!hasOwnKey(record, 'Id')) return false;
  // Relation / browse envelopes: Id plus optional display fields.
  return Object.keys(record).every(key => key === 'Id' || key === 'DisplayName' || key === 'Name');
}

/** Normalize a scalar (including envelopes) for storage under one company key. */
export function normalizeCompanyDependentScalarValue(value: unknown, fm?: FieldMetadata): unknown {
  if (value == null) return value;
  if (value instanceof Date) return value.toISOString();
  if (isBigdecimalEnvelope(value)) {
    if (fm && isDecimalLikeField(fm)) {
      const d = normalizeDecimalByMeta(fm, value.$bigdecimal);
      return d ? d.toString() : String(value.$bigdecimal);
    }
    return String(value.$bigdecimal);
  }
  if (isDecimal(value)) {
    if (fm && isDecimalLikeField(fm)) {
      const d = normalizeDecimalByMeta(fm, value);
      return d ? d.toString() : value.toString();
    }
    return value.toString();
  }
  const record = asObjectRecord(value);
  if (record && hasOwnKey(record, 'Id')) {
    return record.Id ?? null;
  }
  if (fm && isDecimalLikeField(fm) && (typeof value === 'string' || typeof value === 'number')) {
    const d = normalizeDecimalByMeta(fm, value);
    return d ? d.toString() : value;
  }
  if (fm?.type === 'html') {
    return sanitizeHtmlForWrite(value);
  }
  return value;
}

/**
 * Resolve the company id used for companyDependent unwrap/wrap.
 * Prefer activeCompanyId from request / withCompany.
 */
export function resolveCompanyDependentCompanyId(): string | undefined {
  const id = getActiveCompanyId();
  return id || undefined;
}

/** When true, decode returns the full company map instead of an unwrapped scalar. */
export function getPrefetchCompanies(): boolean {
  const raw = getCtxValue('prefetch_companies') ?? getCtxValue('prefetchCompanies');
  return raw === true;
}

/** When true, object writes replace the whole company map instead of merging keys. */
export function getCompanyDependentWriteReplace(): boolean {
  const raw = getCtxValue('company_write_replace') ?? getCtxValue('companyWriteReplace');
  return raw === true;
}

export function assertCompanyDependentCompanyKey(companyId: string, fieldName: string): string {
  const trimmed = String(companyId || '').trim();
  if (!trimmed) {
    throw new Error(`Company-dependent field "${fieldName}" requires a non-empty company id`);
  }
  return trimmed;
}

export function isCompanyValueMap(value: unknown): value is CompanyValueMap {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  if (isCompanyDependentScalarEnvelope(value)) return false;
  return true;
}

export function parseCompanyDependentStoredMap(value: unknown): CompanyValueMap | null {
  if (value == null) return null;
  const parsed = parseJsonObjectLike(value);
  if (!isCompanyValueMap(parsed)) return null;
  const out: CompanyValueMap = {};
  for (const [k, v] of Object.entries(parsed)) {
    const key = String(k || '').trim();
    if (!key) continue;
    out[key] = v;
  }
  return Object.keys(out).length ? out : null;
}

/**
 * Unwrap a stored company map for the active company (F0: missing key → null).
 */
export function unwrapCompanyDependentValue(map: CompanyValueMap | null | undefined, companyId: string): unknown {
  if (map == null || !isCompanyValueMap(map)) return null;
  if (!hasOwn(map, companyId)) return null;
  return map[companyId];
}

export function decodeCompanyDependentFieldValue(
  stored: unknown,
  opts?: { companyId?: string; prefetchCompanies?: boolean }
): unknown {
  const map = parseCompanyDependentStoredMap(stored);
  if (map == null) return null;
  if (opts?.prefetchCompanies ?? getPrefetchCompanies()) {
    return { ...map };
  }
  const companyId = opts?.companyId ?? resolveCompanyDependentCompanyId();
  if (!companyId) return null;
  return unwrapCompanyDependentValue(map, companyId);
}

function normalizeIncomingCompanyMap(
  fieldName: string,
  raw: CompanyValueMap,
  fm?: FieldMetadata
): CompanyValueMap {
  const out: CompanyValueMap = {};
  for (const [k, v] of Object.entries(raw)) {
    const companyId = assertCompanyDependentCompanyKey(k, fieldName);
    if (v === false) {
      throw new Error(
        `Company-dependent field "${fieldName}" does not accept false in write payloads; use UpdateFieldCompanyValues to delete a company key`
      );
    }
    out[companyId] = normalizeCompanyDependentScalarValue(v, fm);
  }
  return out;
}

/**
 * Delete one company key from a stored map. Empty map → null (column NULL).
 */
export function deleteCompanyKey(map: CompanyValueMap, companyId: string, fieldName: string): CompanyValueMap | null {
  const key = assertCompanyDependentCompanyKey(companyId, fieldName);
  if (!hasOwn(map, key)) {
    return Object.keys(map).length ? { ...map } : null;
  }
  const out = { ...map };
  delete out[key];
  return Object.keys(out).length ? out : null;
}

/**
 * Apply Get/UpdateFieldCompanyValues patch: value writes a key; `false` deletes a key.
 */
export function applyFieldCompanyValuesPatch(args: {
  fieldName: string;
  currentMap: CompanyValueMap | null;
  values: Record<string, unknown | false>;
  fieldMeta?: FieldMetadata;
}): CompanyValueMap | null {
  const { fieldName } = args;
  if (!args.values || typeof args.values !== 'object' || Array.isArray(args.values)) {
    throw new Error(`Company-dependent field "${fieldName}" values must be an object map`);
  }
  let out: CompanyValueMap = { ...(args.currentMap || {}) };
  for (const [rawKey, rawVal] of Object.entries(args.values)) {
    const companyId = assertCompanyDependentCompanyKey(rawKey, fieldName);
    if (rawVal === false) {
      const next = deleteCompanyKey(out, companyId, fieldName);
      out = next || {};
      continue;
    }
    out[companyId] = normalizeCompanyDependentScalarValue(rawVal, args.fieldMeta);
  }
  return Object.keys(out).length ? out : null;
}

/**
 * Merge a write value into the current company map.
 * - null → delete current company key (D5); empty map → null
 * - scalar / scalar envelope → set current company key
 * - object → merge keys (or replace when replace=true / ctx company_write_replace)
 * - create mode: only active company key (no base-company dual-write)
 */
export function mergeCompanyDependentWrite(args: {
  fieldName: string;
  value: unknown;
  companyId: string;
  currentMap: CompanyValueMap | null;
  mode: CompanyDependentWriteMode;
  replace?: boolean;
  fieldMeta?: FieldMetadata;
}): CompanyValueMap | null {
  const { fieldName, value, mode, fieldMeta } = args;
  const companyId = assertCompanyDependentCompanyKey(args.companyId, fieldName);
  const replace = args.replace === true;

  if (value === null) {
    // Ordinary scalar null deletes the active company key (D5).
    // Explicit replace (UpdateFieldCompanyValues / whole-map write) clears the column.
    if (replace) return null;
    return deleteCompanyKey(args.currentMap || {}, companyId, fieldName);
  }

  if (isCompanyValueMap(value)) {
    const incoming = normalizeIncomingCompanyMap(fieldName, value, fieldMeta);
    if (replace) {
      return Object.keys(incoming).length ? incoming : null;
    }
    const merged: CompanyValueMap = { ...(args.currentMap || {}), ...incoming };
    return Object.keys(merged).length ? merged : null;
  }

  if (isCompanyDependentScalarEnvelope(value)) {
    const merged: CompanyValueMap = { ...(args.currentMap || {}) };
    merged[companyId] = normalizeCompanyDependentScalarValue(value, fieldMeta);
    void mode;
    return merged;
  }

  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    const merged: CompanyValueMap = { ...(args.currentMap || {}) };
    merged[companyId] = normalizeCompanyDependentScalarValue(value, fieldMeta);
    void mode;
    return merged;
  }

  if (typeof value === 'object') {
    throw new Error(
      `Company-dependent field "${fieldName}" expects scalar, company map object, or null (got object)`
    );
  }

  const merged: CompanyValueMap = { ...(args.currentMap || {}) };
  merged[companyId] = normalizeCompanyDependentScalarValue(value, fieldMeta);
  void mode;
  return merged;
}

export function encodeCompanyDependentMapForDb(map: CompanyValueMap | null, fm?: FieldMetadata): string | null {
  if (map == null) return null;
  if (fm?.type === 'html') {
    const normalized: CompanyValueMap = {};
    for (const [k, v] of Object.entries(map)) {
      normalized[k] = sanitizeHtmlForWrite(v);
    }
    return JSON.stringify(normalized);
  }
  if (!fm || !isDecimalLikeField(fm)) {
    return JSON.stringify(map);
  }
  const normalized: CompanyValueMap = {};
  for (const [k, v] of Object.entries(map)) {
    normalized[k] = normalizeCompanyDependentScalarValue(v, fm);
  }
  return JSON.stringify(normalized);
}

/**
 * Rewrite companyDependent fields on a write payload into full company maps
 * ready for encodeForDb JSON persistence.
 */
export function applyCompanyDependentFieldsForWrite(
  meta: ModelMetadata,
  input: Entity,
  opts: { mode: CompanyDependentWriteMode; companyId?: string; current?: ObjectRecord | null; replace?: boolean }
): Entity {
  if (!input || typeof input !== 'object') return input;
  const companyId = opts.companyId ?? resolveCompanyDependentCompanyId();
  const replace = opts.replace === true || getCompanyDependentWriteReplace();
  const out: UnknownRecord = { ...(input as UnknownRecord) };
  let changed = false;

  meta.fields.forEach((fm, name) => {
    if (!fm?.companyDependent) return;
    if (!Object.prototype.hasOwnProperty.call(out, name)) return;
    const raw = out[name];
    if (raw === undefined) return;

    if (!companyId) {
      throw new Error(`Company-dependent field "${name}" write requires an active company id`);
    }

    const currentStored = opts.current ? opts.current[name] : undefined;
    const currentMap =
      opts.mode === 'update' && !replace ? parseCompanyDependentStoredMap(currentStored) : null;

    const merged = mergeCompanyDependentWrite({
      fieldName: name,
      value: raw,
      companyId,
      currentMap,
      mode: opts.mode,
      replace,
      fieldMeta: fm,
    });
    out[name] = merged;
    changed = true;
  });

  return (changed ? out : input) as Entity;
}

export function payloadHasCompanyDependentFieldWrite(meta: ModelMetadata, input: Entity): boolean {
  if (!input || typeof input !== 'object') return false;
  for (const [name, fm] of meta.fields) {
    if (!fm?.companyDependent) continue;
    if (Object.prototype.hasOwnProperty.call(input, name) && (input as UnknownRecord)[name] !== undefined) {
      return true;
    }
  }
  return false;
}

export function fieldIsCompanyDependent(fm: FieldMetadata | undefined): boolean {
  return fm?.companyDependent === true;
}
