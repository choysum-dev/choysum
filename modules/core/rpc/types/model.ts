// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseModel } from '@/core/service/api/model';
import type { FieldSelection } from '@/core/service/api/selection';
import type { Updateable, Insertable } from '@/core/service/api/input';
import type { QueryCondition, OrderBy } from '@/core/service/api/query';
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

export type ModelService<TCtor extends ModelConstructor = ModelConstructor> = {
  [K in ModelServiceMethodKey<TCtor>]: TCtor[K] extends RpcServiceFn ? ClientModelService<TCtor[K]> : never;
};
