// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate, isTermReference, type TermReference } from '../../i18n';
import { MetadataStorage } from '../metadata/storage';
import type { FieldMetadata, SelectionItem } from '../metadata/field';
import { getModelRepository } from './model_internal_facade';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';

/** Presentation slice returned by FieldsGet (aligned with FE WebFieldMetadata). */
export type FieldsGetFieldMeta = {
  type: string;
  string?: string;
  stringText?: TermReference;
  selection?: Array<{ value: string; label: string }>;
  notNull?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
  isReadonly?: boolean;
  indexed?: boolean;
  [key: string]: unknown;
};

type ModelFieldsGetCtor = RuntimeModelCtor<BaseModel>;

function sortedUniqueStrings(values: string[] | undefined): string[] | undefined {
  if (!values) return undefined;
  const out = [...new Set(values.map(v => String(v || '').trim()).filter(Boolean))];
  out.sort();
  return out;
}

function translateSrc(module: string, src: string, scope?: string): string {
  const trimmed = String(src || '');
  if (!trimmed) return '';
  const { _t } = createTranslate(module, scope ? { scope } : undefined);
  return _t(trimmed);
}

function translateTermReference(ref: TermReference): string {
  const { _t } = createTranslate(ref.module, { scope: ref.scope, kind: ref.kind });
  return _t(ref.src);
}

function translateFieldString(
  field: FieldMetadata,
  fallbackModule: string,
  fallbackScope: string
): string | undefined {
  const msgid = typeof field.string === 'string' ? field.string.trim() : '';
  if (!msgid && !field.stringText) return undefined;
  if (field.stringText && isTermReference(field.stringText)) {
    return translateTermReference(field.stringText);
  }
  if (!msgid) return undefined;
  return translateSrc(fallbackModule, msgid, fallbackScope);
}

function translateSelectionLabels(
  items: readonly SelectionItem[] | undefined,
  fallbackModule: string,
  fallbackScope: string
): Array<{ value: string; label: string }> | undefined {
  if (!Array.isArray(items) || items.length === 0) return undefined;
  const out: Array<{ value: string; label: string }> = [];
  for (const item of items) {
    if (!item || typeof item !== 'object') continue;
    const value = String(item.value || '').trim();
    if (!value) continue;
    let label = '';
    if (item.labelText && isTermReference(item.labelText)) {
      label = translateTermReference(item.labelText);
    } else {
      label = translateSrc(fallbackModule, String(item.label || ''), fallbackScope);
    }
    out.push({ value, label: label || String(item.label || value) });
  }
  return out.length ? out : undefined;
}

async function resolveDenyReadFields(ModelCtor: ModelFieldsGetCtor): Promise<Set<string>> {
  try {
    const repo = getModelRepository(ModelCtor);
    if (!repo || typeof repo.getDenyReadFields !== 'function') {
      return new Set();
    }
    const spec = await repo.getDenyReadFields();
    const deny = Array.isArray(spec?.denyReadFields) ? spec.denyReadFields : [];
    return new Set(deny.map(name => String(name || '').trim()).filter(Boolean));
  } catch {
    return new Set();
  }
}

function buildFieldMeta(
  field: FieldMetadata,
  fallbackModule: string,
  fallbackScope: string
): FieldsGetFieldMeta {
  const meta: FieldsGetFieldMeta = {
    type: String(field.type || ''),
  };

  const translated = translateFieldString(field, fallbackModule, fallbackScope);
  if (translated !== undefined) {
    meta.string = translated;
  } else if (typeof field.string === 'string' && field.string.trim()) {
    meta.string = field.string.trim();
  }
  if (field.stringText && isTermReference(field.stringText)) {
    meta.stringText = { ...field.stringText };
  }

  // P1: static selection only. Callable / dynamic evaluation is P3.
  const selection = translateSelectionLabels(field.selection, fallbackModule, fallbackScope);
  if (selection) {
    meta.selection = selection;
  }

  const column = field.column as { notNull?: boolean; size?: number; precision?: number; scale?: number; index?: boolean | string } | undefined;
  if (column?.notNull === true) meta.notNull = true;
  if (typeof column?.size === 'number') meta.size = column.size;
  if (typeof column?.precision === 'number') meta.precision = column.precision;
  if (typeof column?.scale === 'number') meta.scale = column.scale;
  if (column?.index !== undefined && column.index !== false) meta.indexed = true;

  return meta;
}

function projectAttributes(meta: FieldsGetFieldMeta, attributes: string[] | undefined): FieldsGetFieldMeta {
  if (!attributes || attributes.length === 0) {
    return meta;
  }
  const keep = new Set(attributes.map(a => String(a || '').trim()).filter(Boolean));
  keep.add('type');
  const projected: FieldsGetFieldMeta = { type: meta.type };
  for (const key of Object.keys(meta)) {
    if (keep.has(key)) {
      projected[key] = meta[key];
    }
  }
  return projected;
}

/**
 * Build request-scoped field presentation metadata (translated titles / selection labels).
 *
 * - Filters deny-read fields (D11).
 * - Unknown `fields` names are omitted.
 * - `attributes` narrows keys but always keeps `type`.
 */
export async function fieldsGetModels(
  ModelCtor: ModelFieldsGetCtor,
  fields?: string[],
  attributes?: string[]
): Promise<Record<string, FieldsGetFieldMeta>> {
  const modelMeta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  const allFields = modelMeta?.fields;
  if (!allFields || allFields.size === 0) {
    return {};
  }

  const denyRead = await resolveDenyReadFields(ModelCtor);
  const application = String(modelMeta.application || 'application').trim() || 'application';
  const modelName = String(modelMeta.modelName || modelMeta.name || ModelCtor.name || 'Model').trim();
  const fallbackScope = `${application}.model.${modelName}.fields`;

  const requested = sortedUniqueStrings(fields);
  const attrList = sortedUniqueStrings(attributes);

  const names: string[] = [];
  if (requested && requested.length > 0) {
    for (const name of requested) {
      if (!allFields.has(name)) continue;
      if (denyRead.has(name)) continue;
      names.push(name);
    }
  } else {
    for (const name of allFields.keys()) {
      if (denyRead.has(name)) continue;
      names.push(name);
    }
    names.sort();
  }

  const result: Record<string, FieldsGetFieldMeta> = {};
  for (const name of names) {
    const field = allFields.get(name);
    if (!field) continue;
    result[name] = projectAttributes(buildFieldMeta(field, application, fallbackScope), attrList);
  }
  return result;
}
