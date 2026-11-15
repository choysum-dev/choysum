// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Frontend-only ModelStore stand-in for pages with no ORM model (login/register,
 * settings sheets, PropertyDefinition dialogs, and similar pure-scalar UIs).
 * Satisfies the ModelStore slice that useField / ChoyFieldBase / ChoyFormView
 * read (storeId, fieldsMetadata, getFieldMeta). Values are in-memory; does not
 * call gRPC.
 *
 * Pure scalars only. Relation / monetary / binary / properties / statusbar and
 * pluggable adapters belong in a follow-up wizard store design.
 */

import { reactive } from 'vue';
import { createEmptyQueryState } from '@/web/web/query/state';
import type { SelectionItem, WebFieldMetadata, ModelStore } from '@/web/web/stores/modelStore';

export type UiModelFieldType =
  | 'varchar'
  | 'char'
  | 'text'
  | 'boolean'
  | 'selection'
  | 'integer'
  | 'bigint'
  | 'float'
  | 'date'
  | 'datetime'
  | 'time'
  | 'html'
  | 'jsonobject';

export type UiModelFieldDef = {
  name: string;
  label: string;
  type: UiModelFieldType;
  help?: string;
  /**
   * Required when type is selection (static options for ChoySelectionField).
   * May start empty and be filled later via setFieldSelection.
   */
  selection?: readonly SelectionItem[];
  /** Optional char/varchar length hint written to meta.size. */
  size?: number;
};

export type CreateUiModelStoreOptions = {
  fields: readonly UiModelFieldDef[];
  initialValues?: Record<string, unknown>;
  storeId?: string;
};

const LOCAL_FIELD_TYPES = new Set<UiModelFieldType>([
  'varchar',
  'char',
  'text',
  'boolean',
  'selection',
  'integer',
  'bigint',
  'float',
  'date',
  'datetime',
  'time',
  'html',
  'jsonobject',
]);

let uiModelStoreSeq = 0;

function defaultValueForType(type: UiModelFieldType): unknown {
  switch (type) {
    case 'boolean':
      return false;
    case 'integer':
    case 'bigint':
    case 'float':
    case 'selection':
    case 'date':
    case 'datetime':
    case 'time':
      return null;
    case 'jsonobject':
      return {};
    default:
      return '';
  }
}

function typeAnnotationFor(type: UiModelFieldType): string {
  switch (type) {
    case 'boolean':
      return 'boolean';
    case 'integer':
    case 'float':
      return 'number';
    case 'bigint':
      return 'string';
    case 'jsonobject':
      return 'object';
    default:
      return 'string';
  }
}

function unsupported(method: string): () => never {
  return () => {
    throw new Error(`createUiModelStore does not support ${method}`);
  };
}

function emptyDict<T>(): Record<string, T> {
  return Object.create(null) as Record<string, T>;
}

function normalizeSelection(items: readonly SelectionItem[]): SelectionItem[] {
  return items.map(item => ({
    value: String(item.value),
    label: String(item.label ?? item.value),
  }));
}

function buildFieldsMetadata(fields: readonly UiModelFieldDef[]): Record<string, WebFieldMetadata> {
  const names = new Set<string>();
  const meta = emptyDict<WebFieldMetadata>();
  for (const field of fields) {
    if (!field || typeof field !== 'object') {
      throw new Error('createUiModelStore field definition must be an object');
    }
    const name = String(field.name || '').trim();
    if (!name) {
      throw new Error('createUiModelStore field name must be non-empty');
    }
    if (names.has(name)) {
      throw new Error(`createUiModelStore duplicate field "${name}"`);
    }
    if (!LOCAL_FIELD_TYPES.has(field.type)) {
      throw new Error(`createUiModelStore unsupported field type "${String(field.type)}"`);
    }
    if (field.type === 'selection') {
      if (!Array.isArray(field.selection)) {
        throw new Error(`createUiModelStore selection field "${name}" requires a selection array`);
      }
    }
    names.add(name);
    const entry: WebFieldMetadata = {
      id: name,
      type: field.type,
      typeAnnotation: typeAnnotationFor(field.type),
      string: field.label,
      help: field.help,
    };
    if (field.type === 'selection' && field.selection) {
      entry.selection = normalizeSelection(field.selection);
      entry.selectionKind = 'static';
    }
    if (
      (field.type === 'char' || field.type === 'varchar') &&
      typeof field.size === 'number' &&
      Number.isInteger(field.size) &&
      field.size > 0
    ) {
      entry.size = field.size;
    }
    meta[name] = entry;
  }
  return meta;
}

export type UiModelStore = ModelStore<any> & {
  readonly fieldNames: readonly string[];
  getField(name: string): unknown;
  setField(name: string, value: unknown): void;
  getValues(): Record<string, unknown>;
  /** Replaces a selection field's options; mounted ChoySelectionField re-renders. */
  setFieldSelection(name: string, items: readonly SelectionItem[]): void;
};

