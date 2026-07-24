// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getReadonlyCtx,
  getActiveCompanyId,
  getEnabledCompanyIds,
  getContextLang,
  getContextTimezone,
  getContextCompanyTimezone,
  getUserId,
  withContext,
} from '../../runtime/context';
import type { Context } from '../../runtime/context';

type ModelContextFacadeCtor = {
  ctx: Context;
  companyId: string | undefined;
  companyIds: string[];
  lang: string | undefined;
  tz: string | undefined;
  companyTz: string | undefined;
  userId: string | undefined;
  withContext<R>(ctx: Partial<Context> | (() => Partial<Context>), fn: () => R, opts?: { merge?: boolean }): R;
};

type ModelInstanceLike = {
  constructor: Function;
};

function getModelContextFacadeCtor(instance: ModelInstanceLike): ModelContextFacadeCtor {
  return instance.constructor as unknown as ModelContextFacadeCtor;
}

export function getModelContext(): Context {
  return getReadonlyCtx();
}

export function getModelCompanyId(): string | undefined {
  return getActiveCompanyId();
}

export function getModelCompanyIds(): string[] {
  return getEnabledCompanyIds();
}

export function getModelLang(): string | undefined {
  return getContextLang();
}

export function getModelTimezone(): string | undefined {
  return getContextTimezone();
}

export function getModelCompanyTimezone(): string | undefined {
  return getContextCompanyTimezone();
}

export function getModelUserId(): string | undefined {
  return getUserId();
}

export function withModelContext<R>(ctx: Partial<Context> | (() => Partial<Context>), fn: () => R, opts?: { merge?: boolean }): R {
  return withContext(ctx, fn, opts);
}

export function getInstanceModelContext(instance: ModelInstanceLike): Context {
  return getModelContextFacadeCtor(instance).ctx;
}

export function getInstanceModelCompanyId(instance: ModelInstanceLike): string | undefined {
  return getModelContextFacadeCtor(instance).companyId;
}

export function getInstanceModelCompanyIds(instance: ModelInstanceLike): string[] {
  return getModelContextFacadeCtor(instance).companyIds;
}

export function getInstanceModelLang(instance: ModelInstanceLike): string | undefined {
  return getModelContextFacadeCtor(instance).lang;
}

export function getInstanceModelTimezone(instance: ModelInstanceLike): string | undefined {
  return getModelContextFacadeCtor(instance).tz;
}

export function getInstanceModelCompanyTimezone(instance: ModelInstanceLike): string | undefined {
  return getModelContextFacadeCtor(instance).companyTz;
}

export function getInstanceModelUserId(instance: ModelInstanceLike): string | undefined {
  return getModelContextFacadeCtor(instance).userId;
}

export function withInstanceModelContext<R>(
  instance: ModelInstanceLike,
  ctx: Partial<Context> | (() => Partial<Context>),
  fn: () => R,
  opts?: { merge?: boolean }
): R {
  return getModelContextFacadeCtor(instance).withContext(ctx, fn, opts);
}
