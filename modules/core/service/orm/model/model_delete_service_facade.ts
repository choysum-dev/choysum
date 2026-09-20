// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelCtor } from './types';
import type { SoftDeleteOptions, QueryCondition } from '../repository/types';
import { DeleteOperations } from './model_delete';


export async function deleteModels<T extends BaseModel>(
  ModelCtor: ModelCtor<T>,
  condition: QueryCondition<T>,
  options?: SoftDeleteOptions
): Promise<number> {
  return await DeleteOperations.Delete<T>(ModelCtor, condition, options);
}

export async function deleteModelById<T extends BaseModel>(ModelCtor: ModelCtor<T>, id: string, options?: SoftDeleteOptions): Promise<number> {
  return await DeleteOperations.DeleteById<T>(ModelCtor, id, options);
}
