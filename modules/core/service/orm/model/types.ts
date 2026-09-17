// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { Entity } from '../repository/types/common';

/** Arguments used by the BaseModel factory constructor. */
export type ModelFactoryArgs = [factoryToken: symbol, entity: Entity, fields?: unknown];

/**
 * Constructable model class (factory constructor only).
 * Collection methods use this as `this` so existing CRUD overrides stay compatible.
 */
export type ModelClass<T extends BaseModel = BaseModel> = new (...args: ModelFactoryArgs) => T;

/**
 * Runtime model constructor: factory construct signature plus BaseModel statics.
 * Metadata, facades, and hydration use this type.
 */
export type ModelCtor<T extends BaseModel = BaseModel> = ModelClass<T> & typeof BaseModel;

/** Instance row type for a collection ctor. Prefer over InstanceType — that collapses on ModelCtor's `typeof BaseModel` intersect. */
export type RowOf<C> = C extends ModelCtor<infer R>
  ? R
  : C extends abstract new (...args: never[]) => infer R
    ? R
    : BaseModel;

