// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  TRANSLATED_BASE_LANG,
  mergePropertyLabelWrite,
  mergeSelectionLabelsWrite,
  parsePropertyLabelMap,
  unwrapPropertyLabel,
} from '@/core/service/orm/model/properties_label_i18n';
import { PROPERTIES_V1_TYPES, type PropertyItemDefinition, type PropertyLabel } from '@/core/service/orm/model/properties_types';

import { normalizeSelectionOptions } from './propertiesHelpers';

export const PROPERTY_DEFINITION_V1_TYPE_OPTIONS = [...PROPERTIES_V1_TYPES].sort();

export type DefinitionEditorDraftItem = {
  name: string;
  type: string;
  /** Current-language label shown in the main input. */
  string: string;
  /** Full label lang map (source of truth for Save / translate dialog). */
  stringMap: Record<string, string>;
  /** Current-language help tip shown in the Help input. */
  help: string;
  /** Full help lang map (source of truth for Save / translate dialog). */
  helpMap: Record<string, string>;
  default: string;
  readonly: boolean;
  /** Selection options bridge: JSON tuples of [value, current-lang label] for the list editor. */
  selectionText: string;
  /** Option value → label lang map (survives current-lang list edits via merge). */
  selectionLabelMaps: Record<string, Record<string, string>>;
};

/** One selection option row in the definition list editor (O2M-like). */
export type SelectionDraftRow = {
  value: string;
  label: string;
};

/** Parse draft.selectionText into editable list rows (may include blank values while typing). */
export function selectionRowsFromDraft(draft: DefinitionEditorDraftItem): SelectionDraftRow[] {
  try {
    const parsed = draft.selectionText ? JSON.parse(draft.selectionText) : [];
    return normalizeSelectionOptions(parsed).map(o => ({
      value: o.value,
      label: String(o.label ?? ''),
    }));
  } catch {
    return Object.keys(draft.selectionLabelMaps || {}).map(value => ({
      value,
      label: unwrapPropertyLabel(draft.selectionLabelMaps[value] || {}, TRANSLATED_BASE_LANG) || value,
    }));
  }
}

/**
 * Write list-editor rows back onto the draft.
 * Preserves lang maps across value renames by index when possible.
 */
export function writeSelectionRowsToDraft(
  draft: DefinitionEditorDraftItem,
  rows: SelectionDraftRow[],
  lang: string = TRANSLATED_BASE_LANG,
): void {
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  const prevRows = selectionRowsFromDraft(draft);
  const prevMaps: Record<string, Record<string, string>> = {};
  for (const [k, v] of Object.entries(draft.selectionLabelMaps || {})) {
    prevMaps[k] = { ...(v || {}) };
  }
  const cleaned = (rows || []).map(r => ({
    value: String(r?.value ?? '').trim(),
    label: String(r?.label ?? ''),
  }));
  const nextMaps: Record<string, Record<string, string>> = {};
  const usedPrev = new Set<string>();
  for (let i = 0; i < cleaned.length; i++) {
    const row = cleaned[i]!;
    if (!row.value) continue;
    const oldValue = String(prevRows[i]?.value || '').trim();
    let base = prevMaps[row.value];
    if (!base && oldValue && oldValue !== row.value && !usedPrev.has(oldValue)) {
      base = prevMaps[oldValue];
      usedPrev.add(oldValue);
    }
    nextMaps[row.value] = asLabelMap(mergePropertyLabelWrite(base || {}, row.label, code));
  }
  draft.selectionLabelMaps = nextMaps;
  const display = cleaned.map(r => [r.value, r.label || r.value] as [string, string]);
  try {
    draft.selectionText = display.length ? JSON.stringify(display) : '';
  } catch {
    draft.selectionText = '';
  }
}

export function emptyDraftItem(): DefinitionEditorDraftItem {
  return {
    name: '',
    type: 'char',
    string: '',
    stringMap: {},
    help: '',
    helpMap: {},
    default: '',
    readonly: false,
    selectionText: '',
    selectionLabelMaps: {},
  };
}

