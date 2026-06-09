// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import type { DeleteOptions, QueryCondition } from '../repository/types';
import { DeleteOperations } from './model_delete';

type ModelDeleteServiceFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T>;

export async function deleteModels<T extends BaseModel>(
  ModelCtor: ModelDeleteServiceFacadeCtor<T>,
  condition: QueryCondition<T>,
  options?: DeleteOptions
): Promise<number> {
  return await DeleteOperations.Delete(ModelCtor, condition, options);
}

export async function deleteModelById<T extends BaseModel>(ModelCtor: ModelDeleteServiceFacadeCtor<T>, id: string, options?: DeleteOptions): Promise<number> {
  return await DeleteOperations.DeleteById(ModelCtor, id, options);
}
