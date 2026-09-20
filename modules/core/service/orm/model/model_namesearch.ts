// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { UntypedQueryCondition, FieldSelection, QueryCondition, SearchOptions } from '../repository/types';
import { andRepositoryConditions, isEmptyRepositoryCondition } from '../repository/query/condition_layer';
import type BaseModel from './model';
import type { ModelCtor } from './types';
const DEFAULT_NAME_SEARCH_FIELDS = ['Id', 'DisplayName'] as const;

/**
 * Build NameSearch condition: optional DisplayName like keyword And domain (D2/D4).
 */
export function buildNameSearchCondition<T extends BaseModel>(
  name: string,
  condition?: QueryCondition<T> | []
): QueryCondition<T> | [] {
  const kw = String(name ?? '').trim();
  const parts: Array<UntypedQueryCondition | []> = [];

  if (kw) {
    parts.push(['DisplayName', 'like', `%${kw}%`] as UntypedQueryCondition);
  }

  if (!isEmptyRepositoryCondition(condition as UntypedQueryCondition | [] | undefined)) {
    parts.push(condition as UntypedQueryCondition | []);
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
    fields: [...DEFAULT_NAME_SEARCH_FIELDS] as unknown as FieldSelection<T>,
  };
}

/**
 * Default NameSearch: DisplayName keyword + domain → Model.Search (D1/D2/D4).
 */
export async function nameSearchModels<T extends BaseModel>(
  ModelCtor: ModelCtor<T>,
  name: string,
  condition?: QueryCondition<T> | [],
  options?: SearchOptions<T>
): Promise<T[]> {
  const merged = buildNameSearchCondition(name, condition);
  const searchOptions = mergeNameSearchOptions(options);
  // Default fields are Id+DisplayName; keep the NameSearch contract as full T[].
  const search = ModelCtor.Search as (
    condition?: QueryCondition<T> | [],
    options?: SearchOptions<T>
  ) => Promise<T[]>;
  return await search.call(ModelCtor, merged, searchOptions);
}
