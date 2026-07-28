// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../runtime/context';
import { MetadataStorage } from '../metadata/storage';
import {
  applyFieldCompanyValuesPatch,
  isCompanyValueMap,
  parseCompanyDependentStoredMap,
  type CompanyValueMap,
} from '../repository/projection/company_dependent_field_codec';
import type BaseModel from './model';
import { browseModel } from './model_read_facade';
import { updateModelById } from './model_update_service_facade';
import type { RuntimeModelCtor } from './types';

export type FieldCompanyValuesMap = CompanyValueMap;

type FieldCompanyValuesCtor<T extends BaseModel> = RuntimeModelCtor<T>;

function resolveCompanyDependentFieldMeta(ModelCtor: FieldCompanyValuesCtor<BaseModel>, fieldName: string) {
  const name = String(fieldName || '').trim();
  if (!name) {
    throw new Error('GetFieldCompanyValues/UpdateFieldCompanyValues requires a non-empty fieldName');
  }
  const modelMeta = MetadataStorage.instance.getModelMetadata(ModelCtor as never);
  const field = modelMeta?.fields?.get(name);
  if (!field?.companyDependent) {
    throw new Error(`Field "${name}" is not a company-dependent field`);
  }
  return { name, field, modelMeta };
}

function asCompanyValuesMap(value: unknown): FieldCompanyValuesMap {
  if (value == null) return {};
  if (isCompanyValueMap(value)) {
    return { ...value };
  }
  const parsed = parseCompanyDependentStoredMap(value);
  return parsed ? { ...parsed } : {};
}

function filterCompanyIds(map: FieldCompanyValuesMap, companyIds?: string[]): FieldCompanyValuesMap {
  if (!companyIds || !companyIds.length) return { ...map };
  const out: FieldCompanyValuesMap = {};
  for (const raw of companyIds) {
    const id = String(raw || '').trim();
    if (!id) continue;
    if (Object.prototype.hasOwnProperty.call(map, id)) {
      out[id] = map[id];
    }
  }
  return out;
}

/**
 * Returns the stored company map for a companyDependent field (optionally filtered).
 */
export async function getModelFieldCompanyValues<T extends BaseModel>(
  ModelCtor: FieldCompanyValuesCtor<T>,
  id: string,
  fieldName: string,
  companyIds?: string[]
): Promise<FieldCompanyValuesMap> {
  const { name } = resolveCompanyDependentFieldMeta(ModelCtor, fieldName);
  const recordId = String(id || '').trim();
  if (!recordId) {
    throw new Error('GetFieldCompanyValues requires a non-empty id');
  }

  const row = await withContext({ prefetch_companies: true }, () =>
    browseModel(ModelCtor, recordId, [name] as never)
  );
  const map = asCompanyValuesMap((row as unknown as Record<string, unknown>)[name]);
  return filterCompanyIds(map, companyIds);
}

/**
 * Patch company-dependent field keys: value writes; `false` deletes.
 */
export async function updateModelFieldCompanyValues<T extends BaseModel>(
  ModelCtor: FieldCompanyValuesCtor<T>,
  id: string,
  fieldName: string,
  values: Record<string, unknown | false>
): Promise<boolean> {
  const { name } = resolveCompanyDependentFieldMeta(ModelCtor, fieldName);
  const recordId = String(id || '').trim();
  if (!recordId) {
    throw new Error('UpdateFieldCompanyValues requires a non-empty id');
  }

  const current = await getModelFieldCompanyValues(ModelCtor, recordId, name);
  const next = applyFieldCompanyValuesPatch({
    fieldName: name,
    currentMap: Object.keys(current).length ? current : null,
    values,
  });

  await withContext({ company_write_replace: true }, () =>
    updateModelById(ModelCtor, recordId, { [name]: next } as never)
  );
  return true;
}
