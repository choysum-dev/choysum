// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { DeleteResult, UpdateResult, InsertResult, ExpressionBuilder, ExpressionWrapper, SelectQueryBuilder, Compilable } from 'kysely';
export { SimplifyResult } from '../../../infra/database';

import type BaseModel from '../../model/model';
import type { StandardFields } from '../../metadata/field';
import type { ObjectRecord } from '../../../../utils/types';
import type { NonNil } from './shared';

export type SelectResult = ObjectRecord;
/** @deprecated Prefer SelectResult. Kept for compatibility. */
export type Entity = SelectResult;

type IsQuerySupportedValue<V> = [V] extends [Function]
  ? false
  : [V] extends [readonly unknown[]]
    ? false
    : [NonNil<V>] extends [StandardFields]
      ? true
      : [NonNil<V>] extends [BaseModel]
        ? true
        : [NonNil<V>] extends [object]
          ? true
          : false;

type ValidQueryPropertyType<T, K extends keyof T> = IsQuerySupportedValue<T[K]> extends true ? K : never;
type InputSupportedTypes = StandardFields | BaseModel | ObjectRecord | ReadonlyArray<BaseModel> | undefined;
type ExcludeUnderscoreProps<K extends string> = K extends `_${string}` ? never : K;
type OnlyUppercaseProps<K extends string> = K extends `${infer F}${infer R}` ? (F extends Uppercase<F> ? `${F}${R}` : never) : never;
type ValidInputPropertyType<T, K extends keyof T> = T[K] extends Function ? never : T[K] extends InputSupportedTypes ? K : never;

export type FilteredQueryProperties<T> = {
  [K in keyof T as ExcludeUnderscoreProps<string & K> extends never
    ? never
    : OnlyUppercaseProps<string & K> extends never
      ? never
      : ValidQueryPropertyType<T, K>]: T[K];
};

export type FilteredInputProperties<T> = {
  [K in keyof T as ExcludeUnderscoreProps<string & K> extends never
    ? never
    : OnlyUppercaseProps<string & K> extends never
      ? never
      : ValidInputPropertyType<T, K>]: T[K];
};

export type Selectable<T> = FilteredQueryProperties<T>;
/** @deprecated Prefer Selectable<T>. Kept for compatibility. */
export type Queryable<T> = Selectable<T>;
