// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { PROPERTIES_V1_TYPES, type PropertyItemDefinition } from '@/core/service/orm/model/properties_types';

export const PROPERTY_DEFINITION_V1_TYPE_OPTIONS = [...PROPERTIES_V1_TYPES].sort();

export type DefinitionEditorDraftItem = {
  name: string;
  type: string;
  string: string;
  default: string;
  readonly: boolean;
  selectionText: string;
};

export function emptyDraftItem(): DefinitionEditorDraftItem {
  return {
    name: '',
    type: 'char',
    string: '',
    default: '',
    readonly: false,
    selectionText: '',
  };
}

export function definitionItemsToDrafts(items: unknown): DefinitionEditorDraftItem[] {
  if (!Array.isArray(items)) return [];
  const out: DefinitionEditorDraftItem[] = [];
  for (const raw of items) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) continue;
    const item = raw as Record<string, unknown>;
    const name = String(item.name || '').trim();
    if (!name) continue;
    const type = String(item.type || 'char');
    let selectionText = '';
    if (Array.isArray(item.selection)) {
      try {
        selectionText = JSON.stringify(item.selection);
      } catch {
        selectionText = '';
      }
    }
    out.push({
      name,
      type,
      string: item.string == null ? '' : String(item.string),
      default: item.default == null ? '' : String(item.default),
      readonly: item.readonly === true,
      selectionText,
    });
  }
  return out;
}

/** Build Definition payload for Create/Update. Invalid selection JSON → throws. */
export function draftsToDefinitionItems(drafts: DefinitionEditorDraftItem[] | null | undefined): PropertyItemDefinition[] {
  const out: PropertyItemDefinition[] = [];
  for (const draft of drafts || []) {
    const name = String(draft.name || '').trim();
    if (!name) continue;
    const type = String(draft.type || '').trim();
    if (!PROPERTIES_V1_TYPES.has(type)) {
      throw new Error(`unsupported property type '${type}' (${name})`);
    }
    const item: PropertyItemDefinition = { name, type };
    const label = String(draft.string || '').trim();
    if (label) item.string = label;
    if (draft.readonly) item.readonly = true;
    const defRaw = String(draft.default ?? '');
    if (defRaw !== '') {
      item.default = coerceDefaultForType(type, defRaw);
    }
    if (type === 'selection') {
      const text = String(draft.selectionText || '').trim();
      if (!text) {
        throw new Error(`selection property '${name}' requires selection JSON`);
      }
      let parsed: unknown;
      try {
        parsed = JSON.parse(text);
      } catch {
        throw new Error(`selection property '${name}' has invalid JSON`);
      }
      if (!Array.isArray(parsed) || !parsed.length) {
        throw new Error(`selection property '${name}' requires a non-empty selection array`);
      }
      item.selection = parsed as PropertyItemDefinition['selection'];
    }
    out.push(item);
  }
  return out;
}

function coerceDefaultForType(type: string, raw: string): unknown {
  if (type === 'boolean') {
    const s = raw.trim().toLowerCase();
    if (s === 'true' || s === '1') return true;
    if (s === 'false' || s === '0') return false;
    return raw;
  }
  if (type === 'integer') {
    const n = Number(raw);
    return Number.isFinite(n) ? Math.trunc(n) : raw;
  }
  if (type === 'float') {
    const n = Number(raw);
    return Number.isFinite(n) ? n : raw;
  }
  return raw;
}

export function buildDefinitionScopeCondition(opts: {
  targetModel: string;
  propertiesField: string;
  containerModel?: string | null;
  containerId?: string | null;
}): unknown[] {
  const targetModel = String(opts.targetModel || '').trim();
  const propertiesField = String(opts.propertiesField || '').trim();
  const containerModel = opts.containerModel == null || opts.containerModel === '' ? null : String(opts.containerModel);
  const containerId = opts.containerId == null || opts.containerId === '' ? null : String(opts.containerId);
  return [
    ['TargetModel', '=', targetModel],
    ['PropertiesField', '=', propertiesField],
    containerModel == null ? ['ContainerModel', '=', null] : ['ContainerModel', '=', containerModel],
    containerId == null ? ['ContainerId', '=', null] : ['ContainerId', '=', containerId],
  ];
}