function asLabelMap(value: unknown): Record<string, string> {
  return parsePropertyLabelMap(value) || {};
}

/** Resolve a selection option's raw label (tuple / object / other → undefined). */
export function selectionRawLabel(opt: unknown): unknown {
  if (Array.isArray(opt)) return opt[1] ?? opt[0];
  if (opt && typeof opt === 'object' && !Array.isArray(opt)) {
    return (opt as { label?: unknown }).label ?? (opt as { value?: unknown }).value;
  }
  return undefined;
}

function selectionRawValue(opt: unknown): string | null {
  if (Array.isArray(opt) && typeof opt[0] === 'string') return opt[0];
  if (opt && typeof opt === 'object' && !Array.isArray(opt) && typeof (opt as { value?: unknown }).value === 'string') {
    return String((opt as { value: string }).value);
  }
  return null;
}

/** Sync the current-lang `string` / `help` / selectionText views from stored maps. */
export function syncDraftLabelsForLang(draft: DefinitionEditorDraftItem, lang: string): void {
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  draft.string = unwrapPropertyLabel(draft.stringMap, code);
  draft.help = unwrapPropertyLabel(draft.helpMap, code);
  if (draft.type !== 'selection') return;
  const options = normalizeSelectionOptions(
    (() => {
      try {
        return draft.selectionText ? JSON.parse(draft.selectionText) : [];
      } catch {
        return [];
      }
    })(),
  );
  const values = options.length
    ? options.map(o => o.value)
    : Object.keys(draft.selectionLabelMaps);
  const display = values.map(value => {
    const map = draft.selectionLabelMaps[value] || {};
    return [value, unwrapPropertyLabel(map, code) || value] as [string, string];
  });
  try {
    draft.selectionText = display.length ? JSON.stringify(display) : '';
  } catch {
    draft.selectionText = '';
  }
}

/** Apply the current-lang UI fields into the draft maps (before Save / before opening translate dialog). */
export function applyDraftCurrentLangToMaps(draft: DefinitionEditorDraftItem, lang: string): void {
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  draft.string = String(draft.string || '').trim();
  draft.stringMap = asLabelMap(mergePropertyLabelWrite(draft.stringMap, draft.string, code));
  draft.help = String(draft.help || '').trim();
  draft.helpMap = asLabelMap(mergePropertyLabelWrite(draft.helpMap, draft.help, code));
  if (draft.type !== 'selection') return;
  const text = String(draft.selectionText || '').trim();
  if (!text) return;
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return;
  }
  const options = normalizeSelectionOptions(parsed);
  if (!options.length) return;
  const merged = mergeSelectionLabelsWrite(
    Object.entries(draft.selectionLabelMaps).map(([value, labelMap]) => [value, labelMap] as [string, PropertyLabel]),
    options,
    code,
  );
  const nextMaps: Record<string, Record<string, string>> = {};
  for (const opt of merged || []) {
    const value = selectionRawValue(opt);
    if (value == null) continue;
    nextMaps[value] = asLabelMap(selectionRawLabel(opt));
  }
  draft.selectionLabelMaps = nextMaps;
}

export function definitionItemsToDrafts(
  items: unknown,
  lang: string = TRANSLATED_BASE_LANG,
): DefinitionEditorDraftItem[] {
  if (!Array.isArray(items)) return [];
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  const out: DefinitionEditorDraftItem[] = [];
  for (const raw of items) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) continue;
    const item = raw as Record<string, unknown>;
    const name = String(item.name || '').trim();
    if (!name) continue;
    const type = String(item.type || 'char') || 'char';
    const stringMap = asLabelMap(item.string);
    const helpMap = asLabelMap(item.help);
    const selectionLabelMaps: Record<string, Record<string, string>> = {};
    let selectionText = '';
    if (Array.isArray(item.selection)) {
      const display: Array<[string, string]> = [];
      for (const opt of item.selection) {
        const value = selectionRawValue(opt);
        if (value == null) continue;
        const map = asLabelMap(selectionRawLabel(opt));
        selectionLabelMaps[value] = map;
        display.push([value, unwrapPropertyLabel(map, code) || value]);
      }
      try {
        selectionText = display.length ? JSON.stringify(display) : '';
      } catch {
        selectionText = '';
      }
    }
    out.push({
      name,
      type,
      string: unwrapPropertyLabel(stringMap, code),
      stringMap,
      help: unwrapPropertyLabel(helpMap, code),
      helpMap,
      default: item.default == null ? '' : String(item.default),
      readonly: item.readonly === true,
      selectionText,
      selectionLabelMaps,
    });
  }
  return out;
}

