// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Entity } from '../types';
import type { FieldMetadata, ModelMetadata } from '../../metadata';
import { DEC_SCALE_ALIAS_PREFIX, buildHiddenScaleAlias } from '../hidden_scale_alias';
import { asBigdecimal, isBigdecimalEnvelope, isDecimal, normalizeDecimalByMeta } from '@/core/utils/decimal';
import { asObjectRecord, hasOwnKey } from '../../../../utils/object';
import type { UnknownRecord } from '../../../../utils/types';

type DecimalMetaLike = {
  column?: UnknownRecord;
  select?: UnknownRecord;
};

function getFieldSpec(fm: FieldMetadata | undefined): UnknownRecord {
  return asObjectRecord(fm?.column) ?? asObjectRecord(fm?.select) ?? {};
}

function normalizedModelIdentity(meta: ModelMetadata): string {
  const full = String(meta.fullModelName || '').trim();
  if (full) return full;
  const app = String((meta as { application?: string })?.application || '').trim();
  const model = String(meta.modelName || meta.name || meta.className || meta.type?.name || '').trim();
  if (app && model) return `${app}.${model}`;
  return model;
}

function isStorageBlobCarrierModel(meta: ModelMetadata): boolean {
  const identity = normalizedModelIdentity(meta);
  if (
    identity === 'document.AttachmentObject' ||
    identity === 'document.UploadSession' ||
    identity === 'document.AttachmentContent' ||
    identity === 'document.AttachmentUploadSession' ||
    identity === 'document.StoredContent'
  ) {
    return true;
  }

  const app = String((meta as { application?: string })?.application || '')
    .trim()
    .toLowerCase();
  const model = String(meta.modelName || meta.name || '')
    .trim()
    .toLowerCase();
  return (
    (app === 'storage' && (model === 'attachmentobject' || model === 'uploadsession')) ||
    (app === 'document' && (model === 'attachmentcontent' || model === 'attachmentuploadsession' || model === 'storedcontent'))
  );
}

function shouldPersistPhysicalField(meta: ModelMetadata, fm: FieldMetadata | undefined): boolean {
  if (!fm?.column) return false;
  const type = String(fm.type || '')
    .trim()
    .toLowerCase();
  if (type !== 'binary' && type !== 'image') {
    return true;
  }
  return isStorageBlobCarrierModel(meta);
}

function toDecimalMetaWithScale(fm: FieldMetadata | undefined, scale: number | undefined): FieldMetadata | DecimalMetaLike | undefined {
  if (scale == null) return fm;
  return { column: { ...getFieldSpec(fm), scale } };
}

function maybeJsonFast(str: string): boolean {
  if (!str) return false;
  const c = str.charCodeAt(0);
  return c === 123 || c === 91 || c === 34 || c === 45 || (c >= 48 && c <= 57) || c === 116 || c === 102 || c === 110;
}

export function parseJsonObjectFieldValue(v: unknown): unknown {
  if (v == null) return {};
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
  if (typeof v === 'object') return v;
  return v;
}

export function resolveDecimalScaleForWrite(fm: FieldMetadata | undefined, input: Entity): number | undefined {
  if (!fm || fm.type !== 'decimal') return undefined;
  const spec = getFieldSpec(fm);
  if (typeof spec.scale === 'number') return spec.scale;
  const sField = spec.scaleField;
  if (typeof sField !== 'string' || !sField) return undefined;
  const val = asObjectRecord(input)?.[sField];
  if (val == null) throw new Error(`Writing "${String(fm.name)}" requires "${sField}" as scale, but it was not provided`);
  const n = Number(val);
  if (!Number.isInteger(n) || n < 0 || n > 18) {
    throw new Error(`Writing "${String(fm.name)}" has an invalid scaleField="${sField}" value (expected an integer in 0..18, got ${val})`);
  }
  return n;
}

export function resolveDecimalScaleFromRow(meta: ModelMetadata, fm: FieldMetadata | undefined, fieldName: string, row: unknown): number | undefined {
  void meta;
  if (!fm || fm.type !== 'decimal') return undefined;
  const spec = getFieldSpec(fm);
  if (typeof spec.scale === 'number') return spec.scale;

  const sField = typeof spec.scaleField === 'string' ? spec.scaleField : undefined;
  if (!sField) return undefined;

  const rowRecord = asObjectRecord(row);
  const inline = rowRecord?.[sField];
  if (inline != null && Number.isInteger(Number(inline))) return Number(inline);

  const hiddenAlias = buildHiddenScaleAlias(fieldName);
  const hidden = rowRecord?.[hiddenAlias];
  if (hidden != null && Number.isInteger(Number(hidden))) return Number(hidden);

  return undefined;
}

