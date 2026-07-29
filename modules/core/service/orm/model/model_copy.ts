// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, ModelMetadata } from '../metadata';
import { MetadataStorage } from '../metadata/storage';
import { resolveOneToManyRelationConfig } from '../relation/types';
import type { FieldSelection, Insertable } from '../repository/types';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord, UnknownRecord } from '../../../utils/types';
import { getActiveCompanyId, getEnabledCompanyIds } from '../../runtime/context/scope';
import { createModel } from './model_create_service_facade';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';
import { ReadOperations } from './model_read';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';

/** Max nested OneToMany depth for Copy (design §6.1). */
export const COPY_MAX_RELATION_DEPTH = 8;

/** Company ownership rewrite mode for Copy (company-field-design D10). */
export type CopyCompanyMode = 'keep' | 'active';

/** Optional Copy behavior beyond defaults overrides. */
export type CopyOptions = {
  /**
   * How to set the isolated model's ownership field (`companyField`).
   * - `keep` (default): preserve source ownership unless it is out of enabled scope.
   * - `active`: rewrite ownership to the request `activeCompanyId`.
   */
  copyCompany?: CopyCompanyMode;
};

type CopyWalkState = {
  ancestorIds: Set<string>;
  depth: number;
};

function isAttachmentType(type: unknown): boolean {
  const normalized = String(type || '')
    .trim()
    .toLowerCase();
  return normalized === 'binary' || normalized === 'image';
}

function extractRelationId(value: unknown): string | null {
  if (value == null) return null;
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed || null;
  }
  if (typeof value === 'number' || typeof value === 'bigint') {
    return String(value);
  }
  const record = asObjectRecord(value);
  if (!record) return null;
  const id = record.Id ?? record.id;
  if (typeof id === 'string' && id.trim()) return id.trim();
  if (typeof id === 'number' || typeof id === 'bigint') return String(id);
  return null;
}

function normalizeOwnershipValue(value: unknown): string {
  if (value == null) return '';
  if (typeof value === 'object') {
    return extractRelationId(value) || '';
  }
  return String(value).trim();
}

/**
 * Apply D10 ownership rewrite using the model's `companyField`.
 * Defaults that already set the ownership key win over keep/active.
 */
export function applyCopyCompanyOwnership(
  meta: ModelMetadata,
  out: UnknownRecord,
  defaults?: Partial<Record<string, unknown>>,
  options?: CopyOptions
): void {
  const ownershipField = String(meta.companyField ?? '').trim();
  if (!ownershipField) return;

  if (defaults && Object.prototype.hasOwnProperty.call(defaults, ownershipField) && defaults[ownershipField] !== undefined) {
    out[ownershipField] = defaults[ownershipField];
    return;
  }

  const mode: CopyCompanyMode = options?.copyCompany === 'active' ? 'active' : 'keep';
  const enabled = getEnabledCompanyIds();
  const active = getActiveCompanyId() ?? (enabled.length === 1 ? enabled[0] : undefined);
  const sourceNormalized = normalizeOwnershipValue(out[ownershipField]);
  const outOfEnabled = Boolean(sourceNormalized && enabled.length > 0 && !enabled.includes(sourceNormalized));

  if (mode === 'active' || outOfEnabled) {
    if (!active) {
      throw new Error('[Copy] missing ctx.activeCompanyId for copyCompany rewrite');
    }
    out[ownershipField] = active;
  }
}

/**
 * Whether a field should be omitted from Copy payloads (D4/D5/D7).
 * `copy === false`, primary keys, SqlCompute, non-stored compute/related are skipped.
 */
export function shouldSkipFieldForCopy(meta: ModelMetadata, fieldName: string, field: FieldMetadata): boolean {
  if (field.copy === false) return true;
  if (field.column?.primaryKey === true) return true;
  if (meta.sqlComputeHandlers?.has(fieldName)) return true;

  const compute = meta.computeHandlers?.get(fieldName);
  if (compute?.store === false) return true;

  if (field.related && field.related.store !== true) return true;

  return false;
}

function getTargetModelMetadata(field: FieldMetadata): ModelMetadata | undefined {
  const relation = asObjectRecord(field.relation);
  const targetModel = relation?.targetModel;
  if (typeof targetModel !== 'function') return undefined;
  try {
    return MetadataStorage.instance.getModelMetadata(targetModel() as RuntimeModelCtor);
  } catch {
    return undefined;
  }
}

/**
 * Build Browse field selection covering scalars, attachments, and copyable relations.
 */
export function buildCopyBrowseSelection(meta: ModelMetadata, depth = 0): FieldSelection<BaseModel> {
  const selection: unknown[] = ['*'];

  for (const [fieldName, field] of meta.fields || []) {
    if (shouldSkipFieldForCopy(meta, fieldName, field)) continue;

    if (isAttachmentType(field.type)) {
      selection.push(fieldName);
      continue;
    }

    if (field.type === 'ManyToOne') {
      // FK column + minimal nested Id for object-shaped reads.
      selection.push(fieldName);
      selection.push({ [fieldName]: ['Id'] });
      continue;
    }

    if (field.type === 'ManyToMany') {
      selection.push({ [fieldName]: ['Id'] });
      continue;
    }

    if (field.type === 'OneToMany') {
      if (depth >= COPY_MAX_RELATION_DEPTH) continue;
      const childMeta = getTargetModelMetadata(field);
      if (!childMeta) continue;
      selection.push({ [fieldName]: buildCopyBrowseSelection(childMeta, depth + 1) });
    }
  }

  return selection as FieldSelection<BaseModel>;
}

