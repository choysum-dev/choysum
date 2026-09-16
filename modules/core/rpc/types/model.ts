// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseModel } from '@/core/service/api/model';
import type { FieldSelection, Projected } from '@/core/service/api/selection';
import type { Updateable, Insertable } from '@/core/service/api/input';
import type { QueryCondition, OrderBy, SearchOptions, SoftDeleteOptions, UpdateOptions } from '@/core/service/api/query';
import type { FieldPath, FieldPathType } from '@/core/service/api/field';

export type { BaseModel, FieldSelection, Updateable, Insertable, QueryCondition, OrderBy, FieldPath, FieldPathType };

export type DeepPartial<T> = T extends object ? { [P in keyof T]?: DeepPartial<T[P]> } : T;

type PrimitiveKeys<T> = {
  [K in keyof T]: T[K] extends Function
    ? never
    : T[K] extends Array<BaseModel>
      ? never
      : T[K] extends BaseModel
        ? never
        : K extends string
          ? K extends `${Uppercase<K[0]>}${string}`
            ? K
            : never
          : never;
}[keyof T];

export type ModelReference<T extends BaseModel = BaseModel> = {
  Id?: string;
} & {
  [K in PrimitiveKeys<T>]?: T[K];
};

export type ClientModelProps<T> = {
  [K in keyof T as T[K] extends Function ? never : K extends string ? (K extends `${Uppercase<K[0]>}${string}` ? K : never) : never]: T[K] extends BaseModel
    ? ModelReference<T[K]>
    : T[K] extends Array<infer Item>
      ? Item extends BaseModel
        ? Array<ModelReference<Item>>
        : T[K]
      : T[K];
};

export type ClientModel<T> = T extends BaseModel
  ? ClientModelProps<T>
  : T extends Array<infer Item>
    ? Item extends BaseModel
      ? Array<ClientModelProps<Item>>
      : Array<ClientModel<Item>>
    : T extends Promise<infer P>
      ? ClientModel<P>
      : T;

export type RpcServiceFn = (...args: never[]) => unknown;
export type ProcessedReturnType<F extends RpcServiceFn> = ClientModel<ReturnType<F>>;
export type ClientModelService<F extends RpcServiceFn> = (...args: Parameters<F>) => Promise<ProcessedReturnType<F>>;

export type ModelConstructor<TModel extends BaseModel = BaseModel> = abstract new (...args: never[]) => TModel;

type AsyncModelMethod = (...args: never[]) => Promise<unknown>;
type ModelServiceMethodKey<TCtor> = {
  [K in keyof TCtor]: K extends string ? (TCtor[K] extends AsyncModelMethod ? (K extends Capitalize<K> ? K : never) : never) : never;
}[keyof TCtor];

type Row<C extends ModelConstructor> = InstanceType<C>;

/**
 * CRUD surface for {@link ModelService}: binds InstanceType and Projected overloads
 * instead of collapsing generic BaseModel static methods via Parameters/ReturnType.
 */
type CrudService<C extends ModelConstructor> = {
  Create: {
    <F extends FieldSelection<Row<C>>>(value: Partial<Insertable<Row<C>>>, returnFields: F): Promise<ClientModel<Projected<Row<C>, F>>>;
    (value: Partial<Insertable<Row<C>>>, returnFields?: FieldSelection<Row<C>>): Promise<ClientModel<Row<C>>>;
  };
  CreateMany: {
    <F extends FieldSelection<Row<C>>>(
      values: Partial<Insertable<Row<C>>>[],
      returnFields: F
    ): Promise<Array<ClientModel<Projected<Row<C>, F>>>>;
    (values: Partial<Insertable<Row<C>>>[], returnFields?: FieldSelection<Row<C>>): Promise<Array<ClientModel<Row<C>>>>;
  };
  Browse: {
    <F extends FieldSelection<Row<C>>>(id: string, fields: F, options?: SoftDeleteOptions): Promise<ClientModel<Projected<Row<C>, F>>>;
    (id: string, fields?: FieldSelection<Row<C>>, options?: SoftDeleteOptions): Promise<ClientModel<Row<C>>>;
  };
  BrowseMany: {
    <F extends FieldSelection<Row<C>>>(ids: string[], fields: F, options?: SoftDeleteOptions): Promise<Array<ClientModel<Projected<Row<C>, F>>>>;
    (ids: string[], fields?: FieldSelection<Row<C>>, options?: SoftDeleteOptions): Promise<Array<ClientModel<Row<C>>>>;
  };
  Search: {
    <F extends FieldSelection<Row<C>>>(
      condition: QueryCondition<Row<C>> | [],
      options: SearchOptions<Row<C>> & { fields: F }
    ): Promise<Array<ClientModel<Projected<Row<C>, F>>>>;
    (condition?: QueryCondition<Row<C>> | [], options?: SearchOptions<Row<C>>): Promise<Array<ClientModel<Row<C>>>>;
  };
  Update: {
    <F extends FieldSelection<Row<C>>>(
      condition: QueryCondition<Row<C>>,
      values: Partial<Updateable<Row<C>>>,
      returnFields: F,
      options?: UpdateOptions
    ): Promise<Array<ClientModel<Projected<Row<C>, F>>>>;
    (
      condition: QueryCondition<Row<C>>,
      values: Partial<Updateable<Row<C>>>,
      returnFields?: FieldSelection<Row<C>>,
      options?: UpdateOptions
    ): Promise<Array<ClientModel<Partial<Row<C>>>>>;
  };
  UpdateById: {
    <F extends FieldSelection<Row<C>>>(
      id: string,
      values: Partial<Updateable<Row<C>>>,
      returnFields: F,
      options?: UpdateOptions
    ): Promise<ClientModel<Projected<Row<C>, F>>>;
    (
      id: string,
      values: Partial<Updateable<Row<C>>>,
      returnFields?: FieldSelection<Row<C>>,
      options?: UpdateOptions
    ): Promise<ClientModel<Partial<Row<C>>>>;
  };
};

/**
 * Sentinel returned by {@link ModelService} / {@link pool} when the model ctor
 * type argument is omitted. Unlike bare `never`, this is not assignable to
 * ordinary service or ctor shapes, so untyped call sites fail closed.
 */
export type MissingModelCtorTypeArgument = {
  readonly __modelCtorTypeArgumentRequired: never;
};

export type ModelService<TCtor extends ModelConstructor> = [TCtor] extends [never]
  ? MissingModelCtorTypeArgument
  : CrudService<TCtor> & {
      [K in Exclude<ModelServiceMethodKey<TCtor>, keyof CrudService<TCtor>>]: TCtor[K] extends RpcServiceFn ? ClientModelService<TCtor[K]> : never;
    };
