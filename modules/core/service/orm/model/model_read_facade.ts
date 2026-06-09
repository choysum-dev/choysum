// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { RuntimeModelCtor } from './types';
import { ReadOperations } from './model_read';
import { markPlainShallow } from './model_runtime';
import { createModelProxy } from './model_internal_facade';
import type { Context } from '../../runtime/context';
import type {
  CountOptions,
  FieldSelection,
  GroupBySpec,
  QueryCondition,
  ReadGroupCountOptions,
  ReadGroupOptions,
  ReadGroupResult,
  SearchOptions,
  Selectable,
  SoftDeleteOptions,
} from '../repository/types';
import type { Entity } from '../repository';

type ModelReadFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T> & {
  ctx: Context;
};

function createProxyModel<T extends BaseModel>(ModelCtor: ModelReadFacadeCtor<T>, entity: Entity, fields?: FieldSelection<T>): T {
  return createModelProxy<T>(ModelCtor, entity, fields);
}

function resolveReadGroupTimezone<T extends BaseModel>(ModelCtor: ModelReadFacadeCtor<T>, options: { timezone?: string }) {
  const ctx = ModelCtor.ctx as Context & { timezone?: string; tz?: string };
  return options?.timezone ?? ctx?.timezone ?? ctx?.tz;
}

export async function browseModel<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  id: string,
  fields?: FieldSelection<T>,
  options?: SoftDeleteOptions
): Promise<T> {
  const entity = await ReadOperations.Browse<T>(ModelCtor, id, fields, options);
  return createProxyModel(ModelCtor, entity, fields);
}

export async function browseManyModels<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  ids: string[],
  fields?: (keyof Selectable<T>)[],
  options?: SoftDeleteOptions
): Promise<T[]> {
  if (!ids.length) return [];
  const searchOptions = fields ? ({ fields } as SearchOptions<T>) : ({} as SearchOptions<T>);
  if (options?.withDeleted) searchOptions.withDeleted = true;
  if (options?.onlyDeleted) searchOptions.onlyDeleted = true;
  const results = await ReadOperations.Search<T>(ModelCtor, ['Id', 'in', ids] as QueryCondition<T>, searchOptions);
  return results.map(entity => createProxyModel(ModelCtor, entity, fields as FieldSelection<T> | undefined));
}

export async function searchModels<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  condition: QueryCondition<T> | [] = [],
  options?: SearchOptions<T>
): Promise<T[]> {
  const results = await ReadOperations.Search<T>(ModelCtor, condition, options);
  return results.map(entity => createProxyModel(ModelCtor, entity, options?.fields));
}

export async function countModels<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  condition: QueryCondition<T> | [] = [],
  options?: CountOptions
): Promise<number> {
  return await ReadOperations.Count<T>(ModelCtor, condition, options);
}

export async function readGroupedModels<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
  condition: QueryCondition<T> | [] = [],
  options: ReadGroupOptions<T> = {}
): Promise<ReadGroupResult> {
  const timezone = resolveReadGroupTimezone(ModelCtor, options);
  const resolvedOptions = timezone && !options?.timezone ? { ...options, timezone } : options;
  const result = await ReadOperations.ReadGroup<T>(ModelCtor, groupby, condition, resolvedOptions);
  return markPlainShallow(result) as ReadGroupResult;
}

export async function countGroupedModels<T extends BaseModel>(
  ModelCtor: ModelReadFacadeCtor<T>,
  groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
  condition: QueryCondition<T> | [] = [],
  options: ReadGroupCountOptions<T> = {}
): Promise<number> {
  const timezone = resolveReadGroupTimezone(ModelCtor, options);
  const resolvedOptions = timezone && !options?.timezone ? { ...options, timezone } : options;
  const total = await ReadOperations.ReadGroupCount<T>(ModelCtor, groupby, condition, resolvedOptions);
  return Number(total) || 0;
}