function copyScalarOrManyToOneValue(value: unknown): unknown {
  if (value == null) return null;
  // Nested Browse objects must resolve to FK id or null — never pass raw objects into Create.
  if (typeof value === 'object') {
    return extractRelationId(value);
  }
  return value;
}

function copyManyToManyIds(value: unknown): string[] | undefined {
  if (value == null) return undefined;
  if (!Array.isArray(value)) {
    const single = extractRelationId(value);
    return single ? [single] : undefined;
  }
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    const id = extractRelationId(item);
    if (!id || seen.has(id)) continue;
    seen.add(id);
    ids.push(id);
  }
  return ids;
}

/**
 * Transform a Browse row into a Create payload for Copy (design §5–§6).
 */
export function buildCopyValues(
  ModelCtor: RuntimeModelCtor,
  row: ObjectRecord,
  defaults?: Partial<Record<string, unknown>>,
  state?: CopyWalkState,
  options?: CopyOptions
): UnknownRecord {
  const sourceRecord = asObjectRecord(row);
  if (!sourceRecord) {
    throw new Error('[Copy] source row must be a non-null object');
  }

  const meta = getModelRuntimeMetadata(ModelCtor);
  const walk: CopyWalkState = state || {
    ancestorIds: new Set<string>(),
    depth: 0,
  };

  const sourceId = extractRelationId(sourceRecord.Id ?? sourceRecord.id);
  if (sourceId) walk.ancestorIds.add(sourceId);

  const out: UnknownRecord = {};

  for (const [fieldName, field] of meta.fields || []) {
    if (shouldSkipFieldForCopy(meta, fieldName, field)) continue;
    if (!(fieldName in sourceRecord)) continue;

    const raw = sourceRecord[fieldName];
    if (raw === undefined) continue;

    if (field.type === 'OneToMany') {
      if (walk.depth >= COPY_MAX_RELATION_DEPTH) {
        throw new Error(`[Copy] OneToMany depth exceeded (${COPY_MAX_RELATION_DEPTH}) at ${meta.fullModelName || meta.modelName || ModelCtor.name}.${fieldName}`);
      }
      if (!Array.isArray(raw) || raw.length === 0) continue;

      const o2m = resolveOneToManyRelationConfig(field.relation);
      if (!o2m) continue;
      const childCtor = o2m.targetModel() as RuntimeModelCtor;
      const childValues: UnknownRecord[] = [];

      for (const child of raw) {
        const childRow = asObjectRecord(child);
        if (!childRow) continue;
        const childId = extractRelationId(childRow.Id ?? childRow.id);
        if (childId && walk.ancestorIds.has(childId)) {
          throw new Error(`[Copy] cyclic OneToMany detected at ${fieldName} (source Id ${childId})`);
        }

        const childCopied = buildCopyValues(
          childCtor,
          childRow,
          undefined,
          {
            ancestorIds: new Set(walk.ancestorIds),
            depth: walk.depth + 1,
          },
          options
        );
        // Parent Create binds the new parent Id; drop inverse FK from the child payload.
        delete childCopied[o2m.inverseField];
        childValues.push(childCopied);
      }

      if (childValues.length) {
        out[fieldName] = { create: childValues };
      }
      continue;
    }

    if (field.type === 'ManyToMany') {
      const ids = copyManyToManyIds(raw);
      if (ids && ids.length) out[fieldName] = ids;
      continue;
    }

    if (field.type === 'ManyToOne') {
      out[fieldName] = copyScalarOrManyToOneValue(raw);
      continue;
    }

    // Scalars, Refs, attachment binding ids, etc.
    out[fieldName] = raw;
  }

  applyCopyCompanyOwnership(meta, out, defaults, options);

  if (defaults && typeof defaults === 'object') {
    for (const [key, value] of Object.entries(defaults)) {
      if (value === undefined) continue;
      out[key] = value;
    }
  }

  return out;
}

/**
 * Duplicate one record via Browse → buildCopyValues → Create (design D2).
 */
export async function copyModel<T extends BaseModel>(
  ModelCtor: RuntimeModelCtor<T>,
  id: string,
  defaults?: Partial<Record<string, unknown>>,
  options?: CopyOptions
): Promise<T> {
  const trimmedId = String(id || '').trim();
  if (!trimmedId) {
    throw new Error('[Copy] id is required');
  }

  const meta = getModelRuntimeMetadata(ModelCtor);
  const fields = buildCopyBrowseSelection(meta);
  const row = (await ReadOperations.Browse(ModelCtor, trimmedId, fields as FieldSelection<T>)) as ObjectRecord;
  const values = buildCopyValues(
    ModelCtor as RuntimeModelCtor,
    row,
    defaults,
    {
      ancestorIds: new Set([trimmedId]),
      depth: 0,
    },
    options
  );

  return (await createModel(ModelCtor, values as Partial<Insertable<T & BaseModel>>)) as T;
}
