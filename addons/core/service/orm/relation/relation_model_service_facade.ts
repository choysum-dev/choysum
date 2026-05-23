// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { createModel } from '../model/model_create_service_facade';
import { updateModelById } from '../model/model_update_service_facade';
import type { RuntimeModelCtor } from '../model/types';
import type { UnknownRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';

type RelationModelCtor<T extends BaseModel = BaseModel> = RuntimeModelCtor<T>;

function hasStaticOverride(ModelCtor: RelationModelCtor, methodName: 'Create' | 'UpdateById'): boolean {
  return ModelCtor[methodName] !== BaseModel[methodName];
}

function normalizeCreatedId(ModelCtor: RelationModelCtor, created: unknown): string {
  if (typeof created === 'string') return created;
  if (typeof created === 'number' || typeof created === 'bigint') return String(created);
  const createdRecord = asObjectRecord(created);
  if (createdRecord && 'Id' in createdRecord) {
    const id = createdRecord.Id;
    if (typeof id === 'string' || typeof id === 'number' || typeof id === 'bigint') {
      return String(id);
    }
  }
  throw new Error(`Relation create did not return a valid Id for ${ModelCtor.name}`);
}

export async function createRelationModel<T extends BaseModel>(ModelCtor: RelationModelCtor<T>, value: UnknownRecord): Promise<string> {
  const staticCreate = (ModelCtor as RelationModelCtor<T> & { Create?: (value: UnknownRecord) => Promise<unknown> }).Create;
  if (hasStaticOverride(ModelCtor, 'Create')) {
    if (typeof staticCreate !== 'function') {
      throw new Error(`Relation create override is missing for ${ModelCtor.name}`);
    }
    return normalizeCreatedId(ModelCtor, await staticCreate.call(ModelCtor, value));
  }
  return normalizeCreatedId(ModelCtor, await createModel(ModelCtor, value as never));
}

export async function updateRelationModelById<T extends BaseModel>(ModelCtor: RelationModelCtor<T>, id: string, values: UnknownRecord): Promise<boolean> {
  const staticUpdate = (ModelCtor as RelationModelCtor<T> & { UpdateById?: (id: string, values: UnknownRecord) => Promise<unknown> }).UpdateById;
  const result = hasStaticOverride(ModelCtor, 'UpdateById')
    ? typeof staticUpdate === 'function'
      ? await staticUpdate.call(ModelCtor, id, values)
      : null
    : await updateModelById(ModelCtor, id, values as never);

  if (result == null) return false;
  if (typeof result === 'number') return result > 0;
  if (typeof result === 'boolean') return result;
  if (Array.isArray(result)) return result.length > 0;
  return true;
}
