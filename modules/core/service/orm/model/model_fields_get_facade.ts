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
  help?: string;
  helpText?: TermReference;
  selectionKind?: 'static' | 'dynamic';
  selection?: Array<{ value: string; label: string }>;
  notNull?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
  isReadonly?: boolean;
  indexed?: boolean;
  maxUploadBytes?: number;
  maxWidth?: number;
  maxHeight?: number;
  /** false when @Field({ copy: false }); omitted means default true. */
  copy?: boolean;
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

function translateFieldHelp(
  field: FieldMetadata,
  fallbackModule: string,
  fallbackScope: string
): string | undefined {
  const msgid = typeof field.help === 'string' ? field.help.trim() : '';
  if (!msgid && !field.helpText) return undefined;
  if (field.helpText && isTermReference(field.helpText)) {
    return translateTermReference(field.helpText);
  }
  if (!msgid) return undefined;
  return translateSrc(fallbackModule, msgid, fallbackScope);
}

function translateSelectionLabels(
  items: readonly SelectionItem[] | undefined
): Array<{ value: string; label: string }> | undefined {
  if (!Array.isArray(items) || items.length === 0) return undefined;
  const out: Array<{ value: string; label: string }> = [];
  for (const item of items) {
    if (!item || typeof item !== 'object') continue;
    const value = String(item.value || '').trim();
    if (!value) continue;

    // labelText (_lt) → translate; bare string label → pass through (D5 revised).
    // Authors put _lt on declaration label; decorator / normalizeSelectionItems
    // already materialize msgid on `label` and TermReference on `labelText`.
    let label = '';
    if (item.labelText && isTermReference(item.labelText)) {
      label = translateTermReference(item.labelText);
      if (!label) label = String(item.labelText.src || item.label || value);
    } else {
      label = String(item.label || '').trim() || value;
    }
    out.push({ value, label });
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

async function resolveDenyWriteFields(ModelCtor: ModelFieldsGetCtor): Promise<Set<string>> {
  try {
    const repo = getModelRepository(ModelCtor);
    if (!repo || typeof repo.getDenyWriteFields !== 'function') {
      return new Set();
    }
    const spec = await repo.getDenyWriteFields();
    const deny = Array.isArray(spec?.denyWriteFields) ? spec.denyWriteFields : [];
    return new Set(deny.map(name => String(name || '').trim()).filter(Boolean));
  } catch {
    return new Set();
  }
}

function normalizeSelectionItems(raw: unknown): SelectionItem[] {
  if (!Array.isArray(raw)) return [];
  const out: SelectionItem[] = [];
  const seen = new Set<string>();
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue;
    const value = String((item as SelectionItem).value || '').trim();
    if (!value || seen.has(value)) continue;

    const labelRaw = (item as { label?: unknown }).label;
    if (isTermReference(labelRaw)) {
      const src = String(labelRaw.src || '').trim();
      if (!src) continue;
      seen.add(value);
      out.push({ value, label: src, labelText: labelRaw });
      continue;
    }

    const label = String(labelRaw || '').trim();
    if (!label) continue;
    seen.add(value);
    // Explicit labelText on returned objects is ignored; authors must put _lt on label.
    out.push({ value, label });
  }
  return out;
}

/**
 * Resolve dynamic selection via method name or callable.
 * Callables receive `this = ModelCtor` and must not use draft (D9 / T3.3).
 */
function evaluateDynamicSelection(ModelCtor: ModelFieldsGetCtor, field: FieldMetadata): SelectionItem[] {
  if (typeof field.selectionCallable === 'function') {
    return normalizeSelectionItems(field.selectionCallable.call(ModelCtor));
  }
  const methodName = String(field.selectionMethod || '').trim();
  if (!methodName) return [];
  const owner = ModelCtor as unknown as Record<string, unknown>;
  const fn = owner[methodName];
  if (typeof fn !== 'function') {
    throw new Error(`FieldsGet: selection method ${ModelCtor.name}.${methodName} is not a function`);
  }
  return normalizeSelectionItems((fn as (this: unknown) => unknown).call(ModelCtor));
}

function buildFieldMeta(
  ModelCtor: ModelFieldsGetCtor,
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

  const translatedHelp = translateFieldHelp(field, fallbackModule, fallbackScope);
  if (translatedHelp !== undefined) {
    meta.help = translatedHelp;
  }
  if (field.helpText && isTermReference(field.helpText)) {
    meta.helpText = { ...field.helpText };
  }

  if (field.selectionKind === 'dynamic' || field.selectionCallable || field.selectionMethod) {
    meta.selectionKind = 'dynamic';
    const evaluated = evaluateDynamicSelection(ModelCtor, field);
    const selection = translateSelectionLabels(evaluated);
    if (selection) meta.selection = selection;
  } else if (field.selection) {
    meta.selectionKind = 'static';
    const selection = translateSelectionLabels(field.selection);
    if (selection) meta.selection = selection;
  }

  const column = field.column as {
    notNull?: boolean;
    size?: number;
    precision?: number;
    scale?: number;
    scaleField?: string;
    currencyField?: string;
    index?: boolean | string;
  } | undefined;
  if (column?.notNull === true) meta.notNull = true;
  if (typeof column?.size === 'number') meta.size = column.size;
  if (typeof column?.precision === 'number') meta.precision = column.precision;
  if (typeof column?.scale === 'number') meta.scale = column.scale;
  if (typeof column?.scaleField === 'string' && column.scaleField.trim()) meta.scaleField = column.scaleField.trim();
  if (typeof column?.currencyField === 'string' && column.currencyField.trim()) meta.currencyField = column.currencyField.trim();
  if (column?.index !== undefined && column.index !== false) meta.indexed = true;

  if (field.translate === true) {
    meta.translate = true;
  }
  if (field.companyDependent === true) {
    meta.companyDependent = true;
  }
  if (field.copy === false) {
    meta.copy = false;
  }
  if (field.readonly === true) {
    meta.isReadonly = true;
  }
  if (typeof field.maxUploadBytes === 'number' && field.maxUploadBytes > 0) {
    meta.maxUploadBytes = field.maxUploadBytes;
  }
  if (typeof field.maxWidth === 'number' && field.maxWidth > 0) {
    meta.maxWidth = field.maxWidth;
  }
  if (typeof field.maxHeight === 'number' && field.maxHeight > 0) {
    meta.maxHeight = field.maxHeight;
  }
  const hintSize = field.storageHints?.size;
  if (typeof hintSize === 'number' && Number.isInteger(hintSize) && hintSize > 0 && meta.size == null) {
    meta.size = hintSize;
  }

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
 * - Deny-write fields remain visible with `isReadonly: true` (P5).
 * - Declarative `@Field({ readonly: true })` also forces `isReadonly` (PR-P2-F2).
 * - Unknown `fields` names are omitted.
 * - `attributes` narrows keys but always keeps `type`; deny-write / declarative readonly always force `isReadonly`.
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
  const denyWrite = await resolveDenyWriteFields(ModelCtor);
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
    const meta = projectAttributes(buildFieldMeta(ModelCtor, field, application, fallbackScope), attrList);
    if (denyWrite.has(name) || field.readonly === true) {
      meta.isReadonly = true;
    }
    result[name] = meta;
  }
  return result;
}
