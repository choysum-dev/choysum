// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelCtor } from './types';
import type { FieldSelection, QueryCondition, Updateable, UpdateOptions } from '../repository/types';
import { UpdateOperations } from './model_update';

type ModelUpdateServiceFacadeCtor<T extends BaseModel> = ModelCtor<T>;

/**
 * Update matching records. Runtime may return a field subset when `returnFields` is set;
 * callers see {@link Projected} via BaseModel overloads.
 */
export async function updateModels<T extends BaseModel>(
  ModelCtor: ModelUpdateServiceFacadeCtor<T>,
  condition: QueryCondition<T>,
  values: Partial<Updateable<T>>,
  returnFields?: FieldSelection<T>,
  options?: UpdateOptions
): Promise<Partial<T>[]> {
  return await UpdateOperations.Update<T>(ModelCtor, condition, values, returnFields, options);
}

/**
 * Update by Id. Runtime may return a field subset when `returnFields` is set;
 * callers see {@link Projected} via BaseModel overloads.
 */
export async function updateModelById<T extends BaseModel>(
  ModelCtor: ModelUpdateServiceFacadeCtor<T>,
  id: string,
  values: Partial<Updateable<T>>,
  returnFields?: FieldSelection<T>,
  options?: UpdateOptions
): Promise<Partial<T>> {
  return await UpdateOperations.UpdateById<T>(ModelCtor, id, values, returnFields, options);
}
