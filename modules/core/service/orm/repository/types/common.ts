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

type IsExactlyAny<T> = 0 extends 1 & T ? true : false;

type IsQuerySupportedValue<V> = IsExactlyAny<V> extends true
  ? false
  : [V] extends [Function]
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
/** Drop private `_` / ambient `$` props (`$sql`, `$search`, …). `$` counts as "uppercase" for `Uppercase<>`. */
type ExcludePrivateOrAmbientProps<K extends string> = K extends `_${string}` | `$${string}` ? never : K;
type OnlyUppercaseProps<K extends string> = K extends `${infer F}${infer R}` ? (F extends Uppercase<F> ? `${F}${R}` : never) : never;
/**
 * Nullable columns (`string | null`, etc.) must remain insertable/updateable.
 * Match query filtering: strip null|undefined before testing supported value shapes.
 */
type ValidInputPropertyType<T, K extends keyof T> = T[K] extends Function
  ? never
  : [NonNil<T[K]>] extends [never]
    ? never
    : [NonNil<T[K]>] extends [InputSupportedTypes]
      ? K
      : never;

export type FilteredQueryProperties<T> = {
  [K in keyof T as ExcludePrivateOrAmbientProps<string & K> extends never
    ? never
    : OnlyUppercaseProps<string & K> extends never
      ? never
      : ValidQueryPropertyType<T, K>]: T[K];
};

export type FilteredInputProperties<T> = {
  [K in keyof T as ExcludePrivateOrAmbientProps<string & K> extends never
    ? never
    : OnlyUppercaseProps<string & K> extends never
      ? never
      : ValidInputPropertyType<T, K>]: T[K];
};

export type Selectable<T> = FilteredQueryProperties<T>;
/** @deprecated Prefer Selectable<T>. Kept for compatibility. */
export type Queryable<T> = Selectable<T>;
