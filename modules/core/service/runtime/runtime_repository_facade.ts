// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../orm/model/model';
import type { ModelCtor } from '../orm/metadata/field';
import { RepositoryFactory } from '../orm/repository/repository_factory';

export function getRuntimeRepository<T extends BaseModel>(ModelCtor: ModelCtor<T> & typeof BaseModel) {
  return RepositoryFactory.getRepository(ModelCtor);
}
