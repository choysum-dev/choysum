// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelCtor as OrmModelCtor } from '../metadata/field';
import type { Entity } from '../repository/types/common';

export type RuntimeModelCtor<T extends BaseModel = BaseModel> = OrmModelCtor<T> & typeof BaseModel;

/** Constructor this-type used by BaseModel static Create/Update/Search APIs. */
export type BaseModelCtor<T extends BaseModel = BaseModel> = {
  new (factoryToken: Symbol, entity: Entity, fields?: unknown): T;
};

type ModelFactoryArgs = [factoryToken: Symbol, entity: Entity, fields?: unknown];

export type InstantiableModelCtor<T extends BaseModel = BaseModel> = RuntimeModelCtor<T> & { new (...args: ModelFactoryArgs): T };
