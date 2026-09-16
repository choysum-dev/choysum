// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RepositoryFactory } from '../repository/repository_factory';
import type { Repository, Entity } from '../repository';
import type { FieldSelection } from '../repository/types';
import { hydrateModel } from './model_hydration';
import type BaseModel from './model';
import type { ModelStatic } from './types';

type ModelInternalFacadeCtor<T extends BaseModel> = ModelStatic<T>;

export function getModelRepository<T extends BaseModel>(ModelCtor: ModelInternalFacadeCtor<T>): Repository {
  return RepositoryFactory.getRepository(ModelCtor);
}

export function createModelProxy<T extends BaseModel>(ModelCtor: ModelInternalFacadeCtor<T>, entity: Entity, fields?: FieldSelection<T>): T {
  return hydrateModel<T>(ModelCtor as unknown as ModelStatic<T>, entity, fields);
}
