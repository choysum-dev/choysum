// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guards for HC6: omitting createServiceByModel / dial / pool type
 * args must default to `never` (unusable), not fall back to ModelConstructor.
 *
 * Arrow bodies are type-checked but never invoked.
 */
import { createServiceByModel } from '../rpc/service_factory';
import BaseModel from '../orm/model/model';
import { dial, pool } from '../orm/model/model_pool';

export const hc6DialOmitsCtor: () => never = () => dial('hc6.OmitCtor');
export const hc6PoolOmitsCtor: () => never = () => pool('hc6', 'OmitCtor');
export const hc6FactoryOmitsCtor: () => never = () => createServiceByModel('hc6.OmitCtor');
export const hc6StaticDialOmitsCtor: () => never = () => BaseModel.dial('hc6.OmitCtor');
export const hc6StaticPoolOmitsCtor: () => never = () => BaseModel.pool('OmitCtor');

// @ts-expect-error ModelService must not have a default ctor type argument
export type Hc6BareModelService = import('../../rpc/types').ModelService;

type ExpectTrue<T extends true> = T;
export type Hc6OmittedDialIsNever = ExpectTrue<[ReturnType<typeof hc6DialOmitsCtor>] extends [never] ? true : false>;
