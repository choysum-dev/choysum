// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { Entity } from '../repository/types/common';

/** Arguments used by the BaseModel factory constructor. */
export type ModelFactoryArgs = [factoryToken: symbol, entity: Entity, fields?: unknown];

/**
 * Constructable model class (factory constructor only).
 * Collection methods historically used this as `this`; prefer {@link ModelCtor} + {@link RowOf}.
 */
export type ModelClass<T extends BaseModel = BaseModel> = new (...args: ModelFactoryArgs) => T;

/**
 * Keys of `typeof BaseModel` that are not construct signatures.
 * Intersecting the raw `typeof` adds a second `new` that collapses polymorphic `this` / InstanceType to BaseModel.
 */
type BaseModelStaticKey = {
  [K in keyof typeof BaseModel]: (typeof BaseModel)[K] extends abstract new (...args: never[]) => unknown ? never : K;
}[keyof typeof BaseModel];

/**
 * Callable / readable static surface of BaseModel without a construct signature.
 * Private statics (`metadata`, `FACTORY_TOKEN`) remain on the real class value; they are not part of this type.
 */
export type BaseModelStatics = Pick<typeof BaseModel, Exclude<BaseModelStaticKey, 'prototype'>>;

/**
 * Runtime model constructor: single factory construct signature plus BaseModel statics.
 * Metadata, facades, hydration, and polymorphic collection `this` use this type.
 */
export type ModelCtor<T extends BaseModel = BaseModel> = ModelClass<T> & BaseModelStatics;

/**
 * Instance row type for a collection ctor.
 * Prefer this over `InstanceType` when the ctor is (or was) intersected with `typeof BaseModel`.
 */
export type RowOf<C> = C extends ModelCtor<infer R>
  ? R
  : C extends abstract new (...args: never[]) => infer R
    ? R
    : BaseModel;