/**
 * Builds a frontend-only ModelStore. Not registered with createModelStore.
 */
export function createUiModelStore(options: CreateUiModelStoreOptions): UiModelStore {
  const fieldsMetadata = reactive(buildFieldsMetadata(options.fields || []));
  const fieldNames = Object.freeze(Object.keys(fieldsMetadata));
  const allowed = new Set(fieldNames);

  const values = reactive(emptyDict<unknown>());
  for (const name of fieldNames) {
    const type = fieldsMetadata[name]!.type as UiModelFieldType;
    const initialValues = options.initialValues;
    const candidate =
      initialValues && Object.prototype.hasOwnProperty.call(initialValues, name)
        ? initialValues[name]
        : undefined;
    values[name] = candidate ?? defaultValueForType(type);
  }

  const storeId = String(options.storeId || '').trim() || `ui-model:${++uiModelStoreSeq}`;

  let context: Record<string, string> = {};

  const getFieldMeta = (name: string): WebFieldMetadata | undefined => {
    const fieldName = String(name || '').trim();
    return fieldName ? fieldsMetadata[fieldName] : undefined;
  };

  const ensureFieldsGet = async (fields?: string[]): Promise<Record<string, WebFieldMetadata>> => {
    if (!fields?.length) {
      const all = emptyDict<WebFieldMetadata>();
      for (const name of fieldNames) {
        all[name] = fieldsMetadata[name]!;
      }
      return all;
    }
    const slice = emptyDict<WebFieldMetadata>();
    for (const raw of fields) {
      const fieldName = String(raw || '').trim();
      const meta = fieldsMetadata[fieldName];
      if (meta) slice[fieldName] = meta;
    }
    return slice;
  };

  const store = {
    storeId,
    fieldsMetadata,
    fieldNames,
    state: {
      queryState: createEmptyQueryState(),
      result: undefined,
      selection: [] as string[],
      planCache: new Map(),
    },
    isLoading: false,
    destroy() {},
    setContext(ctx: Record<string, string>) {
      context = { ...(ctx || {}) };
    },
    getContext() {
      return { ...context };
    },
    async withContext<T>(ctx: Record<string, string>, fn: () => Promise<T>): Promise<T> {
      const prev = context;
      context = { ...prev, ...(ctx || {}) };
      try {
        return await fn();
      } finally {
        context = prev;
      }
    },
    getField(name: string): unknown {
      const fieldName = String(name || '').trim();
      if (!allowed.has(fieldName)) return undefined;
      return values[fieldName];
    },
    setField(name: string, value: unknown): void {
      const fieldName = String(name || '').trim();
      if (!allowed.has(fieldName)) return;
      values[fieldName] = value;
    },
    getValues(): Record<string, unknown> {
      const out = emptyDict<unknown>();
      for (const name of fieldNames) {
        out[name] = values[name];
      }
      return out;
    },
    setFieldSelection(name: string, items: readonly SelectionItem[]): void {
      const meta = fieldsMetadata[String(name || '').trim()];
      if (!meta || meta.type !== 'selection') {
        throw new Error(`createUiModelStore field "${String(name)}" is not a selection field`);
      }
      meta.selection = normalizeSelection(Array.isArray(items) ? items : []);
    },
    getFieldMeta,
    getFieldsGetTranslatedString: () => undefined,
    getFieldsGetTranslatedHelp: () => undefined,
    clearFieldsGetCache() {},
    ensureFieldsGet,
    FieldsGet: ensureFieldsGet,
    Onchange: async () => ({ value: undefined, messages: [] }),
    Browse: unsupported('Browse'),
    BrowseMany: unsupported('BrowseMany'),
    Create: unsupported('Create'),
    CreateMany: unsupported('CreateMany'),
    Update: unsupported('Update'),
    UpdateById: unsupported('UpdateById'),
    Copy: unsupported('Copy'),
    NameSearch: unsupported('NameSearch'),
    NameCreate: unsupported('NameCreate'),
    Count: unsupported('Count'),
    Search: unsupported('Search'),
    ReadGroup: unsupported('ReadGroup'),
    ReadGroupCount: unsupported('ReadGroupCount'),
    Delete: unsupported('Delete'),
    DeleteById: unsupported('DeleteById'),
    GetFieldTranslations: unsupported('GetFieldTranslations'),
    UpdateFieldTranslations: unsupported('UpdateFieldTranslations'),
    GetFieldCompanyValues: unsupported('GetFieldCompanyValues'),
    UpdateFieldCompanyValues: unsupported('UpdateFieldCompanyValues'),
    ResolveProperties: unsupported('ResolveProperties'),
  };

  return store as unknown as UiModelStore;
}