/** Build Definition payload for Create/Update. Invalid selection JSON → throws. */
export function draftsToDefinitionItems(
  drafts: DefinitionEditorDraftItem[] | null | undefined,
  lang: string = TRANSLATED_BASE_LANG,
  previousItems?: PropertyItemDefinition[] | null,
): PropertyItemDefinition[] {
  const code = String(lang || '').trim() || TRANSLATED_BASE_LANG;
  // previousItems kept for call-site compatibility; draft maps are the merge base.
  void previousItems;
  const out: PropertyItemDefinition[] = [];
  const seenNames = new Set<string>();
  for (const draft of drafts || []) {
    applyDraftCurrentLangToMaps(draft, code);
    const name = String(draft.name || '').trim();
    if (!name) continue;
    if (seenNames.has(name)) {
      throw new Error(`duplicate property name '${name}'`);
    }
    seenNames.add(name);
    const type = String(draft.type || '').trim();
    if (!PROPERTIES_V1_TYPES.has(type)) {
      throw new Error(`unsupported property type '${type}' (${name})`);
    }
    const item: PropertyItemDefinition = { name, type };
    const label = String(draft.string || '').trim();
    draft.string = label;
    if (label || Object.keys(draft.stringMap).length) {
      // Draft maps are the source of truth (translate dialog writes here).
      item.string = mergePropertyLabelWrite(draft.stringMap, label, code);
    }
    const help = String(draft.help || '').trim();
    draft.help = help;
    const helpHasContent =
      !!help || Object.values(draft.helpMap || {}).some(v => String(v || '').trim() !== '');
    if (helpHasContent) {
      item.help = mergePropertyLabelWrite(draft.helpMap, help, code);
    }
    if (draft.readonly) item.readonly = true;
    const defRaw = String(draft.default ?? '');
    if (defRaw.trim() !== '') {
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
      const options = normalizeSelectionOptions(parsed).filter(o => String(o.value || '').trim() !== '');
      if (!options.length) {
        throw new Error(`selection property '${name}' has no usable options`);
      }
      if (item.default != null && !options.some(o => o.value === String(item.default))) {
        throw new Error(`selection property '${name}' default is not one of the options`);
      }
      const previousSelection = Object.entries(draft.selectionLabelMaps || {}).map(
        ([value, map]) => [value, map] as [string, PropertyLabel],
      );
      item.selection = mergeSelectionLabelsWrite(previousSelection, options, code);
    }
    out.push(item);
  }
  return out;
}

function coerceDefaultForType(type: string, raw: string): unknown {
  const trimmed = raw.trim();
  if (type === 'boolean') {
    const s = trimmed.toLowerCase();
    if (s === 'true' || s === '1') return true;
    if (s === 'false' || s === '0') return false;
    return raw;
  }
  if (type === 'integer') {
    if (!trimmed) return raw;
    const n = Number(trimmed);
    return Number.isFinite(n) ? Math.trunc(n) : raw;
  }
  if (type === 'float') {
    if (!trimmed) return raw;
    const n = Number(trimmed);
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
  const containerModel =
    opts.containerModel == null || String(opts.containerModel).trim() === ''
      ? null
      : String(opts.containerModel).trim();
  const containerId =
    opts.containerId == null || String(opts.containerId).trim() === ''
      ? null
      : String(opts.containerId).trim();
  return [
    ['TargetModel', '=', targetModel],
    ['PropertiesField', '=', propertiesField],
    ['ContainerModel', '=', containerModel],
    ['ContainerId', '=', containerId],
  ];
}
