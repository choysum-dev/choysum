// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guards for HC6: omitting createServiceByModel / dial / pool type
 * args must default to an unusable result, not fall back to ModelConstructor.
 *
 * Arrow bodies are type-checked but never invoked.
 */
import type { Hc6MissingModelCtorTypeArgument, ModelConstructor } from '../../rpc/types';
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
type IsMissingCtorSentinel<T> = [T] extends [Hc6MissingModelCtorTypeArgument] ? true : false;

export type Hc6OmittedDialIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof hc6DialOmitsCtor>>>;
export type Hc6OmittedFactoryIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof hc6FactoryOmitsCtor>>>;
export type Hc6OmittedStaticDialIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof hc6StaticDialOmitsCtor>>>;
export type Hc6OmittedPoolIsNever = ExpectTrue<[ReturnType<typeof hc6PoolOmitsCtor>] extends [never] ? true : false>;
export type Hc6OmittedStaticPoolIsNever = ExpectTrue<[ReturnType<typeof hc6StaticPoolOmitsCtor>] extends [never] ? true : false>;

// Positive guard: a real ctor type argument must still produce a usable service.
export const hc6DialWithCtor = (fullName: string) => dial<typeof BaseModel>(fullName);
export type Hc6ProvidedDialIsNotSentinel = ExpectTrue<
  [ReturnType<typeof hc6DialWithCtor>] extends [Hc6MissingModelCtorTypeArgument] ? false : true
>;
export type Hc6ProvidedDialHasSearch = ExpectTrue<'Search' extends keyof ReturnType<typeof hc6DialWithCtor> ? true : false>;

// Mapped custom statics must survive dial (Search alone is already on CrudService).
declare const Hc6ProbeCtor: ModelConstructor & {
  Hc6ProbeOp(req: { id: string }): Promise<{ ok: boolean }>;
};
export const hc6DialProbeOp = () => dial<typeof Hc6ProbeCtor>('hc6.Probe');
export type Hc6DialKeepsModelStatics = ExpectTrue<'Hc6ProbeOp' extends keyof ReturnType<typeof hc6DialProbeOp> ? true : false>;

// Omitted dial must not assign into a real service annotation.
type AnyAnnotatedService = { Search(...args: never[]): unknown };
// @ts-expect-error omitted dial is not assignable to an ordinary service shape
export const hc6OmittedDialNotAssignable: AnyAnnotatedService = dial('hc6.OmitCtor');
