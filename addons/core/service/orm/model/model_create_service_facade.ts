// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import type { FieldSelection, Insertable } from '../repository/types';
import { CreateOperations } from './model_create';

type ModelCreateServiceFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T>;

export async function createModel<T extends BaseModel>(
  ModelCtor: ModelCreateServiceFacadeCtor<T>,
  value: Partial<Insertable<T & BaseModel>>,
  returnFields?: FieldSelection<T>
): Promise<T> {
  return (await CreateOperations.Create(ModelCtor, value as Partial<Insertable<T>>, returnFields)) as T;
}

export async function createManyModels<T extends BaseModel>(
  ModelCtor: ModelCreateServiceFacadeCtor<T>,
  values: Partial<Insertable<T & BaseModel>>[],
  returnFields?: FieldSelection<T>
): Promise<T[]> {
  return (await CreateOperations.CreateMany(ModelCtor, values as Array<Partial<Insertable<T>>>, returnFields)) as T[];
}
