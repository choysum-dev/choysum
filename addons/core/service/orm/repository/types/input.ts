// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../model/model';
import type { FilteredInputProperties } from './common';

export type IdRelationItem = string | { Id: string };
export type ModelRelationItem<T extends BaseModel> = T | Partial<Insertable<T>>;
export type RelationItem<T extends BaseModel> = IdRelationItem | ModelRelationItem<T>;

export interface RelationOperations<T extends BaseModel> {
  create?: Array<ModelRelationItem<T>>;
  update?: Array<{ Id: string } & Partial<Omit<Updateable<T>, 'Id'>>>;
  delete?: Array<IdRelationItem>;
  replace?: Array<RelationItem<T>>;
}

type InsertModelInput<T> = T extends BaseModel
  ? Partial<Insertable<T>> | T | null
  : T extends Array<infer E>
    ? E extends BaseModel
      ? Array<E | Partial<Insertable<E>>>
      : T
    : T;

type UpdateModelInput<T> = T extends BaseModel
  ? Partial<Insertable<T>> | T | null
  : T extends Array<infer E>
    ? E extends BaseModel
      ? RelationOperations<E> | Array<E | Partial<Insertable<E>>>
      : T
    : T;

export type Insertable<T> = {
  -readonly [K in keyof FilteredInputProperties<T>]: InsertModelInput<FilteredInputProperties<T>[K]>;
};

export type Updateable<T> = {
  -readonly [K in keyof FilteredInputProperties<T>]: UpdateModelInput<FilteredInputProperties<T>[K]>;
};
