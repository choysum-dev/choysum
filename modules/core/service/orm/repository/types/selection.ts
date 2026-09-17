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

/** Relation keys selected via object-form {@link DeepRelationSelection} (V1 keeps the key, not nested Pick). */
type RelationSelected<T, F extends FieldSelection<T>> = F[number] extends infer R
  ? R extends DeepRelationSelection<T>
    ? keyof R
    : never
  : never;

/**
 * V1 honest projection: top-level Pick of selected keys.
 * `'*'` (or a selection that includes `'*'`) yields full {@link Selectable}.
 * An empty selection is treated as a full row (ORM default), not `{}`.
 * Nested {@link DeepRelationSelection} entries are not expanded (relation keys stay unprojected).
 */
export type Projected<T, F extends FieldSelection<T>> = F extends readonly []
  ? Selectable<T>
  : Extract<F[number], '*'> extends never
    ? Pick<Selectable<T>, (Extract<F[number], FieldName<T>> | RelationSelected<T, F>) & keyof Selectable<T>>
    : Selectable<T>;

/**
 * When `F` is a concrete {@link FieldSelection} (literal tuple / `fields()`), narrow to
 * {@link Projected}. When `F` is the wide `FieldSelection<T>` itself, return `Partial<T>`
 * (runtime may omit keys; `Projected` would otherwise collapse to full `Selectable` via `'*'`).
 * When `F` is not a selection (e.g. `undefined`) or `any` (common test casts), keep full `T`.
 */
type IsAny<T> = 0 extends 1 & T ? true : false;
export type RowOrProjected<T, F> = IsAny<F> extends true
  ? T
  : F extends FieldSelection<T>
    ? FieldSelection<T> extends F
      ? Partial<T>
      : Projected<T, F>
    : T;

/**
 * Narrow a row to the caller's field selection on a shallow copy (same prototype).
 *
 * Empty selection or `'*'` returns the original row (full-row contract).
 * Nested relation entries keep the top-level key; source row is not mutated.
 * Selected keys are read through the prototype chain so accessors are preserved.
 */
export function projectToSelection<T, F extends FieldSelection<T>>(row: object, selection: F): Projected<T, F> {
  if (selection == null || selection.length === 0 || (selection as readonly unknown[]).includes('*')) {
    return row as Projected<T, F>;
  }
  const keep = new Set<string>();
  for (const entry of selection) {
    if (typeof entry === 'string') keep.add(entry);
    else if (entry && typeof entry === 'object') {
      for (const key of Object.keys(entry as object)) keep.add(key);
    }
  }
  const src = row as Record<string, unknown>;
  const projected = Object.create(Object.getPrototypeOf(row)) as Record<string, unknown>;
  for (const key of keep) {
    if (!(key in src)) continue;
    // defineProperty avoids `__proto__` / setter traps on the projected object.
    Object.defineProperty(projected, key, {
      value: src[key],
      enumerable: true,
      writable: true,
      configurable: true,
    });
  }
  return projected as Projected<T, F>;
}

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
