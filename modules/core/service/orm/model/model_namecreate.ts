// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldSelection, Insertable } from '../repository/types';
import type { ModelMetadata, FieldMetadata } from '../metadata';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';

export type NameCreateOptions<T extends BaseModel> = {
  nameField?: string;
  returnFields?: FieldSelection<T>;
};

type NameCreateModelCtor<T extends BaseModel> = RuntimeModelCtor<T> & {
  Create: (value: Partial<Insertable<T & BaseModel>>, returnFields?: FieldSelection<T>) => Promise<T>;
};

/**
 * True when the field exists and is a writable stored column (not SqlCompute / virtual compute / non-stored related).
 */
export function isWritableStoredField(meta: ModelMetadata, fieldName: string): boolean {
  const name = String(fieldName ?? '').trim();
  if (!name) return false;
  const field = meta.fields.get(name) as FieldMetadata | undefined;
  if (!field?.column) return false;
  if (field.column.primaryKey === true) return false;
  if (meta.sqlComputeHandlers?.has(name)) return false;
  const compute = meta.computeHandlers?.get(name);
  if (compute?.store === false) return false;
  if (meta.computeGraph?.virtualComputeFields?.has(name)) return false;
  if (field.related && field.related.store !== true) return false;
  return true;
}

/**
 * Resolve NameCreate write field: explicit nameField → writable Name → throw (D2).
 */
export function resolveNameCreateField(meta: ModelMetadata, nameField?: string): string {
  const explicit = String(nameField ?? '').trim();
  if (explicit) {
    if (!isWritableStoredField(meta, explicit)) {
      throw new Error(`NameCreate: nameField ${JSON.stringify(explicit)} is missing or not a writable stored field`);
    }
    return explicit;
  }
  if (isWritableStoredField(meta, 'Name')) {
    return 'Name';
  }
  throw new Error('NameCreate: no writable Name field; pass options.nameField or override NameCreate');
}

/**
 * Default NameCreate: trim name → resolve field → Create (D1/D2/D4).
 */
export async function nameCreateModels<T extends BaseModel>(
  ModelCtor: NameCreateModelCtor<T>,
  name: string,
  values?: Partial<Insertable<T & BaseModel>>,
  options?: NameCreateOptions<T>
): Promise<T> {
  const kw = String(name ?? '').trim();
  if (!kw) {
    throw new Error('NameCreate: name is empty');
  }
  const meta = getModelRuntimeMetadata(ModelCtor as RuntimeModelCtor<T>);
  const field = resolveNameCreateField(meta, options?.nameField);
  const payload = {
    ...(values || {}),
    [field]: kw,
  } as Partial<Insertable<T & BaseModel>>;
  return await ModelCtor.Create(payload, options?.returnFields);
}
