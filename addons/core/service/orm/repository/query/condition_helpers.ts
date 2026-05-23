// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, QueryCondition } from '../types';
import { andRepositoryConditions, isEmptyRepositoryCondition } from './condition_layer';

export const isEmptyCondition = isEmptyRepositoryCondition;

export const andAll = andRepositoryConditions;

export function toRepoCondition<T>(condition: BaseQueryCondition | [] | undefined): QueryCondition<T> | [] {
  if (isEmptyCondition(condition)) return [] as [];
  return condition as unknown as QueryCondition<T>;
}
