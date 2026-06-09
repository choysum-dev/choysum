// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { EntityConverter } from '../utils/converter';
import type { Entity } from '../repository';
import type { FieldSelection } from '../repository/types';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { createModelProxy, getModelRepository } from './model_internal_facade';

type ModelEdgeFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T>;

type ModelEdgeFacadeInstance = {
  entity: Entity;
  fields?: FieldSelection<BaseModel>;
};

export async function withModelSavepoint<T extends BaseModel, R>(ModelCtor: ModelEdgeFacadeCtor<T>, fn: () => Promise<R>, name?: string): Promise<R> {
  const repo = getModelRepository(ModelCtor);
  return await repo.withSavepoint(fn, name);
}

export function hydrateModelFacade<T extends BaseModel>(ModelCtor: ModelEdgeFacadeCtor<T>, entity: Entity, fields?: FieldSelection<T>): T {
  return createModelProxy<T>(ModelCtor, entity, fields);
}

export function toPlainObject(instance: BaseModel): Entity {
  const model = instance as unknown as ModelEdgeFacadeInstance;
  return EntityConverter.modelToPlainObject(instance, model.fields);
}

export function toEntity(instance: BaseModel): Entity {
  const model = instance as unknown as ModelEdgeFacadeInstance;
  return { ...model.entity };
}
