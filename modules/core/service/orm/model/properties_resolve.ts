// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type { FieldMetadata } from '../metadata/field';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { lookupPropertyDefinitionModel } from './properties_lookup';
import {
  filterReadablePropertyDefinitionItems,
  normalizePropertiesMap,
  type PropertyItemDefinition,
  type ResolvedPropertyItem,
} from './properties_types';

export type ResolvePropertiesOptions = {
  /** Override container id (unsaved parent change); when set, skips reading the relation from record. */
  containerId?: string | null;
};

function relationIdFromValue(value: unknown): string | null {
  if (value == null || value === false) return null;
  if (typeof value === 'string') {
    const t = value.trim();
    return t || null;
  }
  const rec = asObjectRecord(value);
  if (!rec) return null;
  const id = rec.Id ?? rec.id;
  if (typeof id === 'string' && id.trim()) return id.trim();
  return null;
}

function buildDefinitionSearchCondition(
  targetModel: string,
  propertiesField: string,
  mode: 'app' | 'parent',
  containerModel: string | undefined,
  containerId: string | null
): ObjectRecord {
  const And: unknown[] = [
    ['TargetModel', '=', targetModel],
    ['PropertiesField', '=', propertiesField],
  ];
  if (mode === 'app') {
    And.push(['ContainerId', '=', null]);
  } else {
    if (containerModel) {
      And.push(['ContainerModel', '=', containerModel]);
    }
    And.push(['ContainerId', '=', containerId]);
  }
  return { And };
}

async function loadDefinitionItems(
  application: string,
  targetModel: string,
  propertiesField: string,
  mode: 'app' | 'parent',
  containerModel: string | undefined,
  containerId: string | null
): Promise<PropertyItemDefinition[]> {
  const Ctor = lookupPropertyDefinitionModel(application);
  if (!Ctor) {
    console.warn(`PROPERTY_DEFINITION_MODEL_MISSING app=${application}`);
    return [];
  }
  // Propagate Search failures (do not mask as empty schema → PROPERTIES_WRITE_NO_SCHEMA).
  // Stable orderBy keeps limit:1 deterministic when duplicates exist before uniqueness DDL lands.
  const rows = await Ctor.Search(buildDefinitionSearchCondition(targetModel, propertiesField, mode, containerModel, containerId), {
    fields: ['Id', 'Definition', 'TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'] as any,
    orderBy: [{ field: 'Id', order: 'asc' }],
    limit: 1,
  } as any);
  const row = rows && rows[0];
  if (!row) return [];
  return filterReadablePropertyDefinitionItems(row.Definition, item => {
    console.warn(
      `PROPERTY_DEFINITION_UNKNOWN_TYPE skipped name=${item.name} type=${item.type} app=${application}`
    );
  });
}

function resolveContainerModelName(containerFieldMeta: FieldMetadata | undefined): string | undefined {
  if (!containerFieldMeta?.relation) return undefined;
  const relation = containerFieldMeta.relation as { targetModel?: unknown };
  const targetModel = relation.targetModel;
  if (typeof targetModel === 'function') {
    try {
      const ctor = targetModel();
      const meta = MetadataStorage.instance.getModelMetadata(ctor as any);
      return String(meta?.modelName || '').trim() || undefined;
    } catch {
      return undefined;
    }
  }
  if (typeof targetModel === 'string') {
    // ManyToOneRef full name "app.Model" → short name after last dot
    const trimmed = targetModel.trim();
    if (!trimmed) return undefined;
    const parts = trimmed.split('.');
    return parts[parts.length - 1] || trimmed;
  }
  return undefined;
}

/**
 * Merge effective schema ⊕ value map into property items (PP3).
 * Browse never attaches this — Form / callers invoke explicitly.
 */
export async function resolveProperties(
  ModelCtor: RuntimeModelCtor<BaseModel>,
  record: ObjectRecord | BaseModel | null | undefined,
  fieldName: string,
  opts?: ResolvePropertiesOptions
): Promise<ResolvedPropertyItem[]> {
  const meta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  const fm = meta.fields.get(fieldName);
  if (!fm || fm.type !== 'properties') {
    return [];
  }

  const application = String(meta.application || '').trim();
  const targetModel = String(meta.modelName || '').trim();
  if (!application || !targetModel) return [];

  const row = (record || {}) as ObjectRecord;
  const valueMap = normalizePropertiesMap(row[fieldName]);

  const definitionKey = typeof fm.definition === 'string' ? fm.definition.trim() : '';
  let items: PropertyItemDefinition[];

  if (!definitionKey) {
    // App-level container
    items = await loadDefinitionItems(application, targetModel, fieldName, 'app', undefined, null);
  } else {
    // Parent-record container (PP2: empty parent id → [] ; no App-level fallback)
    let containerId: string | null;
    if (opts && Object.prototype.hasOwnProperty.call(opts, 'containerId')) {
      const raw = opts.containerId;
      containerId = raw == null || raw === '' ? null : String(raw).trim() || null;
    } else {
      containerId = relationIdFromValue(row[definitionKey]);
    }
    if (!containerId) {
      return [];
    }
    const containerFm = meta.fields.get(definitionKey);
    const containerModel = resolveContainerModelName(containerFm);
    items = await loadDefinitionItems(application, targetModel, fieldName, 'parent', containerModel, containerId);
  }

  return items.map(item => {
    const resolved: ResolvedPropertyItem = { ...item };
    if (Object.prototype.hasOwnProperty.call(valueMap, item.name)) {
      resolved.value = valueMap[item.name];
    }
    return resolved;
  });
}

/**
 * Load effective schema items only (no values). Used by write validation / DefaultGet.
 */
export async function loadEffectivePropertySchema(
  ModelCtor: RuntimeModelCtor<BaseModel>,
  fieldName: string,
  rowCtx: ObjectRecord,
  opts?: ResolvePropertiesOptions
): Promise<PropertyItemDefinition[]> {
  const resolved = await resolveProperties(ModelCtor, rowCtx, fieldName, opts);
  return resolved.map(({ value: _v, ...rest }) => rest);
}
