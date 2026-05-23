// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { InstantiableModelCtor } from './types';
import type { Entity, FieldSelection } from '../repository/types';
import { ModelProxyFactory } from '../../runtime/proxy';
import { asRuntimeCarrier } from '../../../utils/object';

type HydrationModelCtor<T extends BaseModel = BaseModel> = InstantiableModelCtor<T>;

/**
 * Hydrates a proxy-wrapped model instance from an entity payload.
 */
export function hydrateModel<T extends BaseModel>(ModelCtor: HydrationModelCtor<T>, entity: Entity, fields?: FieldSelection<T>): T {
  const factoryToken = asRuntimeCarrier(ModelCtor)?.FACTORY_TOKEN as symbol;
  const instance = new ModelCtor(factoryToken, entity, fields) as T;
  return new ModelProxyFactory<T>(instance, entity, fields).create();
}
