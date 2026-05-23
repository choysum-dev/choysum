// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import type { FieldSelection, QueryCondition, Updateable, UpdateOptions } from '../repository/types';
import { UpdateOperations } from './model_update';

type ModelUpdateServiceFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T>;

export async function updateModels<T extends BaseModel>(
  ModelCtor: ModelUpdateServiceFacadeCtor<T>,
  condition: QueryCondition<T>,
  values: Partial<Updateable<T & BaseModel>>,
  returnFields?: FieldSelection<T>,
  options?: UpdateOptions
): Promise<Partial<T>[]> {
  return await UpdateOperations.Update<T>(ModelCtor, condition, values as Partial<Updateable<T>>, returnFields, options);
}

export async function updateModelById<T extends BaseModel>(
  ModelCtor: ModelUpdateServiceFacadeCtor<T>,
  id: string,
  values: Partial<Updateable<T & BaseModel>>,
  returnFields?: FieldSelection<T>,
  options?: UpdateOptions
): Promise<Partial<T>> {
  return await UpdateOperations.UpdateById<T>(ModelCtor, id, values as Partial<Updateable<T>>, returnFields, options);
}
