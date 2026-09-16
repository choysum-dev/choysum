// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from '../../model/model';
import type { Selectable } from './common';

type IsModelLike<T> = T extends BaseModel ? true : T extends (infer U)[] ? (U extends BaseModel ? true : false) : false;
type ModelElem<T> = T extends (infer U)[] ? U : T;

type RelationKeys<T> = {
  [K in keyof T]: IsModelLike<T[K]> extends true ? K : never;
}[keyof T];

export type DeepRelationSelection<T> = {
  [K in RelationKeys<T>]?: Array<keyof Selectable<ModelElem<T[K]>> | DeepRelationSelection<ModelElem<T[K]>>>;
};

/**
 * Field selection for Browse/Search/Create return projections.
 * ReadonlyArray so `as const` / `fields()` helpers keep literal keys for {@link Projected}.
 */
export type FieldSelection<T> = ReadonlyArray<'*' | keyof Selectable<T> | RelationKeys<T> | DeepRelationSelection<T>>;

/** Scalar / relation key names that can appear in a field selection (excludes `*` and nested objects). */
type FieldName<T> = Exclude<FieldSelection<T>[number], object | '*'>;

/**
 * V1 honest projection: top-level Pick of selected keys.
 * `'*'` (or a selection that includes `'*'`) yields full {@link Selectable}.
 * An empty selection is treated as a full row (ORM default), not `{}`.
 * Nested {@link DeepRelationSelection} entries are not expanded (relation keys stay unprojected).
 */
export type Projected<T, F extends FieldSelection<T>> = F extends readonly []
  ? Selectable<T>
  : Extract<F[number], '*'> extends never
    ? Pick<Selectable<T>, Extract<F[number], FieldName<T>> & keyof Selectable<T>>
    : Selectable<T>;

/**
 * Build a field-selection tuple that preserves literal keys for {@link Projected} inference.
 *
 * @example
 * ```ts
 * await Model.Search(cond, { fields: fields<Model>()('Id', 'Name') });
 * ```
 */
export function fields<T>() {
  return <const F extends FieldSelection<T>>(...names: F): F => names;
}
