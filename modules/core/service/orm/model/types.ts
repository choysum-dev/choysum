// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { Entity } from '../repository/types/common';

/** Arguments used by the BaseModel factory constructor. */
export type ModelFactoryArgs = [factoryToken: symbol, entity: Entity, fields?: unknown];

/**
 * Constructable model class (factory constructor only).
 * Use this as the `this` type on BaseModel collection methods.
 */
export type ModelClass<T extends BaseModel = BaseModel> = new (...args: ModelFactoryArgs) => T;

/**
 * Collection-entry / metadata / facade ctor: factory constructor plus BaseModel statics.
 */
export type ModelStatic<T extends BaseModel = BaseModel> = ModelClass<T> & typeof BaseModel;
