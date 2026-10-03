// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * In-memory form store for pages with no ORM model (login/register drafts).
 * Satisfies the WebModelStore slice that useField / FieldBase / ChoyFormView
 * read (storeId, fieldsMetadata, getFieldMeta). Does not call gRPC.
 */

import { reactive } from 'vue';
import { createEmptyQueryState } from '@/web/web/query/state';
import type { WebFieldMetadata, WebModelStore } from '@/web/web/stores/modelStore';

export type LocalFormFieldType = 'varchar' | 'boolean';

export type LocalFormFieldDef = {
  name: string;
  label: string;
  type: LocalFormFieldType;
  help?: string;
};

export type CreateLocalFormStoreOptions = {
  fields: readonly LocalFormFieldDef[];
  initialValues?: Record<string, unknown>;
  storeId?: string;
};

const LOCAL_FIELD_TYPES = new Set<LocalFormFieldType>(['varchar', 'boolean']);

let localFormStoreSeq = 0;

function defaultValueForType(type: LocalFormFieldType): unknown {
  return type === 'boolean' ? false : '';
}

function typeAnnotationFor(type: LocalFormFieldType): string {
  return type === 'boolean' ? 'boolean' : 'string';
}

function unsupported(method: string): () => never {
  return () => {
    throw new Error(`createLocalFormStore does not support ${method}`);
  };
}

function emptyDict<T>(): Record<string, T> {
  return Object.create(null) as Record<string, T>;
}

function buildFieldsMetadata(fields: readonly LocalFormFieldDef[]): Record<string, WebFieldMetadata> {
  const names = new Set<string>();
  const meta = emptyDict<WebFieldMetadata>();
  for (const field of fields) {
    const name = String(field.name || '').trim();
    if (!name) {
      throw new Error('createLocalFormStore field name must be non-empty');
    }
    if (names.has(name)) {
      throw new Error(`createLocalFormStore duplicate field "${name}"`);
    }
    if (!LOCAL_FIELD_TYPES.has(field.type)) {
      throw new Error(`createLocalFormStore unsupported field type "${String(field.type)}"`);
    }
    names.add(name);
    meta[name] = {
      id: name,
      type: field.type,
      typeAnnotation: typeAnnotationFor(field.type),
      string: field.label,
      help: field.help,
    };
  }
  return meta;
}

export type LocalFormStore = WebModelStore<any> & {
  readonly fieldNames: readonly string[];
  getField(name: string): unknown;
  setField(name: string, value: unknown): void;
  getValues(): Record<string, unknown>;
};

/**
 * Builds a store-only create draft. Not registered with createStoreByModel.
 */
export function createLocalFormStore(options: CreateLocalFormStoreOptions): LocalFormStore {
  const fieldsMetadata = buildFieldsMetadata(options.fields || []);
  const fieldNames = Object.freeze(Object.keys(fieldsMetadata));
  const allowed = new Set(fieldNames);

  const values = reactive(emptyDict<unknown>());
  for (const name of fieldNames) {
    const type = fieldsMetadata[name]!.type as LocalFormFieldType;
    const initialValues = options.initialValues;
    const candidate =
      initialValues && Object.prototype.hasOwnProperty.call(initialValues, name)
        ? initialValues[name]
        : undefined;
    values[name] = candidate === undefined ? defaultValueForType(type) : candidate;
  }

  const storeId = String(options.storeId || '').trim() || `local-form:${++localFormStoreSeq}`;

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

  return store as unknown as LocalFormStore;
}
