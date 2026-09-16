// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelCtor } from './types';
import type { FieldSelection, Insertable } from '../repository/types';
import { CreateOperations } from './model_create';

type ModelCreateServiceFacadeCtor<T extends BaseModel> = ModelCtor<T>;

/**
 * Create one record. Runtime may return a field subset when `returnFields` is set;
 * callers see {@link Projected} via BaseModel overloads.
 */
export async function createModel<T extends BaseModel>(
  ModelCtor: ModelCreateServiceFacadeCtor<T>,
  value: Partial<Insertable<T>>,
  returnFields?: FieldSelection<T>
): Promise<T> {
  return await CreateOperations.Create<T>(ModelCtor, value, returnFields);
}

/**
 * Create many records. Runtime may return a field subset when `returnFields` is set;
 * callers see {@link Projected} via BaseModel overloads.
 */
export async function createManyModels<T extends BaseModel>(
  ModelCtor: ModelCreateServiceFacadeCtor<T>,
  values: Partial<Insertable<T>>[],
  returnFields?: FieldSelection<T>
): Promise<T[]> {
  return await CreateOperations.CreateMany<T>(ModelCtor, values, returnFields);
}
