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

export const hc6DialOmitsCtor = () => dial('hc6.OmitCtor');
export const hc6PoolOmitsCtor = () => pool('hc6', 'OmitCtor');
export const hc6FactoryOmitsCtor = () => createServiceByModel('hc6.OmitCtor');
export const hc6StaticDialOmitsCtor = () => BaseModel.dial('hc6.OmitCtor');
export const hc6StaticPoolOmitsCtor = () => BaseModel.pool('OmitCtor');

// @ts-expect-error ModelService must not have a default ctor type argument
export type Hc6BareModelService = import('../../rpc/types').ModelService;

// @ts-expect-error only model ctor types satisfy the constraint
export type Hc6NonCtorTypeRejected = import('../../rpc/types').ModelService<string>;

type ExpectTrue<T extends true> = T;
export type Hc6OmittedDialIsNever = ExpectTrue<[ReturnType<typeof hc6DialOmitsCtor>] extends [never] ? true : false>;
export type Hc6OmittedPoolIsNever = ExpectTrue<[ReturnType<typeof hc6PoolOmitsCtor>] extends [never] ? true : false>;
export type Hc6OmittedFactoryIsNever = ExpectTrue<[ReturnType<typeof hc6FactoryOmitsCtor>] extends [never] ? true : false>;
export type Hc6OmittedStaticDialIsNever = ExpectTrue<[ReturnType<typeof hc6StaticDialOmitsCtor>] extends [never] ? true : false>;
export type Hc6OmittedStaticPoolIsNever = ExpectTrue<[ReturnType<typeof hc6StaticPoolOmitsCtor>] extends [never] ? true : false>;

// Positive guard: a real ctor type argument must still produce a usable service.
export const hc6DialWithCtor = (fullName: string) => dial<typeof BaseModel>(fullName);
export type Hc6ProvidedDialIsNotNever = ExpectTrue<[ReturnType<typeof hc6DialWithCtor>] extends [never] ? false : true>;
export type Hc6ProvidedDialHasSearch = ExpectTrue<'Search' extends keyof ReturnType<typeof hc6DialWithCtor> ? true : false>;
