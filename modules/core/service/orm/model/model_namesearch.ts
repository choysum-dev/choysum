// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, QueryCondition, SearchOptions } from '../repository/types';
import { andRepositoryConditions, isEmptyRepositoryCondition } from '../repository/query/condition_layer';
import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';

const DEFAULT_NAME_SEARCH_FIELDS = ['Id', 'DisplayName'] as const;

/**
 * Build NameSearch condition: optional DisplayName like keyword And domain (D2/D4).
 */
export function buildNameSearchCondition<T extends BaseModel>(
  name: string,
  condition?: QueryCondition<T> | []
): QueryCondition<T> | [] {
  const kw = String(name ?? '').trim();
  const parts: Array<BaseQueryCondition | []> = [];

  if (kw) {
    parts.push(['DisplayName', 'like', `%${kw}%`] as BaseQueryCondition);
  }

  if (!isEmptyRepositoryCondition(condition as BaseQueryCondition | [] | undefined)) {
    parts.push(condition as BaseQueryCondition | []);
  }

  return andRepositoryConditions(...parts) as QueryCondition<T> | [];
}

/**
 * Merge Search options with default Id+DisplayName fields when omitted (D2).
 */
export function mergeNameSearchOptions<T extends BaseModel>(options?: SearchOptions<T>): SearchOptions<T> {
  if (options?.fields != null) return options;
  return {
    ...(options || {}),
    fields: [...DEFAULT_NAME_SEARCH_FIELDS] as SearchOptions<T>['fields'],
  };
}

type NameSearchModelCtor<T extends BaseModel> = RuntimeModelCtor<T> & {
  Search: (condition?: QueryCondition<T> | [], options?: SearchOptions<T>) => Promise<T[]>;
};

/**
 * Default NameSearch: DisplayName keyword + domain → Model.Search (D1/D2/D4).
 */
export async function nameSearchModels<T extends BaseModel>(
  ModelCtor: NameSearchModelCtor<T>,
  name: string,
  condition?: QueryCondition<T> | [],
  options?: SearchOptions<T>
): Promise<T[]> {
  const merged = buildNameSearchCondition(name, condition);
  const searchOptions = mergeNameSearchOptions(options);
  return await ModelCtor.Search(merged, searchOptions);
}
