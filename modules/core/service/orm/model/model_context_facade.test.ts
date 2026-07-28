// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getReadonlyCtx, withContext } from '../../runtime/context';
import {
  getInstanceModelCompanyId,
  getInstanceModelCompanyIds,
  getInstanceModelCompanyTimezone,
  getInstanceModelContext,
  getInstanceModelLang,
  getInstanceModelTimezone,
  getInstanceModelUserId,
  getModelCompanyId,
  getModelCompanyIds,
  getModelCompanyTimezone,
  getModelContext,
  getModelLang,
  getModelTimezone,
  getModelUserId,
  withInstanceModelContext,
  withModelContext,
  withInstanceModelCompany,
  withModelCompany,
} from './model_context_facade';

function withTempChoysum<T>(root: any, fn: () => T): T {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  const restore = () => {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  };

  if (root === undefined) delete globalAny.$choysum;
  else globalAny.$choysum = root;

  try {
    const result = fn();
    if (result instanceof Promise) {
      return result.finally(() => restore()) as T;
    }
    restore();
    return result;
  } catch (error) {
    restore();
    throw error;
  }
}

test('model context facade forwards static context accessors to runtime context', async () => {
  const root = {
    request: {
      context: {
        identity: { userId: 'U200' },
        ctx: {
          activeCompanyId: ' C1 ',
          enabledCompanyIds: ['C1', ' C2 '],
          language: ' en-US ',
          timezone: ' UTC ',
          companyTz: ' Asia/Shanghai ',
        },
      },
    },
  };

  await withTempChoysum(root, async () => {
    const runtimeCtx = getReadonlyCtx();

    expect(getModelContext()).toBe(runtimeCtx as any);
    expect(getModelCompanyId()).toBe('C1');
    expect(getModelCompanyIds()).toEqual(['C1', 'C2']);
    expect(getModelLang()).toBe('en-US');
    expect(getModelTimezone()).toBe('UTC');
    expect(getModelCompanyTimezone()).toBe('Asia/Shanghai');
    expect(getModelUserId()).toBe('U200');

    const nested = await withModelContext({ lang: 'fr' }, async () => {
      expect(getModelContext()).toBe(getReadonlyCtx() as any);
      expect(getModelLang()).toBe('fr');
      return Promise.resolve(getModelLang());
    });

    expect(nested).toBe('fr');
    expect(getModelContext()).toBe(runtimeCtx as any);
    expect(getModelLang()).toBe('en-US');
  });
});

test('model context facade reads instance values from constructor statics and delegates withContext untouched', () => {
  const ctorCalls: Array<{ ctx: Record<string, unknown> | (() => Record<string, unknown>); opts: { merge?: boolean } | undefined }> = [];
  const ctor = {
    ctx: { lang: 'ja' },
    companyId: 'C9',
    companyIds: ['C9', 'C10'],
    lang: 'ja',
    tz: 'Asia/Tokyo',
    companyTz: 'Asia/Shanghai',
    userId: 'U900',
    withContext(ctx: Record<string, unknown> | (() => Record<string, unknown>), fn: () => string, opts?: { merge?: boolean }) {
      ctorCalls.push({ ctx, opts });
      return fn();
    },
  };
  const instance = { constructor: ctor } as any;

  expect(getInstanceModelContext(instance)).toBe(ctor.ctx as any);
  expect(getInstanceModelCompanyId(instance)).toBe('C9');
  expect(getInstanceModelCompanyIds(instance)).toEqual(['C9', 'C10']);
  expect(getInstanceModelLang(instance)).toBe('ja');
  expect(getInstanceModelTimezone(instance)).toBe('Asia/Tokyo');
  expect(getInstanceModelCompanyTimezone(instance)).toBe('Asia/Shanghai');
  expect(getInstanceModelUserId(instance)).toBe('U900');

  const source = () => ({ lang: 'ko' });
  const value = withInstanceModelContext(instance, source as any, () => 'ok', { merge: false });

  expect(value).toBe('ok');
  expect(ctorCalls.length).toBe(1);
  expect(ctorCalls[0]?.ctx).toBe(source as any);
  expect(ctorCalls[0]?.opts).toEqual({ merge: false });
});

test('model context facade withCompany overrides and restores company getters; merges lang', async () => {
  const root = {
    request: {
      context: {
        ctx: {
          activeCompanyId: 'OUTER',
          enabledCompanyIds: ['OUTER', 'OTHER'],
          lang: 'en',
        },
      },
    },
  };

  await withTempChoysum(root, async () => {
    expect(getModelCompanyId()).toBe('OUTER');
    expect(getModelCompanyIds()).toEqual(['OUTER', 'OTHER']);

    const nested = await withModelContext({ lang: 'fr' }, async () => {
      return withModelCompany({ activeCompanyId: 'IN', enabledCompanyIds: ['IN', 'X'] }, async () => {
        expect(getModelCompanyId()).toBe('IN');
        expect(getModelCompanyIds()).toEqual(['IN', 'X']);
        expect(getModelLang()).toBe('fr');
        await Promise.resolve();
        return getModelLang();
      });
    });

    expect(nested).toBe('fr');
    expect(getModelCompanyId()).toBe('OUTER');
    expect(getModelCompanyIds()).toEqual(['OUTER', 'OTHER']);
    expect(getModelLang()).toBe('en');
  });
});

test('model context facade instance withCompany delegates to constructor', () => {
  const calls: Array<{ company: unknown }> = [];
  const ctor = {
    ctx: {},
    companyId: undefined,
    companyIds: [],
    lang: undefined,
    tz: undefined,
    companyTz: undefined,
    userId: undefined,
    withContext() {
      return undefined;
    },
    withUser() {
      return undefined;
    },
    withCompany(company: unknown, fn: () => string) {
      calls.push({ company });
      return fn();
    },
    sudo() {
      return undefined;
    },
  };
  const instance = { constructor: ctor } as any;
  const value = withInstanceModelCompany(instance, 'C-DEL', () => 'ok');
  expect(value).toBe('ok');
  expect(calls).toEqual([{ company: 'C-DEL' }]);
});

test('Model.companyTz and Model.tz map display vs company business timezone', async () => {
  const { default: BaseModel } = await import('./model');

  const root = {
    request: {
      context: {
        ctx: {
          tz: 'America/New_York',
          companyTz: 'Asia/Shanghai',
        },
      },
    },
  };

  await withTempChoysum(root, async () => {
    expect(BaseModel.tz).toBe('America/New_York');
    expect(BaseModel.companyTz).toBe('Asia/Shanghai');

    const instance = Object.create(BaseModel.prototype);
    expect(instance.tz).toBe('America/New_York');
    expect(instance.companyTz).toBe('Asia/Shanghai');
  });
});