export function cleanupHiddenScaleKeys(row: unknown) {
  const rowRecord = asObjectRecord(row);
  if (!rowRecord) return;
  for (const k of Object.keys(rowRecord)) {
    if (k.startsWith(DEC_SCALE_ALIAS_PREFIX)) {
      delete rowRecord[k];
    }
  }
}

export function encodeForDb(meta: ModelMetadata, input: Entity): Entity {
  if (!input || typeof input !== 'object') return input;

  const allowed = new Set<string>();
  meta.fields.forEach((f, k) => {
    if (shouldPersistPhysicalField(meta, f)) allowed.add(k);
  });

  const out: UnknownRecord = {};
  for (const [k, v] of Object.entries(input)) {
    if (k.startsWith('__')) continue;
    if (!allowed.has(k)) continue;
    if (v === undefined) continue;

    const fm = meta.fields.get(k);

    if (fm?.type === 'ManyToOne' && fm.column) {
      const vRecord = asObjectRecord(v);
      out[k] = vRecord && hasOwnKey(vRecord, 'Id') ? (vRecord.Id ?? null) : v;
      continue;
    }

    if (fm?.type === 'ManyToOneRef' && fm.column) {
      if (v == null) {
        out[k] = null;
      } else if (typeof v === 'string') {
        out[k] = v;
      } else {
        const vRecord = asObjectRecord(v);
        if (typeof vRecord?.Id === 'string') {
          out[k] = vRecord.Id;
        } else {
          out[k] = String(v);
        }
      }
      continue;
    }

    if (fm?.type === 'ManyToManyRef' && fm.column) {
      if (v == null) {
        out[k] = null;
      } else if (Array.isArray(v)) {
        out[k] = v;
      } else {
        out[k] = [v];
      }
      continue;
    }

    if (fm?.type === 'jsonobject' && fm.column) {
      if (v == null) {
        out[k] = null;
      } else {
        out[k] = JSON.stringify(v);
      }
      continue;
    }

    if (fm?.type === 'decimal' && v != null) {
      try {
        const source = isBigdecimalEnvelope(v) ? v.$bigdecimal : v;
        const effScale = resolveDecimalScaleForWrite(fm, input);
        const overrideFm = toDecimalMetaWithScale(fm, effScale);

        const d = normalizeDecimalByMeta(overrideFm, source);
        out[k] = d ? { $bigdecimal: d.toString() } : asBigdecimal(v);
      } catch {
        out[k] = asBigdecimal(v);
      }
      continue;
    }

    out[k] = v;
  }

  return out;
}

export function decodeFromDb(meta: ModelMetadata, row: Entity): Entity {
  if (!row || typeof row !== 'object') return row;
  const out: UnknownRecord = { ...row };

  meta.fields.forEach((f, k) => {
    const t = f.type;
    const cur = out[k];

    if (t === 'jsonobject') {
      out[k] = parseJsonObjectFieldValue(cur);
      return;
    }

    if (t === 'decimal') {
      if (cur == null) return;

      const effScale = resolveDecimalScaleFromRow(meta, f, k, out);
      const overrideFm = toDecimalMetaWithScale(f, effScale);

      try {
        if (isDecimal(cur)) {
          const d = normalizeDecimalByMeta(overrideFm, cur);
          if (d) out[k] = d;
          return;
        }
        const source = isBigdecimalEnvelope(cur) ? cur.$bigdecimal : cur;
        const d = normalizeDecimalByMeta(overrideFm, source);
        if (d) out[k] = d;
      } catch {}
      return;
    }

    if (t === 'ManyToManyRef') {
      if (cur == null) {
        out[k] = [];
        return;
      }
      if (Array.isArray(cur)) {
        out[k] = cur.map(x => String(x));
        return;
      }
      if (typeof cur === 'string') {
        try {
          const parsed = JSON.parse(cur);
          if (Array.isArray(parsed)) {
            out[k] = parsed.map(x => String(x));
            return;
          }
        } catch {}
        out[k] = [String(cur)];
        return;
      }
      out[k] = [String(cur)];
      return;
    }
  });

  cleanupHiddenScaleKeys(out);

  return out as Entity;
}
