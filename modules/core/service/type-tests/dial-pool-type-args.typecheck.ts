// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guards: omitting createServiceByModel / dial / pool type args
 * must yield an unusable sentinel, not fall back to ModelConstructor.
 *
 * Arrow bodies are type-checked but never invoked.
 */
import type { MissingModelCtorTypeArgument, ModelConstructor } from '../../rpc/types';
import { createServiceByModel } from '../rpc/service_factory';
import BaseModel from '../orm/model/model';
import { dial, pool } from '../orm/model/model_pool';

export const dialOmitsCtor = () => dial('probe.OmitCtor');
export const poolOmitsCtor = () => pool('probe', 'OmitCtor');
export const factoryOmitsCtor = () => createServiceByModel('probe.OmitCtor');
export const staticDialOmitsCtor = () => BaseModel.dial('probe.OmitCtor');
export const staticPoolOmitsCtor = () => BaseModel.pool('OmitCtor');

// @ts-expect-error ModelService must not have a default ctor type argument
export type BareModelService = import('../../rpc/types').ModelService;

// @ts-expect-error only model ctor types satisfy the constraint
export type NonCtorTypeRejected = import('../../rpc/types').ModelService<string>;

type ExpectTrue<T extends true> = T;
type IsMissingCtorSentinel<T> = [T] extends [MissingModelCtorTypeArgument] ? true : false;

export type OmittedDialIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof dialOmitsCtor>>>;
export type OmittedFactoryIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof factoryOmitsCtor>>>;
export type OmittedStaticDialIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof staticDialOmitsCtor>>>;
export type OmittedPoolIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof poolOmitsCtor>>>;
export type OmittedStaticPoolIsSentinel = ExpectTrue<IsMissingCtorSentinel<ReturnType<typeof staticPoolOmitsCtor>>>;

export const dialWithCtor = (fullName: string) => dial<typeof BaseModel>(fullName);
export type ProvidedDialIsNotSentinel = ExpectTrue<
  [ReturnType<typeof dialWithCtor>] extends [MissingModelCtorTypeArgument] ? false : true
>;
export type ProvidedDialHasSearch = ExpectTrue<'Search' extends keyof ReturnType<typeof dialWithCtor> ? true : false>;

declare const ProbeCtor: ModelConstructor & {
  ProbeOp(req: { id: string }): Promise<{ ok: boolean }>;
};
export const dialProbeOp = () => dial<typeof ProbeCtor>('probe.Model');
export type DialKeepsModelStatics = ExpectTrue<'ProbeOp' extends keyof ReturnType<typeof dialProbeOp> ? true : false>;

export const factoryProbeOp = () => createServiceByModel<typeof ProbeCtor>('probe.Model');
export type FactoryKeepsModelStatics = ExpectTrue<'ProbeOp' extends keyof ReturnType<typeof factoryProbeOp> ? true : false>;

type AnyAnnotatedService = { Search(...args: never[]): unknown };
// @ts-expect-error omitted dial is not assignable to an ordinary service shape
export const omittedDialNotAssignable = (): AnyAnnotatedService => dial('probe.OmitCtor');
